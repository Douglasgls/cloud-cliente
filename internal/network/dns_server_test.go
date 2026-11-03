package network

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"cloud-client/internal/cloud"
)

// getFreePort asks the kernel for a free port on 127.0.0.1.
func getFreePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func TestDNSServer_LifecycleAndQueries(t *testing.T) {
	port := getFreePort(t)
	listenAddr := fmt.Sprintf("127.0.0.1:%d", port)

	// Setup registry with test endpoints
	registry := NewEndpointRegistry(nil)
	registry.Update(1, []cloud.NetworkEndpoint{
		{
			Hostname:    "api-server",
			FQDN:        "api.interno",
			TailscaleIP: "100.64.0.10",
			Status:      "online",
		},
		{
			Hostname:    "db-server",
			FQDN:        "db.interno",
			TailscaleIP: "100.64.0.11",
			Status:      "offline",
		},
	})

	resolver := NewDNSResolver(registry, nil)
	server := NewDNSServer(listenAddr, resolver, nil)

	// 1. Start Server
	err := server.Start()
	require.NoError(t, err, "Server should start without error")

	// Calling Start again should be a no-op and return nil
	assert.NoError(t, server.Start())

	udpClient := &dns.Client{Net: "udp", Timeout: 2 * time.Second}
	tcpClient := &dns.Client{Net: "tcp", Timeout: 2 * time.Second}

	// 2. Query valid .interno name over UDP
	t.Run("Resolve valid .interno over UDP", func(t *testing.T) {
		req := new(dns.Msg)
		req.SetQuestion("api.interno.", dns.TypeA)

		resp, _, err := udpClient.Exchange(req, listenAddr)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, dns.RcodeSuccess, resp.Rcode)
		require.Len(t, resp.Answer, 1)

		aRecord, ok := resp.Answer[0].(*dns.A)
		require.True(t, ok)
		assert.Equal(t, "127.0.0.1", aRecord.A.String())
	})

	// 3. Query valid .interno name over TCP
	t.Run("Resolve valid .interno over TCP", func(t *testing.T) {
		req := new(dns.Msg)
		req.SetQuestion("api.interno.", dns.TypeA)

		resp, _, err := tcpClient.Exchange(req, listenAddr)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, dns.RcodeSuccess, resp.Rcode)
		require.Len(t, resp.Answer, 1)

		aRecord, ok := resp.Answer[0].(*dns.A)
		require.True(t, ok)
		assert.Equal(t, "127.0.0.1", aRecord.A.String())
	})

	// 4. Query non-existent .interno domain (NXDOMAIN)
	t.Run("Query NXDOMAIN for non-existent domain", func(t *testing.T) {
		req := new(dns.Msg)
		req.SetQuestion("naoexiste.interno.", dns.TypeA)

		resp, _, err := udpClient.Exchange(req, listenAddr)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, dns.RcodeNameError, resp.Rcode)
		assert.Empty(t, resp.Answer)
	})

	// 5. Query offline endpoint (SERVFAIL)
	t.Run("Query offline endpoint", func(t *testing.T) {
		req := new(dns.Msg)
		req.SetQuestion("db.interno.", dns.TypeA)

		resp, _, err := udpClient.Exchange(req, listenAddr)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, dns.RcodeServerFailure, resp.Rcode)
	})

	// 6. Query external domain (Refused)
	t.Run("Query external domain is refused", func(t *testing.T) {
		req := new(dns.Msg)
		req.SetQuestion("example.com.", dns.TypeA)

		resp, _, err := udpClient.Exchange(req, listenAddr)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, dns.RcodeRefused, resp.Rcode)
	})

	// 7. Concurrent Queries test
	t.Run("Concurrent queries stress test", func(t *testing.T) {
		const workers = 40
		var wg sync.WaitGroup
		wg.Add(workers)

		for i := 0; i < workers; i++ {
			go func(idx int) {
				defer wg.Done()
				client := &dns.Client{Net: "udp", Timeout: 2 * time.Second}
				req := new(dns.Msg)

				if idx%2 == 0 {
					req.SetQuestion("api.interno.", dns.TypeA)
					resp, _, err := client.Exchange(req, listenAddr)
					assert.NoError(t, err)
					if resp != nil {
						assert.Equal(t, dns.RcodeSuccess, resp.Rcode)
					}
				} else {
					req.SetQuestion("unknown.interno.", dns.TypeA)
					resp, _, err := client.Exchange(req, listenAddr)
					assert.NoError(t, err)
					if resp != nil {
						assert.Equal(t, dns.RcodeNameError, resp.Rcode)
					}
				}
			}(i)
		}

		wg.Wait()
	})

	// 8. Graceful Stop
	t.Run("Stop server and verify port is released", func(t *testing.T) {
		err := server.Stop()
		assert.NoError(t, err)

		// Calling Stop again should be a safe no-op
		assert.NoError(t, server.Stop())

		// Verify connection is now refused
		req := new(dns.Msg)
		req.SetQuestion("api.interno.", dns.TypeA)
		shortClient := &dns.Client{Net: "tcp", Timeout: 300 * time.Millisecond}
		_, _, err = shortClient.Exchange(req, listenAddr)
		assert.Error(t, err, "Expected connection error after stopping server")
	})
}
