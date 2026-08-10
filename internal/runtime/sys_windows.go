//go:build windows

package runtime

import (
	"os/exec"
	"syscall"
)

// ConfigureCmdHideWindow configures the process attributes to hide the console window.
func ConfigureCmdHideWindow(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= 0x08000000 // CREATE_NO_WINDOW
}

// ConfigureCmdDaemon configures the daemon process attributes.
func ConfigureCmdDaemon(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= 0x00000200 | 0x08000000 // CREATE_NEW_PROCESS_GROUP | CREATE_NO_WINDOW
}
