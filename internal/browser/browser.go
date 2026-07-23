package browser

import (
	"fmt"
	"os/exec"
	"runtime"
)

type Opener interface {
	Open(targetURL string) error
}

type Browser struct{}

func New() *Browser {
	return &Browser{}
}

func (b *Browser) Open(targetURL string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", targetURL)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", targetURL)
	case "darwin":
		cmd = exec.Command("open", targetURL)
	default:
		return fmt.Errorf("unsupported platform %s for browser opening", runtime.GOOS)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open browser: %w", err)
	}
	return nil
}
