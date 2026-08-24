//go:build linux

package network

import (
	"bytes"
	"fmt"
	"net"
	"os/exec"
	"sync"

	"cloud-client/pkg/logger"
)

const (
	VirtualDNSInterfaceName = "cloud-dns"
	VirtualDNSDummyIP       = "192.168.254.254/32"
)

type LinuxDNSIntegration struct {
	logger  *logger.Logger
	enabled bool
	mu      sync.Mutex
}

func NewOSDNSIntegration(log *logger.Logger) DnsSystemIntegration {
	return &LinuxDNSIntegration{
		logger: log,
	}
}

func isSystemdResolvedActive() bool {
	cmd := exec.Command("systemctl", "is-active", "--quiet", "systemd-resolved")
	if err := cmd.Run(); err == nil {
		return true
	}

	cmd = exec.Command("resolvectl", "status")
	if err := cmd.Run(); err == nil {
		return true
	}

	return false
}

func interfaceExists(name string) bool {
	_, err := net.InterfaceByName(name)
	return err == nil
}

// Enable reuses the existing "cloud-dns" virtual interface if present, or creates it once.
// It assigns systemd-resolved routing domain (~interno -> 127.0.0.1) on this fixed link.
// Physical interfaces (enp37s0, etc.) are NEVER touched.
func (l *LinuxDNSIntegration) Enable(listenAddr string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !isSystemdResolvedActive() {
		return fmt.Errorf("systemd-resolved is not active on this Linux system; OS DNS integration requires systemd-resolved")
	}

	listenIP, _, err := net.SplitHostPort(listenAddr)
	if err != nil {
		listenIP = "127.0.0.1"
	}

	var stderr bytes.Buffer

	// 1. Create interface only if it doesn't already exist (keeps the link index fixed forever)
	if !interfaceExists(VirtualDNSInterfaceName) {
		cmdAdd := exec.Command("ip", "link", "add", "dev", VirtualDNSInterfaceName, "type", "dummy")
		cmdAdd.Stderr = &stderr
		if err := cmdAdd.Run(); err != nil {
			return fmt.Errorf("failed to create virtual network interface %s: %w (stderr: %s)", VirtualDNSInterfaceName, err, stderr.String())
		}

		// Assign static L3 IP so systemd-resolved activates DNS scope
		cmdIP := exec.Command("ip", "addr", "add", VirtualDNSDummyIP, "dev", VirtualDNSInterfaceName)
		cmdIP.Stderr = &stderr
		_ = cmdIP.Run()
	}

	// 2. Ensure interface is UP
	cmdUp := exec.Command("ip", "link", "set", "dev", VirtualDNSInterfaceName, "up")
	cmdUp.Stderr = &stderr
	if err := cmdUp.Run(); err != nil {
		return fmt.Errorf("failed to bring up virtual network interface %s: %w (stderr: %s)", VirtualDNSInterfaceName, err, stderr.String())
	}

	// 3. Configure systemd-resolved ONLY on the fixed virtual interface "cloud-dns"
	// - DNS server: 127.0.0.1
	// - Routing domain: ~interno
	// - Default route: false (ensures non-.interno traffic never uses this link)
	cmdDns := exec.Command("resolvectl", "dns", VirtualDNSInterfaceName, listenIP)
	if err := cmdDns.Run(); err != nil {
		_ = exec.Command("systemd-resolve", fmt.Sprintf("--interface=%s", VirtualDNSInterfaceName), fmt.Sprintf("--set-dns=%s", listenIP)).Run()
	}

	cmdDomain := exec.Command("resolvectl", "domain", VirtualDNSInterfaceName, "~interno")
	if err := cmdDomain.Run(); err != nil {
		_ = exec.Command("systemd-resolve", fmt.Sprintf("--interface=%s", VirtualDNSInterfaceName), "--set-domain=~interno").Run()
	}

	_ = exec.Command("resolvectl", "default-route", VirtualDNSInterfaceName, "false").Run()

	// 4. Reset server features and flush DNS caches
	_ = exec.Command("resolvectl", "reset-server-features").Run()
	_ = exec.Command("resolvectl", "flush-caches").Run()

	l.enabled = true
	if l.logger != nil {
		l.logger.Info("[LinuxDNS] Active virtual interface '%s' with systemd-resolved routing domain (~interno -> %s)", VirtualDNSInterfaceName, listenIP)
	}

	return nil
}

// Disable deactivates DNS routing on "cloud-dns" without deleting the link.
// The interface stays at a fixed link index in the kernel, avoiding index increments (+1).
func (l *LinuxDNSIntegration) Disable() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Revert systemd-resolved settings on cloud-dns (clears ~interno domain and 127.0.0.1 DNS)
	_ = exec.Command("resolvectl", "revert", VirtualDNSInterfaceName).Run()
	_ = exec.Command("ip", "link", "set", "dev", VirtualDNSInterfaceName, "down").Run()

	l.enabled = false
	_ = exec.Command("resolvectl", "flush-caches").Run()
	if l.logger != nil {
		l.logger.Info("[LinuxDNS] Deactivated DNS routing on virtual interface '%s'", VirtualDNSInterfaceName)
	}
	return nil
}
