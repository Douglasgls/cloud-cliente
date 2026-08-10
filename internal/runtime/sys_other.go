//go:build !windows

package runtime

import (
	"os/exec"
)

// ConfigureCmdHideWindow is a no-op on non-Windows platforms.
func ConfigureCmdHideWindow(cmd *exec.Cmd) {}

// ConfigureCmdDaemon is a no-op on non-Windows platforms.
func ConfigureCmdDaemon(cmd *exec.Cmd) {}
