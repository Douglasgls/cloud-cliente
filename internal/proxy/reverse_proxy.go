package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"time"
)

func NewReverseProxy(targetURL *url.URL) *httputil.ReverseProxy {
	return NewReverseProxyWithSocks5(targetURL, "")
}

func NewReverseProxyWithSocks5(targetURL *url.URL, socks5Addr string) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = targetURL.Host
		if req.Header.Get("User-Agent") == "" {
			req.Header.Set("User-Agent", "cloud-client-proxy")
		}
	}

	if socks5Addr != "" {
		proxy.Transport = &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialSocks5(socks5Addr, addr)
			},
			ResponseHeaderTimeout: 30 * time.Second,
			IdleConnTimeout:       60 * time.Second,
		}
	}

	return proxy
}

func dialSocks5(socks5Addr, targetAddr string) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", socks5Addr, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SOCKS5 proxy %s: %w", socks5Addr, err)
	}

	// 1. Method negotiation: VER=5, NMETHODS=1, METHODS=[0 (NO AUTH)]
	if _, err := conn.Write([]byte{0x05, 0x01, 0x00}); err != nil {
		conn.Close()
		return nil, err
	}

	buf := make([]byte, 2)
	if _, err := io.ReadFull(conn, buf); err != nil || buf[0] != 0x05 || buf[1] != 0x00 {
		conn.Close()
		return nil, fmt.Errorf("SOCKS5 method negotiation failed")
	}

	// 2. CONNECT request
	host, portStr, err := net.SplitHostPort(targetAddr)
	if err != nil {
		host = targetAddr
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
		return nil, err
	}

	resp := make([]byte, 4)
	if _, err := io.ReadFull(conn, resp); err != nil || resp[0] != 0x05 || resp[1] != 0x00 {
		conn.Close()
		return nil, fmt.Errorf("SOCKS5 connect request failed (status: %d)", resp[1])
	}

	switch resp[3] {
	case 0x01: // IPv4
		dummy := make([]byte, 4+2)
		_, _ = io.ReadFull(conn, dummy)
	case 0x04: // IPv6
		dummy := make([]byte, 16+2)
		_, _ = io.ReadFull(conn, dummy)
	case 0x03: // Domain
		var domainLen [1]byte
		_, _ = io.ReadFull(conn, domainLen[:])
		dummy := make([]byte, int(domainLen[0])+2)
		_, _ = io.ReadFull(conn, dummy)
	}

	return conn, nil
}
