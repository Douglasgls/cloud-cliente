package tailscale

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRuntimeManager struct {
	tailscalePath  string
	socketPath     string
	socks5Addr     string
	isDaemonRun    bool
	daemonPID      int
	isPrepared     bool
}

func (m *mockRuntimeManager) Prepare() error                      { return nil }
func (m *mockRuntimeManager) IsPrepared() bool                   { return m.isPrepared }
func (m *mockRuntimeManager) EnsureDaemonRunning(ctx context.Context) error { return nil }
func (m *mockRuntimeManager) PrintDebugInfo(ctx context.Context) error      { return nil }
func (m *mockRuntimeManager) StopDaemon() error                  { return nil }
func (m *mockRuntimeManager) RuntimeDir() string                 { return "/mock/runtime" }
func (m *mockRuntimeManager) StateDir() string                   { return "/mock/state" }
func (m *mockRuntimeManager) SocketPath() string                 { return m.socketPath }
func (m *mockRuntimeManager) Socks5Addr() string                 { return m.socks5Addr }
func (m *mockRuntimeManager) TailscalePath() string              { return m.tailscalePath }
func (m *mockRuntimeManager) TailscaledPath() string             { return "/mock/bin/tailscaled" }
func (m *mockRuntimeManager) DaemonPID() int                     { return m.daemonPID }
func (m *mockRuntimeManager) IsDaemonRunning() bool              { return m.isDaemonRun }

func TestTailscaleService_Up(t *testing.T) {
	mockRt := &mockRuntimeManager{
		tailscalePath: "/usr/bin/tailscale",
		socketPath:    "/tmp/tailscaled.sock",
		isPrepared:    true,
	}

	t.Run("Successful Up with all parameters", func(t *testing.T) {
		var capturedBinary string
		var capturedArgs []string

		service := NewServiceWithExecutor(mockRt, func(ctx context.Context, binary string, args ...string) (string, string, error) {
			capturedBinary = binary
			capturedArgs = args
			return "Success", "", nil
		})

		ctx := context.Background()
		err := service.Up(ctx, "https://headscale.example.com", "tskey-auth-123", "my-client:laptop\nsecond-line")
		require.NoError(t, err)

		assert.Equal(t, "/usr/bin/tailscale", capturedBinary)
		// Socket argument included
		assert.Contains(t, capturedArgs, "--socket=/tmp/tailscaled.sock")
		assert.Contains(t, capturedArgs, "up")
		assert.Contains(t, capturedArgs, "--reset")
		assert.Contains(t, capturedArgs, "--accept-dns=false")
		assert.Contains(t, capturedArgs, "--force-reauth")
		assert.Contains(t, capturedArgs, "--login-server=https://headscale.example.com")
		assert.Contains(t, capturedArgs, "--authkey=tskey-auth-123")
		// Hostname sanitized (colon replaced with hyphen, newlines trimmed)
		assert.Contains(t, capturedArgs, "--hostname=my-client-laptop")
	})

	t.Run("Up failure maps to ErrExecutionFailed", func(t *testing.T) {
		service := NewServiceWithExecutor(mockRt, func(ctx context.Context, binary string, args ...string) (string, string, error) {
			return "", "invalid pre-auth key provided", errors.New("exit status 1")
		})

		err := service.Up(context.Background(), "https://headscale.example.com", "invalid-key", "my-client")
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrExecutionFailed))
		assert.Contains(t, err.Error(), "invalid pre-auth key provided")
	})

	t.Run("Tailscale binary not found error", func(t *testing.T) {
		service := NewServiceWithExecutor(mockRt, func(ctx context.Context, binary string, args ...string) (string, string, error) {
			return "", "", exec.ErrNotFound
		})

		err := service.Up(context.Background(), "https://headscale.example.com", "key", "host")
		assert.ErrorIs(t, err, ErrTailscaleNotInstalled)
	})
}

func TestTailscaleService_LifecycleCommands(t *testing.T) {
	mockRt := &mockRuntimeManager{
		tailscalePath: "/usr/bin/tailscale",
		socketPath:    "/tmp/tailscaled.sock",
	}

	t.Run("Down execution", func(t *testing.T) {
		var executedArgs []string
		service := NewServiceWithExecutor(mockRt, func(ctx context.Context, binary string, args ...string) (string, string, error) {
			executedArgs = args
			return "", "", nil
		})

		err := service.Down(context.Background())
		require.NoError(t, err)
		assert.Contains(t, executedArgs, "down")
	})

	t.Run("Logout execution", func(t *testing.T) {
		var executedArgs []string
		service := NewServiceWithExecutor(mockRt, func(ctx context.Context, binary string, args ...string) (string, string, error) {
			executedArgs = args
			return "", "", nil
		})

		err := service.Logout(context.Background())
		require.NoError(t, err)
		assert.Contains(t, executedArgs, "logout")
	})

	t.Run("Status query success and error", func(t *testing.T) {
		service := NewServiceWithExecutor(mockRt, func(ctx context.Context, binary string, args ...string) (string, string, error) {
			if strings.Contains(strings.Join(args, " "), "status") {
				return "100.64.0.2  client-node  user@  linux  active\n", "", nil
			}
			return "", "", errors.New("unknown")
		})

		status, err := service.Status(context.Background())
		require.NoError(t, err)
		assert.Contains(t, status, "100.64.0.2")
	})

	t.Run("IP command returns sanitized address", func(t *testing.T) {
		service := NewServiceWithExecutor(mockRt, func(ctx context.Context, binary string, args ...string) (string, string, error) {
			return "100.64.0.15\n", "", nil
		})

		ip, err := service.IP(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "100.64.0.15", ip)
	})

	t.Run("Version command", func(t *testing.T) {
		service := NewServiceWithExecutor(mockRt, func(ctx context.Context, binary string, args ...string) (string, string, error) {
			return "1.60.0\n", "", nil
		})

		ver, err := service.Version(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "1.60.0", ver)
	})

	t.Run("Service returns ErrTailscaleNotInstalled when path is empty", func(t *testing.T) {
		emptyRt := &mockRuntimeManager{tailscalePath: ""}
		service := NewService(emptyRt)

		_, _, err := service.runCommand(context.Background(), "status")
		assert.ErrorIs(t, err, ErrTailscaleNotInstalled)
	})
}
