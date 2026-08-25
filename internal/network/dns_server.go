package network

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/miekg/dns"

	"cloud-client/pkg/logger"
)

type DNSServer struct {
	addr      string
	resolver  *DNSResolver
	logger    *logger.Logger
	udpServer *dns.Server
	tcpServer *dns.Server

	mu      sync.Mutex
	running bool
}

func NewDNSServer(listenAddr string, resolver *DNSResolver, log *logger.Logger) *DNSServer {
	if listenAddr == "" {
		listenAddr = "127.0.0.1:53"
	}
	return &DNSServer{
		addr:     listenAddr,
		resolver: resolver,
		logger:   log,
	}
}

func (s *DNSServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	// Test if address is bindable before launching servers
	udpConn, err := net.ListenPacket("udp", s.addr)
	if err != nil {
		return fmt.Errorf("failed to bind UDP listener on %s (privileges required for port 53): %w", s.addr, err)
	}
	_ = udpConn.Close()

	tcpConn, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("failed to bind TCP listener on %s: %w", s.addr, err)
	}
	_ = tcpConn.Close()

	s.udpServer = &dns.Server{
		Addr:    s.addr,
		Net:     "udp",
		Handler: s.resolver,
	}

	s.tcpServer = &dns.Server{
		Addr:    s.addr,
		Net:     "tcp",
		Handler: s.resolver,
	}

	errChan := make(chan error, 2)

	go func() {
		if s.logger != nil {
			s.logger.Info("[DNSServer] Starting UDP listener on %s", s.addr)
		}
		if err := s.udpServer.ListenAndServe(); err != nil {
			errChan <- fmt.Errorf("UDP DNS server error: %w", err)
		}
	}()

	go func() {
		if s.logger != nil {
			s.logger.Info("[DNSServer] Starting TCP listener on %s", s.addr)
		}
		if err := s.tcpServer.ListenAndServe(); err != nil {
			errChan <- fmt.Errorf("TCP DNS server error: %w", err)
		}
	}()

	// Give servers a brief moment to spin up
	time.Sleep(100 * time.Millisecond)

	select {
	case err := <-errChan:
		_ = s.stopInternal()
		return err
	default:
	}

	s.running = true
	if s.logger != nil {
		s.logger.Info("[DNSServer] Local DNS server successfully running on %s", s.addr)
	}
	return nil
}

func (s *DNSServer) stopInternal() error {
	var errs []error
	if s.udpServer != nil {
		if err := s.udpServer.Shutdown(); err != nil {
			errs = append(errs, err)
		}
		s.udpServer = nil
	}
	if s.tcpServer != nil {
		if err := s.tcpServer.Shutdown(); err != nil {
			errs = append(errs, err)
		}
		s.tcpServer = nil
	}
	s.running = false
	if len(errs) > 0 {
		return fmt.Errorf("errors during DNS server shutdown: %v", errs)
	}
	return nil
}

func (s *DNSServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	if s.logger != nil {
		s.logger.Info("[DNSServer] Stopping local DNS server on %s...", s.addr)
	}
	return s.stopInternal()
}
