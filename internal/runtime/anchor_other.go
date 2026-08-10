//go:build !windows

package runtime

import "context"

// runAnchorConnection is a no-op on non-Windows platforms.
// On Linux/macOS, tailscaled does not exit when clients disconnect — its
// lifecycle is managed by systemd or launchd, so no persistent anchor is needed.
// readyCh is signalled immediately so callers that wait on it are not blocked.
func runAnchorConnection(ctx context.Context, socketPath string,
	logFn func(format string, args ...interface{}),
	readyCh chan<- struct{},
) {
	if readyCh != nil {
		select {
		case readyCh <- struct{}{}:
		default:
		}
	}
	<-ctx.Done()
}
