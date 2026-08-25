//go:build windows

package network

import (
	"fmt"
	"net"
	"os/exec"
	"syscall"

	"cloud-client/pkg/logger"
)

type WindowsDNSIntegration struct {
	logger *logger.Logger
}

const NRPTCommentIdentifier = "CloudClient-Internal-DNS"

func hideCmdWindow(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= 0x08000000 // CREATE_NO_WINDOW
	cmd.SysProcAttr.HideWindow = true
}

func NewOSDNSIntegration(log *logger.Logger) DnsSystemIntegration {
	return &WindowsDNSIntegration{
		logger: log,
	}
}

func (w *WindowsDNSIntegration) Enable(listenAddr string) error {
	listenIP, _, err := net.SplitHostPort(listenAddr)
	if err != nil {
		listenIP = "127.0.0.1"
	}

	// Clean up any existing rule from previous runs first
	_ = w.Disable()

	psCmd := fmt.Sprintf(
		"Add-DnsClientNrptRule -Namespace '.interno' -NameServers '%s' -Comment '%s'",
		listenIP,
		NRPTCommentIdentifier,
	)

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", psCmd)
	hideCmdWindow(cmd)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Attempt elevated execution via Start-Process -Verb RunAs (triggers Windows UAC prompt if un-elevated)
		elevatedCmd := fmt.Sprintf(
			`Start-Process powershell -ArgumentList "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", "%s" -Verb RunAs -WindowStyle Hidden -Wait`,
			psCmd,
		)
		fallbackCmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", elevatedCmd)
		hideCmdWindow(fallbackCmd)

		if fallbackOut, fallbackErr := fallbackCmd.CombinedOutput(); fallbackErr != nil {
			return fmt.Errorf("failed to add NRPT rule via PowerShell (requires Administrator privileges): %v (output: %s; elevated fallback: %s)", err, string(output), string(fallbackOut))
		}
	}

	if w.logger != nil {
		w.logger.Info("[WindowsDNS] Created NRPT rule for .interno -> %s (Comment=%s)", listenIP, NRPTCommentIdentifier)
	}

	// Flush DNS cache silently
	flushCmd := exec.Command("ipconfig", "/flushdns")
	hideCmdWindow(flushCmd)
	_ = flushCmd.Run()

	return nil
}

func (w *WindowsDNSIntegration) Disable() error {
	psCmd := fmt.Sprintf(
		"Get-DnsClientNrptRule | Where-Object { $_.Comment -eq '%s' } | Remove-DnsClientNrptRule -Force",
		NRPTCommentIdentifier,
	)

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", psCmd)
	hideCmdWindow(cmd)

	output, err := cmd.CombinedOutput()
	if err != nil {
		elevatedCmd := fmt.Sprintf(
			`Start-Process powershell -ArgumentList "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", "%s" -Verb RunAs -WindowStyle Hidden -Wait`,
			psCmd,
		)
		fallbackCmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", elevatedCmd)
		hideCmdWindow(fallbackCmd)

		if fallbackOut, fallbackErr := fallbackCmd.CombinedOutput(); fallbackErr != nil {
			if w.logger != nil {
				w.logger.Warn("[WindowsDNS] Failed to remove NRPT rule: %v (output: %s; elevated fallback: %s)", err, string(output), string(fallbackOut))
			}
		}
	} else if w.logger != nil {
		w.logger.Info("[WindowsDNS] Removed NRPT rule '%s'", NRPTCommentIdentifier)
	}

	// Flush DNS cache silently
	flushCmd := exec.Command("ipconfig", "/flushdns")
	hideCmdWindow(flushCmd)
	_ = flushCmd.Run()

	return nil
}
