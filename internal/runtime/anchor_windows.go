//go:build windows

package runtime

// This file implements the persistent "anchor" HTTP connection to the tailscaled
// LocalAPI over the Windows Named Pipe.
//
// Design rationale:
// In non-service mode, tailscaled is lifecycle-bound to its Named Pipe clients.
// When the LAST client disconnects, the daemon resets the Tailscale session to
// NoState ("client disconnected... disconnecting Tailscale").  Every short-lived
// tailscale.exe CLI invocation (status, up, …) opens the pipe, does its work,
// and closes it.  When it exits and no other client holds the pipe, the daemon
// resets — even if it had just reached the Running state.
//
// The solution: maintain one persistent HTTP connection to the LocalAPI endpoint
// /localapi/v0/watch-ipn-bus, which streams IPN events indefinitely.  This
// "anchor" keeps the daemon from seeing itself as client-less, so all CLI calls
// work on top of it without disrupting the daemon state.
//
// readyCh: callers that need to block until the anchor is live pass a non-nil
// channel; it is signalled (sent struct{}{}) as soon as the first HTTP 200
// response is received from the daemon.

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/Microsoft/go-winio"
	"golang.org/x/sys/windows"
)

// namedPipeDialer routes HTTP requests through the Windows Named Pipe.
type namedPipeDialer struct{ pipePath string }

func (d *namedPipeDialer) DialContext(ctx context.Context, _, _ string) (net.Conn, error) {
	// We MUST use DialPipeAccessImpLevel with PipeImpLevelIdentification.
	// Otherwise, tailscaled's safesocket server fails to impersonate our connection
	// to retrieve the user's access token, resulting in access denied (403 Forbidden)
	// or immediate connection drops. This matches safesocket.connect() in tailscale.
	return winio.DialPipeAccessImpLevel(ctx, d.pipePath, windows.GENERIC_READ|windows.GENERIC_WRITE, winio.PipeImpLevelIdentification)
}

// newPipeHTTPClient returns an http.Client that routes all requests through the
// Named Pipe at pipePath.  Keep-alives are enabled so the underlying pipe
// connection is reused across retries within the same transport.
func newPipeHTTPClient(pipePath string) *http.Client {
	dialer := &namedPipeDialer{pipePath: pipePath}
	return &http.Client{
		Transport: &http.Transport{
			DialContext:           dialer.DialContext,
			DisableKeepAlives:     false,
			MaxIdleConns:          1,
			MaxIdleConnsPerHost:   1,
			IdleConnTimeout:       0,
			ResponseHeaderTimeout: 5 * time.Second,
		},
		Timeout: 0, // no overall timeout — watch-ipn-bus is an infinite stream
	}
}

// runAnchorConnection maintains a persistent GET /localapi/v0/watch-ipn-bus
// stream, reconnecting with exponential back-off if the connection drops.
// It blocks until ctx is cancelled; always call it in a goroutine.
//
// readyCh: if non-nil, a struct{}{} is sent to it the moment the first
// successful HTTP connection is established (response headers received).
// Only one signal is ever sent; pass a buffered channel of size 1.
func runAnchorConnection(ctx context.Context, socketPath string,
	logFn func(format string, args ...interface{}),
	readyCh chan<- struct{},
) {
	backoff := 300 * time.Millisecond
	signalled := false

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		connected, err := connectAndDrain(ctx, socketPath, func() {
			// Called once when the HTTP response headers arrive (daemon is ready).
			if !signalled && readyCh != nil {
				select {
				case readyCh <- struct{}{}:
				default:
				}
				signalled = true
			}
		})

		if !connected {
			// Connection was never established — back off before retrying.
			if ctx.Err() != nil {
				return
			}
			if logFn != nil {
				logFn("[anchor] could not connect: %v — retrying in %v", err, backoff)
			}
		} else if err != nil {
			// Connection was established but then dropped.
			if ctx.Err() != nil {
				return
			}
			if logFn != nil {
				logFn("[anchor] stream dropped: %v — reconnecting in %v", err, backoff)
			}
		} else {
			// Clean close (ctx cancelled).
			return
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff *= 2
		if backoff > 10*time.Second {
			backoff = 10 * time.Second
		}
	}
}

// connectAndDrain dials the daemon once and reads from watch-ipn-bus until ctx
// is cancelled or the stream closes.
//
// Returns (connected bool, err error):
//   - connected=false, err≠nil  → dial or HTTP error before response headers
//   - connected=true,  err=nil  → clean close (ctx cancelled)
//   - connected=true,  err≠nil  → stream closed unexpectedly after connection
func connectAndDrain(ctx context.Context, socketPath string, onReady func()) (bool, error) {
	client := newPipeHTTPClient(socketPath)

	// mask=3 = NotifyWatchEngineUpdates(1) | NotifyInitialState(2)
	//
	// NotifyInitialState: causes the daemon to immediately send a JSON event
	// with the current IPN state, so our ReadString('\n') below receives data
	// right away and we can confirm the connection is alive.
	//
	// NotifyWatchEngineUpdates: causes the daemon to start pollRequestEngineStatus
	// in a background goroutine, which sends a Notify event roughly every 2 seconds.
	// This is essential: without it, no events are ever placed in the channel and
	// the HTTP response body stays silent — triggering our transport's idle-read
	// deadline and producing the "stream closed by daemon (EOF)" loop.
	//
	// mask=0 must NOT be used: WatchNotificationsAs skips both the initial-state
	// snapshot and the engine-status polling when no bits are set, so the channel
	// never receives any events and the connection closes immediately.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"http://local-tailscaled.sock/localapi/v0/watch-ipn-bus?mask=3", nil)
	if err != nil {
		return false, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Connection", "keep-alive")

	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return false, nil // context cancelled before connect
		}
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return false, fmt.Errorf("HTTP %s: %s", resp.Status, strings.TrimSpace(string(bodyBytes)))
	}

	// Signal the caller that we are now anchored.
	if onReady != nil {
		onReady()
	}

	// Drain the streaming body (newline-delimited JSON events) so the pipe
	// window never fills up, keeping the connection alive.
	reader := bufio.NewReaderSize(resp.Body, 4096)
	for {
		if ctx.Err() != nil {
			return true, nil // clean exit
		}
		_, err := reader.ReadString('\n')
		if err != nil {
			if ctx.Err() != nil {
				return true, nil
			}
			if err == io.EOF {
				return true, fmt.Errorf("stream closed by daemon (EOF)")
			}
			if strings.Contains(err.Error(), "context canceled") ||
				strings.Contains(err.Error(), "context deadline exceeded") {
				return true, nil
			}
			return true, fmt.Errorf("read stream: %w", err)
		}
	}
}
