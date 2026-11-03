package forwarding

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// directTestDialer dials TCP directly on localhost for integration tests.
type directTestDialer struct{}

func (d *directTestDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	var netDialer net.Dialer
	return netDialer.DialContext(ctx, network, addr)
}

// getFreeLocalPort returns an available port on 127.0.0.1.
func getFreeLocalPort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// startEchoServer launches a local TCP echo server on an ephemeral port.
func startEchoServer(t *testing.T) (net.Listener, int) {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				_, _ = io.Copy(c, c)
			}(conn)
		}
	}()

	port := l.Addr().(*net.TCPAddr).Port
	return l, port
}

func TestTCPProxy_DataForwarding(t *testing.T) {
	echoListener, targetPort := startEchoServer(t)
	defer echoListener.Close()

	localPort := getFreeLocalPort(t)
	proxy := NewTCPProxy("test-proxy", localPort, "127.0.0.1", targetPort, &directTestDialer{}, nil)

	err := proxy.Start()
	require.NoError(t, err)
	defer proxy.Stop()

	require.True(t, proxy.IsRunning())

	// Connect to local proxy port
	proxyAddr := fmt.Sprintf("127.0.0.1:%d", localPort)
	conn, err := net.DialTimeout("tcp", proxyAddr, 2*time.Second)
	require.NoError(t, err)
	defer conn.Close()

	// Send test payload
	testPayload := []byte("Hello, TCP Proxy Forwarding Test 12345!")
	_, err = conn.Write(testPayload)
	require.NoError(t, err)

	buf := make([]byte, len(testPayload))
	_, err = io.ReadFull(conn, buf)
	require.NoError(t, err)
	assert.Equal(t, testPayload, buf, "Echo response through proxy must match sent payload")
}

func TestTCPProxy_MultipleConnections(t *testing.T) {
	echoListener, targetPort := startEchoServer(t)
	defer echoListener.Close()

	localPort := getFreeLocalPort(t)
	proxy := NewTCPProxy("concurrent-proxy", localPort, "127.0.0.1", targetPort, &directTestDialer{}, nil)

	require.NoError(t, proxy.Start())
	defer proxy.Stop()

	proxyAddr := fmt.Sprintf("127.0.0.1:%d", localPort)
	const numClients = 25
	var wg sync.WaitGroup
	wg.Add(numClients)

	for i := 0; i < numClients; i++ {
		go func(clientID int) {
			defer wg.Done()

			conn, err := net.DialTimeout("tcp", proxyAddr, 3*time.Second)
			if !assert.NoError(t, err) {
				return
			}
			defer conn.Close()

			payload := fmt.Sprintf("client-%d-message-timestamp-%d", clientID, time.Now().UnixNano())
			_, err = conn.Write([]byte(payload))
			assert.NoError(t, err)

			buf := make([]byte, len(payload))
			_, err = io.ReadFull(conn, buf)
			assert.NoError(t, err)
			assert.Equal(t, payload, string(buf))
		}(i)
	}

	wg.Wait()
}

func TestTCPProxy_TargetDisconnect(t *testing.T) {
	// A target server that accepts connection, writes a banner, and abruptly disconnects
	targetListener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer targetListener.Close()

	targetPort := targetListener.Addr().(*net.TCPAddr).Port

	go func() {
		conn, err := targetListener.Accept()
		if err != nil {
			return
		}
		// Write banner then immediately close to simulate destination drop
		_, _ = conn.Write([]byte("SERVER_CLOSING"))
		time.Sleep(50 * time.Millisecond)
		_ = conn.Close()
	}()

	localPort := getFreeLocalPort(t)
	proxy := NewTCPProxy("disconnect-proxy", localPort, "127.0.0.1", targetPort, &directTestDialer{}, nil)

	require.NoError(t, proxy.Start())
	defer proxy.Stop()

	proxyAddr := fmt.Sprintf("127.0.0.1:%d", localPort)
	clientConn, err := net.DialTimeout("tcp", proxyAddr, 2*time.Second)
	require.NoError(t, err)
	defer clientConn.Close()

	// Read until EOF
	received, err := io.ReadAll(clientConn)
	assert.True(t, bytes.Contains(received, []byte("SERVER_CLOSING")))
	// Client connection must have terminated cleanly without hanging
	assert.NoError(t, err)
}

func TestTCPProxy_GracefulStop(t *testing.T) {
	echoListener, targetPort := startEchoServer(t)
	defer echoListener.Close()

	localPort := getFreeLocalPort(t)
	proxy := NewTCPProxy("stop-proxy", localPort, "127.0.0.1", targetPort, &directTestDialer{}, nil)

	require.NoError(t, proxy.Start())
	assert.True(t, proxy.IsRunning())

	// Stop proxy
	require.NoError(t, proxy.Stop())
	assert.False(t, proxy.IsRunning())

	// Calling Stop again should be safe and idempotent
	assert.NoError(t, proxy.Stop())

	// Verifying new connections are rejected immediately
	proxyAddr := fmt.Sprintf("127.0.0.1:%d", localPort)
	_, err := net.DialTimeout("tcp", proxyAddr, 200*time.Millisecond)
	assert.Error(t, err, "Connection should fail after proxy is stopped")
}
