package proxy

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"cloud-client/pkg/logger"
)

type ProxyService interface {
	Start(target string) (localURL string, err error)
	Stop(ctx context.Context) error
	TargetURL() string
	LocalURL() string
}

type Service struct {
	logger     *logger.Logger
	server     *Server
	socks5Addr string
	localURL   string
	targetURL  string
	mu         sync.Mutex
}

func NewService(log *logger.Logger, socks5Addr string) *Service {
	return &Service{
		logger:     log,
		socks5Addr: socks5Addr,
	}
}

func (s *Service) TargetURL() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.targetURL
}

func (s *Service) LocalURL() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.localURL
}

func (s *Service) Start(target string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.server != nil {
		return s.localURL, nil
	}

	cleanTarget := strings.TrimSpace(target)
	if cleanTarget == "" {
		return "", fmt.Errorf("target host cannot be empty")
	}

	// Extract first line if hostname contains multiple lines
	lines := strings.Split(cleanTarget, "\n")
	cleanTarget = strings.TrimSpace(lines[0])

	if !strings.HasPrefix(cleanTarget, "http://") && !strings.HasPrefix(cleanTarget, "https://") {
		cleanTarget = "http://" + cleanTarget
	}

	parsedURL, err := url.Parse(cleanTarget)
	if err != nil {
		return "", fmt.Errorf("invalid target URL %s: %w", cleanTarget, err)
	}

	listener, err := findAvailableListener(8080, 8095)
	if err != nil {
		return "", fmt.Errorf("failed to find an available local port: %w", err)
	}

	host, portStr, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		host = "127.0.0.1"
		portStr = "8080"
	}
	if host == "::" || host == "0.0.0.0" {
		host = "127.0.0.1"
	}

	s.localURL = fmt.Sprintf("http://%s:%s", host, portStr)
	s.targetURL = parsedURL.String()

	revProxy := NewReverseProxyWithSocks5(parsedURL, s.socks5Addr)
	s.server = NewServer(listener, revProxy)

	go func() {
		if err := s.server.Start(); err != nil && s.logger != nil {
			s.logger.Error("Proxy server error: %v", err)
		}
	}()

	return s.localURL, nil
}

func (s *Service) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.server == nil {
		return nil
	}

	err := s.server.Stop(ctx)
	s.server = nil
	return err
}

func findAvailableListener(startPort, endPort int) (net.Listener, error) {
	for port := startPort; port <= endPort; port++ {
		addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
		l, err := net.Listen("tcp", addr)
		if err == nil {
			return l, nil
		}
	}
	// Fallback to dynamic OS port assignment
	return net.Listen("tcp", "127.0.0.1:0")
}
