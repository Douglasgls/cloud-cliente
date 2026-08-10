package runtime

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"cloud-client/pkg/logger"
)

var (
	ErrUnsupportedOS   = errors.New("unsupported operating system")
	ErrRuntimeNotReady = errors.New("runtime is not ready")
	ErrRuntimeNotFound = errors.New("tailscale runtime binary not found")
)

type RuntimeManager interface {
	Prepare() error
	IsPrepared() bool
	EnsureDaemonRunning(ctx context.Context) error
	PrintDebugInfo(ctx context.Context) error
	StopDaemon() error

	RuntimeDir() string
	StateDir() string
	SocketPath() string
	Socks5Addr() string
	TailscalePath() string
	TailscaledPath() string
	DaemonPID() int
	IsDaemonRunning() bool
}

type Manager struct {
	mu             sync.Mutex
	logger         *logger.Logger
	isPrepared     bool
	runtimeDir     string
	stateDir       string
	tailscalePath  string
	tailscaledPath string
	socketPath     string
	socks5Addr     string
	daemonPID      int
	daemonCmd      *exec.Cmd
	cancelAnchor   context.CancelFunc // stops the persistent pipe-anchor goroutine
}

func NewManager(log *logger.Logger) *Manager {
	return &Manager{
		logger:     log,
		socks5Addr: "127.0.0.1:1055",
	}
}

func (m *Manager) IsPrepared() bool {
	return m.isPrepared
}

func (m *Manager) RuntimeDir() string {
	return m.runtimeDir
}

func (m *Manager) StateDir() string {
	return m.stateDir
}

func (m *Manager) TailscalePath() string {
	return m.tailscalePath
}

func (m *Manager) TailscaledPath() string {
	return m.tailscaledPath
}

func (m *Manager) SocketPath() string {
	return m.socketPath
}

func (m *Manager) Socks5Addr() string {
	return m.socks5Addr
}

func (m *Manager) DaemonPID() int {
	return m.daemonPID
}

func (m *Manager) IsDaemonRunning() bool {
	return isSocketAlive(context.Background(), m.socketPath)
}

func (m *Manager) Prepare() error {
	if m.isPrepared {
		return nil
	}

	osName := runtime.GOOS
	if m.logger != nil {
		m.logger.Info("Runtime detected: %s", osName)
		m.logger.Info("Preparing runtime...")
	}

	switch osName {
	case "linux":
		if err := m.prepareLinux(); err != nil {
			return fmt.Errorf("failed to prepare linux runtime: %w", err)
		}
	case "windows":
		if err := m.prepareWindows(); err != nil {
			return fmt.Errorf("failed to prepare windows runtime: %w", err)
		}
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedOS, osName)
	}

	m.isPrepared = true
	if m.logger != nil {
		m.logger.Info("Runtime ready.")
	}

	return nil
}

func (m *Manager) PrintDebugInfo(ctx context.Context) error {
	if m.logger == nil {
		m.logger = logger.New(true)
	}

	socketExists := isSocketAlive(ctx, m.socketPath)

	m.logger.Info("Runtime:")
	m.logger.Info("  RuntimeDir:   %s", m.runtimeDir)
	m.logger.Info("  StateDir:     %s", m.stateDir)
	m.logger.Info("  Socket:       %s", m.socketPath)
	m.logger.Info("  tailscale:    %s", m.tailscalePath)
	m.logger.Info("  tailscaled:   %s", m.tailscaledPath)
	m.logger.Info("")
	m.logger.Info("Daemon:")
	m.logger.Info("  PID:          %d", m.daemonPID)
	m.logger.Info("  Running:      %v", m.IsDaemonRunning())
	m.logger.Info("  Socket Exists:%v", socketExists)

	return nil
}

// startAnchorAndWait cancels any previous anchor goroutine, launches a fresh
// one, and blocks until the anchor signals it is connected (or until timeout or
// ctx is cancelled).  Pass timeout=0 to fire-and-forget (non-blocking).
func (m *Manager) startAnchorAndWait(ctx context.Context, timeout time.Duration) error {
	m.mu.Lock()
	if m.cancelAnchor != nil {
		m.cancelAnchor()
	}
	anchorCtx, cancelAnchor := context.WithCancel(context.Background())
	m.cancelAnchor = cancelAnchor
	socketPath := m.socketPath
	var logFn func(string, ...interface{})
	if m.logger != nil {
		logFn = func(format string, args ...interface{}) {
			m.logger.Info(format, args...)
		}
	}
	m.mu.Unlock()

	readyCh := make(chan struct{}, 1)
	go runAnchorConnection(anchorCtx, socketPath, logFn, readyCh)

	if timeout <= 0 {
		return nil // fire-and-forget
	}

	select {
	case <-readyCh:
		if logFn != nil {
			logFn("[anchor] persistent connection established.")
		}
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("anchor timed out after %v — daemon may not be fully ready", timeout)
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *Manager) StopDaemon() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Stop the persistent anchor connection first so the daemon can clean up
	// gracefully without seeing a surprise disconnection.
	if m.cancelAnchor != nil {
		m.cancelAnchor()
		m.cancelAnchor = nil
	}

	if m.daemonCmd != nil && m.daemonCmd.Process != nil {
		_ = m.daemonCmd.Process.Kill()
		m.daemonCmd = nil
		m.daemonPID = 0
	}
	return nil
}
