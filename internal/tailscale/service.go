package tailscale

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"cloud-client/internal/runtime"
)

var (
	ErrTailscaleNotInstalled = errors.New("Tailscale is not installed.")
	ErrExecutionFailed       = errors.New("tailscale command execution failed")
)

type TailscaleService interface {
	Up(ctx context.Context, loginServer, authKey, hostname string) error
	Status(ctx context.Context) (string, error)
	Version(ctx context.Context) (string, error)
}

type Service struct {
	runtime runtime.RuntimeManager
}

func NewService(runtimeMgr runtime.RuntimeManager) *Service {
	return &Service{
		runtime: runtimeMgr,
	}
}

func (s *Service) Up(ctx context.Context, loginServer, authKey, hostname string) error {
	args := []string{"up"}
	if loginServer != "" {
		args = append(args, "--login-server="+loginServer)
	}
	if authKey != "" {
		args = append(args, "--authkey="+authKey)
	}
	cleanHostname := sanitizeHostname(hostname)
	if cleanHostname != "" {
		args = append(args, "--hostname="+cleanHostname)
	}

	stdout, stderr, err := s.runCommand(ctx, args...)
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) || errors.Is(err, ErrTailscaleNotInstalled) {
			return ErrTailscaleNotInstalled
		}
		errMsg := strings.TrimSpace(stderr)
		if errMsg == "" {
			errMsg = strings.TrimSpace(stdout)
		}
		if errMsg != "" {
			return fmt.Errorf("%w: %s", ErrExecutionFailed, errMsg)
		}
		return fmt.Errorf("%w: %v", ErrExecutionFailed, err)
	}

	return nil
}

func (s *Service) Status(ctx context.Context) (string, error) {
	stdout, stderr, err := s.runCommand(ctx, "status")
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) || errors.Is(err, ErrTailscaleNotInstalled) {
			return "", ErrTailscaleNotInstalled
		}
		out := strings.TrimSpace(stdout)
		if out != "" {
			return out, nil
		}
		errMsg := strings.TrimSpace(stderr)
		if errMsg != "" {
			return "", fmt.Errorf("%w: %s", ErrExecutionFailed, errMsg)
		}
		return "", fmt.Errorf("%w: %v", ErrExecutionFailed, err)
	}
	return strings.TrimSpace(stdout), nil
}

func (s *Service) Version(ctx context.Context) (string, error) {
	stdout, stderr, err := s.runCommand(ctx, "version")
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) || errors.Is(err, ErrTailscaleNotInstalled) {
			return "", ErrTailscaleNotInstalled
		}
		return "", fmt.Errorf("%w: %s", ErrExecutionFailed, strings.TrimSpace(stderr))
	}
	return strings.TrimSpace(stdout), nil
}

func (s *Service) runCommand(ctx context.Context, args ...string) (string, string, error) {
	if s.runtime == nil || s.runtime.TailscalePath() == "" {
		return "", "", ErrTailscaleNotInstalled
	}

	var cmdArgs []string
	if socketPath := s.runtime.SocketPath(); socketPath != "" {
		cmdArgs = append(cmdArgs, "--socket="+socketPath)
	}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.CommandContext(ctx, s.runtime.TailscalePath(), cmdArgs...)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	return stdoutBuf.String(), stderrBuf.String(), err
}

func sanitizeHostname(h string) string {
	lines := strings.Split(h, "\n")
	first := strings.TrimSpace(lines[0])
	first = strings.ReplaceAll(first, ":", "-")
	return first
}
