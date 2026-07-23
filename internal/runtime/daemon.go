package runtime

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func (m *Manager) EnsureDaemonRunning(ctx context.Context) error {
	if !m.isPrepared {
		if err := m.Prepare(); err != nil {
			return err
		}
	}

	if m.logger != nil {
		m.logger.Info("Runtime directory:")
		m.logger.Info("%s", m.runtimeDir)
		m.logger.Info("")
		m.logger.Info("tailscale:")
		m.logger.Info("%s", m.tailscalePath)
		m.logger.Info("")
		m.logger.Info("tailscaled:")
		m.logger.Info("%s", m.tailscaledPath)
		m.logger.Info("")
		m.logger.Info("State directory:")
		m.logger.Info("%s", m.stateDir)
		m.logger.Info("")
		m.logger.Info("Socket:")
		m.logger.Info("%s", m.socketPath)
		m.logger.Info("")
	}

	// Test if socket is already responsive
	if m.validateDaemon(ctx) == nil {
		if m.logger != nil {
			m.logger.Info("Daemon is already running and responsive.")
		}
		return nil
	}

	if m.tailscaledPath == "" {
		return fmt.Errorf("%w: tailscaled binary path is empty", ErrRuntimeNotFound)
	}

	if err := os.MkdirAll(m.stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory %s: %w", m.stateDir, err)
	}

	logFilePath := filepath.Join(m.stateDir, "tailscaled.log")
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create daemon log file: %w", err)
	}

	if m.socks5Addr == "" {
		m.socks5Addr = "127.0.0.1:1055"
	}

	args := []string{
		"--statedir=" + m.stateDir,
		"--socket=" + m.socketPath,
		"--socks5-server=" + m.socks5Addr,
	}

	if runtime.GOOS != "windows" {
		args = append(args, "--tun=userspace-networking")
	}

	cmdStr := fmt.Sprintf("%s \\\n    %s", m.tailscaledPath, strings.Join(args, " \\\n    "))

	if m.logger != nil {
		m.logger.Info("Starting tailscaled...")
		m.logger.Info("")
		m.logger.Info("Command:")
		m.logger.Info("%s", cmdStr)
		m.logger.Info("")
	}

	cmd := exec.Command(m.tailscaledPath, args...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("failed to start tailscaled background process: %w", err)
	}

	m.daemonPID = cmd.Process.Pid

	if m.logger != nil {
		m.logger.Info("PID:")
		m.logger.Info("%d", m.daemonPID)
		m.logger.Info("")
		m.logger.Info("Waiting daemon...")
	}

	// Poll and validate daemon
	ticker := time.NewTicker(150 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.After(6 * time.Second)

	for {
		select {
		case <-ctx.Done():
			logFile.Close()
			return ctx.Err()
		case <-timeout:
			logFile.Close()
			logContent, _ := os.ReadFile(logFilePath)
			if m.logger != nil {
				m.logger.Error("Daemon failed to start within timeout. Log output:\n%s", string(logContent))
			}
			return fmt.Errorf("timeout waiting for tailscaled daemon to respond (logs in %s)", logFilePath)
		case <-ticker.C:
			if err := m.validateDaemon(ctx); err == nil {
				logFile.Close()
				if m.logger != nil {
					m.logger.Info("Daemon started successfully.")
					m.logger.Info("")
				}
				return nil
			}
		}
	}
}

func (m *Manager) validateDaemon(ctx context.Context) error {
	if m.socketPath == "" {
		return errors.New("socket path is empty")
	}

	// Check UNIX socket connectivity directly first
	if runtime.GOOS != "windows" {
		var d net.Dialer
		dialCtx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
		conn, err := d.DialContext(dialCtx, "unix", m.socketPath)
		cancel()
		if err != nil {
			return fmt.Errorf("socket not reachable: %w", err)
		}
		conn.Close()
	}

	// Run tailscale --socket=<SocketPath> status to verify end-to-end responsiveness
	cmd := exec.CommandContext(ctx, m.tailscalePath, "--socket="+m.socketPath, "status")
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	_ = cmd.Run()
	output := stdoutBuf.String() + stderrBuf.String()

	if strings.Contains(output, "failed to connect to local tailscaled") ||
		strings.Contains(output, "no such file or directory") ||
		strings.Contains(output, "connection refused") {
		return fmt.Errorf("daemon not ready: %s", strings.TrimSpace(output))
	}

	return nil
}

func isSocketAlive(ctx context.Context, socketPath string) bool {
	if socketPath == "" {
		return false
	}
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		return false
	}

	var d net.Dialer
	dialCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()

	conn, err := d.DialContext(dialCtx, "unix", socketPath)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
