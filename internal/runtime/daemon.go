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
		// Block until the anchor is confirmed live — this guarantees that
		// subsequent tailscale CLI calls won't be the last client on the pipe.
		if err := m.startAnchorAndWait(ctx, 5*time.Second); err != nil && m.logger != nil {
			m.logger.Error("[anchor] warning: %v — continuing anyway", err)
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

	// Always use userspace networking on both Windows and Linux
	args = append(args, "--tun=userspace-networking")

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

	ConfigureCmdDaemon(cmd)

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("failed to start tailscaled background process: %w", err)
	}

	m.daemonPID = cmd.Process.Pid
	m.daemonCmd = cmd

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
				// Block until anchor is confirmed live before returning,
				// so the caller can run tailscale up safely.
				if err := m.startAnchorAndWait(ctx, 5*time.Second); err != nil && m.logger != nil {
					m.logger.Error("[anchor] warning: %v — continuing anyway", err)
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

	if runtime.GOOS == "windows" {
		// On Windows, passively check Named Pipe existence via os.Stat.
		// Do NOT call tailscale CLI here — every CLI invocation opens the pipe
		// and closes it on exit, which resets the daemon session to NoState
		// ("client disconnected... disconnecting Tailscale") when it is the
		// last client.  os.Stat calls GetFileAttributesW which does NOT open
		// a Named Pipe connection, so it is safe.
		_, err := os.Stat(m.socketPath)
		if err != nil {
			return fmt.Errorf("named pipe not yet available: %w", err)
		}
		return nil
	}

	// Linux: dial the Unix socket to confirm it is reachable.
	var d net.Dialer
	dialCtx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	conn, err := d.DialContext(dialCtx, "unix", m.socketPath)
	cancel()
	if err != nil {
		return fmt.Errorf("socket not reachable: %w", err)
	}
	conn.Close()

	// Run tailscale status to verify end-to-end daemon responsiveness.
	valCtx, cancelVal := context.WithTimeout(ctx, 1*time.Second)
	defer cancelVal()

	cmd := exec.CommandContext(valCtx, m.tailscalePath, "--socket="+m.socketPath, "status")
	ConfigureCmdHideWindow(cmd)
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

	if runtime.GOOS == "windows" {
		if strings.HasPrefix(socketPath, `\\.\pipe\`) {
			// IMPORTANT: Do NOT use os.OpenFile on the Named Pipe here.
			// Opening the pipe registers a control-session with tailscaled;
			// closing it immediately triggers the daemon shutdown.
			// Use a passive filesystem probe instead: Named Pipes on Windows
			// appear as entries under \\.\pipe\, so os.Stat is safe and
			// does not open a control session.
			_, err := os.Stat(socketPath)
			return err == nil
		}
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
