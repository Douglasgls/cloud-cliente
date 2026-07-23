package runtime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"

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
	logger         *logger.Logger
	isPrepared     bool
	runtimeDir     string
	stateDir       string
	tailscalePath  string
	tailscaledPath string
	socketPath     string
	socks5Addr     string
	daemonPID      int
}

func NewManager(log *logger.Logger) *Manager {
	return &Manager{
		logger: log,
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

	socketExists := false
	if _, err := os.Stat(m.socketPath); err == nil {
		socketExists = true
	}

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
