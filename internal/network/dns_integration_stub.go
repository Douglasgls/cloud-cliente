//go:build !linux && !windows

package network

import (
	"cloud-client/pkg/logger"
)

type StubDNSIntegration struct {
	logger *logger.Logger
}

func NewOSDNSIntegration(log *logger.Logger) DnsSystemIntegration {
	return &StubDNSIntegration{
		logger: log,
	}
}

func (s *StubDNSIntegration) Enable(listenAddr string) error {
	if s.logger != nil {
		s.logger.Warn("[OSDNSIntegration] Automatic OS DNS integration is not implemented for this platform. Local DNS server remains active on %s.", listenAddr)
	}
	return nil
}

func (s *StubDNSIntegration) Disable() error {
	return nil
}
