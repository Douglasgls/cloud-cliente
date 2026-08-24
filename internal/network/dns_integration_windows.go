//go:build windows

package network

import (
	"fmt"
	"net"
	"os/exec"

	"cloud-client/pkg/logger"
)

type WindowsDNSIntegration struct {
	logger *logger.Logger
}

const NRPTCommentIdentifier = "CloudClient-Internal-DNS"

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

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to add NRPT rule via PowerShell (requires Administrator privileges): %v (output: %s)", err, string(output))
	}

	if w.logger != nil {
		w.logger.Info("[WindowsDNS] Created NRPT rule for .interno -> %s (Comment=%s)", listenIP, NRPTCommentIdentifier)
	}

	// Flush DNS cache
	_ = exec.Command("ipconfig", "/flushdns").Run()
	return nil
}

func (w *WindowsDNSIntegration) Disable() error {
	psCmd := fmt.Sprintf(
		"Get-DnsClientNrptRule | Where-Object { $_.Comment -eq '%s' } | Remove-DnsClientNrptRule -Force",
		NRPTCommentIdentifier,
	)

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if w.logger != nil {
			w.logger.Warn("[WindowsDNS] Failed to remove NRPT rule: %v (output: %s)", err, string(output))
		}
	} else if w.logger != nil {
		w.logger.Info("[WindowsDNS] Removed NRPT rule '%s'", NRPTCommentIdentifier)
	}

	_ = exec.Command("ipconfig", "/flushdns").Run()
	return nil
}
