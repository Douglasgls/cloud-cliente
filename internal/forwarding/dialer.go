package forwarding

import (
	"context"
	"fmt"
	"io"
	"net"
	"strconv"

	"cloud-client/pkg/logger"
)

type Dialer interface {
	DialContext(ctx context.Context, network, addr string) (net.Conn, error)
}

type DirectDialer struct{}

func NewDirectDialer() *DirectDialer {
	return &DirectDialer{}
}

func (d *DirectDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	var netDialer net.Dialer
	return netDialer.DialContext(ctx, network, addr)
}

type Socks5Dialer struct {
	socks5Addr string
	log        *logger.Logger
}

func NewSocks5Dialer(socks5Addr string, log *logger.Logger) *Socks5Dialer {
	return &Socks5Dialer{
		socks5Addr: socks5Addr,
		log:        log,
	}
}

func (s *Socks5Dialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	socksAddr := s.socks5Addr
	if socksAddr == "" {
		socksAddr = "127.0.0.1:1055"
	}

	var netDialer net.Dialer
	conn, err := netDialer.DialContext(ctx, "tcp", socksAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SOCKS5 proxy %s: %w", socksAddr, err)
	}

	if s.log != nil {
		s.log.Info("[SOCKS5] conexão SOCKS5 criada em %s", socksAddr)
	}

	// 1. Method negotiation: VER=5, NMETHODS=1, METHODS=[0 (NO AUTH)]
	if _, err := conn.Write([]byte{0x05, 0x01, 0x00}); err != nil {
		conn.Close()
		return nil, fmt.Errorf("SOCKS5 method write failed: %w", err)
	}

	buf := make([]byte, 2)
	if _, err := io.ReadFull(conn, buf); err != nil || buf[0] != 0x05 || buf[1] != 0x00 {
		conn.Close()
		return nil, fmt.Errorf("SOCKS5 method negotiation failed")
	}

	// 2. CONNECT request
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
		portStr = "80"
	}
	port, _ := strconv.Atoi(portStr)

	req := []byte{0x05, 0x01, 0x00} // VER=5, CMD=1 (CONNECT), RSV=0
	ip := net.ParseIP(host)
	if ip4 := ip.To4(); ip4 != nil {
		req = append(req, 0x01)
		req = append(req, ip4...)
	} else if ip6 := ip.To16(); ip6 != nil {
		req = append(req, 0x04)
		req = append(req, ip6...)
	} else {
		req = append(req, 0x03, byte(len(host)))
		req = append(req, []byte(host)...)
	}

	req = append(req, byte(port>>8), byte(port&0xff))

	if _, err := conn.Write(req); err != nil {
		conn.Close()
		return nil, fmt.Errorf("SOCKS5 connect write failed: %w", err)
	}

	resp := make([]byte, 4)
	if _, err := io.ReadFull(conn, resp); err != nil || resp[0] != 0x05 || resp[1] != 0x00 {
		conn.Close()
		return nil, fmt.Errorf("SOCKS5 connect request failed (status: %d)", resp[1])
	}

	switch resp[3] {
	case 0x01: // IPv4
		dummy := make([]byte, 4+2)
		if _, err := io.ReadFull(conn, dummy); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed reading SOCKS5 bind addr: %w", err)
		}
	case 0x04: // IPv6
		dummy := make([]byte, 16+2)
		if _, err := io.ReadFull(conn, dummy); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed reading SOCKS5 bind addr: %w", err)
		}
	case 0x03: // Domain
		var domainLen [1]byte
		if _, err := io.ReadFull(conn, domainLen[:]); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed reading SOCKS5 domain len: %w", err)
		}
		dummy := make([]byte, int(domainLen[0])+2)
		if _, err := io.ReadFull(conn, dummy); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed reading SOCKS5 domain addr: %w", err)
		}
	}

	if s.log != nil {
		s.log.Info("[SOCKS5] conexão remota estabelecida para %s", addr)
	}

	return conn, nil
}
