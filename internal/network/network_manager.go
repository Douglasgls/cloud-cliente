package network

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cloud-client/internal/cloud"
	"cloud-client/pkg/logger"
)

type NetworkManager struct {
	registry      *EndpointRegistry
	cloudSync     *CloudSync
	resolver      *DNSResolver
	dnsServer     *DNSServer
	osIntegration DnsSystemIntegration
	logger        *logger.Logger
	listenAddr    string

	mu      sync.Mutex
	running bool
}

func NewNetworkManager(cloudClient cloud.CloudClient, listenAddr string, syncInterval time.Duration, log *logger.Logger) *NetworkManager {
	if listenAddr == "" {
		listenAddr = "127.0.0.1:53"
	}
	registry := NewEndpointRegistry(log)
	cloudSync := NewCloudSync(cloudClient, registry, "", syncInterval, log)
	resolver := NewDNSResolver(registry, log)
	dnsServer := NewDNSServer(listenAddr, resolver, log)
	osIntegration := NewOSDNSIntegration(log)

	return &NetworkManager{
		registry:      registry,
		cloudSync:     cloudSync,
		resolver:      resolver,
		dnsServer:     dnsServer,
		osIntegration: osIntegration,
		logger:        log,
		listenAddr:    listenAddr,
	}
}

func NewNetworkManagerWithComponents(
	registry *EndpointRegistry,
	cloudSync *CloudSync,
	dnsServer *DNSServer,
	osIntegration DnsSystemIntegration,
	listenAddr string,
	log *logger.Logger,
) *NetworkManager {
	if listenAddr == "" {
		listenAddr = "127.0.0.1:53"
	}
	return &NetworkManager{
		registry:      registry,
		cloudSync:     cloudSync,
		dnsServer:     dnsServer,
		osIntegration: osIntegration,
		logger:        log,
		listenAddr:    listenAddr,
	}
}

// Start executes the exact approved sequence:
// 1. Start local DNS Server on 127.0.0.1:53
// 2. Perform initial sync with Cloud API
// 3. Enable OS DNS integration (systemd-resolved / NRPT)
// 4. Start background periodic polling loop
func (m *NetworkManager) Start(ctx context.Context, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return nil
	}

	if m.logger != nil {
		m.logger.Info("[NetworkManager] Starting local network & DNS subsystem...")
	}

	// Set auth token for cloud sync
	m.cloudSync.token = token

	// 1. Start local DNS Server
	if err := m.dnsServer.Start(); err != nil {
		return fmt.Errorf("failed to start local DNS server: %w", err)
	}

	// 2. Initial sync with Cloud
	if m.logger != nil {
		m.logger.Info("[NetworkManager] Performing initial endpoints sync with Cloud...")
	}
	if err := m.cloudSync.SyncOnce(ctx); err != nil {
		if m.logger != nil {
			m.logger.Warn("[NetworkManager] Initial sync returned error (will retry in background): %v", err)
		}
	}

	// 3. Enable OS DNS integration
	if m.osIntegration != nil {
		if m.logger != nil {
			m.logger.Info("[NetworkManager] Enabling OS DNS integration for *.interno -> %s...", m.listenAddr)
		}
		if err := m.osIntegration.Enable(m.listenAddr); err != nil {
			if m.logger != nil {
				m.logger.Warn("[NetworkManager] OS DNS integration warning: %v", err)
			}
		}
	}


	// 4. Start background periodic polling
	if err := m.cloudSync.Start(ctx); err != nil {
		if m.logger != nil {
			m.logger.Warn("[NetworkManager] Failed to start background sync loop: %v", err)
		}
	}

	m.running = true
	if m.logger != nil {
		m.logger.Info("[NetworkManager] Network & DNS subsystem fully initialized and running.")
	}
	return nil
}

// Stop executes the exact approved shutdown sequence:
// 1. Stop background polling/sync
// 2. Disable OS DNS integration
// 3. Stop local DNS server
func (m *NetworkManager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}

	if m.logger != nil {
		m.logger.Info("[NetworkManager] Shutting down network & DNS subsystem...")
	}

	// 1. Stop polling loop
	if m.cloudSync != nil {
		m.cloudSync.Stop()
	}

	// 2. Disable OS DNS integration
	if m.osIntegration != nil {
		if m.logger != nil {
			m.logger.Info("[NetworkManager] Disabling OS DNS integration...")
		}
		if err := m.osIntegration.Disable(); err != nil {
			if m.logger != nil {
				m.logger.Warn("[NetworkManager] Error disabling OS DNS integration: %v", err)
			}
		}
	}

	// 3. Stop local DNS Server
	if m.dnsServer != nil {
		if err := m.dnsServer.Stop(); err != nil {
			if m.logger != nil {
				m.logger.Warn("[NetworkManager] Error stopping DNS server: %v", err)
			}
		}
	}

	m.running = false
	if m.logger != nil {
		m.logger.Info("[NetworkManager] Network & DNS subsystem stopped successfully.")
	}
	return nil
}

func (m *NetworkManager) Registry() *EndpointRegistry {
	return m.registry
}
