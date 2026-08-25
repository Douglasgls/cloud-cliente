package network

import (
	"net"
	"testing"

	"github.com/miekg/dns"

	"cloud-client/internal/cloud"
)

type dummyResponseWriter struct {
	lastMsg *dns.Msg
}

func (w *dummyResponseWriter) LocalAddr() net.Addr         { return nil }
func (w *dummyResponseWriter) RemoteAddr() net.Addr        { return nil }
func (w *dummyResponseWriter) WriteMsg(msg *dns.Msg) error { w.lastMsg = msg; return nil }
func (w *dummyResponseWriter) Write([]byte) (int, error)   { return 0, nil }
func (w *dummyResponseWriter) Close() error                { return nil }
func (w *dummyResponseWriter) TsigStatus() error           { return nil }
func (w *dummyResponseWriter) TsigTimersOnly(bool)         {}
func (w *dummyResponseWriter) Hijack()                     {}

func TestDNSResolver(t *testing.T) {
	registry := NewEndpointRegistry(nil)
	registry.Update(1, []cloud.NetworkEndpoint{
		{
			Hostname:    "postgres",
			FQDN:        "postgres.interno",
			TailscaleIP: "100.64.0.16",
			Status:      "online",
		},
		{
			Hostname:    "redis",
			FQDN:        "redis.interno",
			TailscaleIP: "100.64.0.17",
			Status:      "offline",
		},
	})

	resolver := NewDNSResolver(registry, nil)

	t.Run("Valid .interno Query", func(t *testing.T) {
		rw := &dummyResponseWriter{}
		req := new(dns.Msg)
		req.SetQuestion("POSTGRES.INTERNO.", dns.TypeA)

		resolver.ServeDNS(rw, req)

		if rw.lastMsg == nil {
			t.Fatal("expected DNS response, got nil")
		}
		if rw.lastMsg.Rcode != dns.RcodeSuccess {
			t.Errorf("expected RcodeSuccess, got %v", rw.lastMsg.Rcode)
		}
		if len(rw.lastMsg.Answer) != 1 {
			t.Fatalf("expected 1 answer RR, got %d", len(rw.lastMsg.Answer))
		}
		aRecord, ok := rw.lastMsg.Answer[0].(*dns.A)
		if !ok {
			t.Fatalf("expected *dns.A RR, got %T", rw.lastMsg.Answer[0])
		}
		if aRecord.A.String() != "127.0.0.1" {
			t.Errorf("expected IP 127.0.0.1 (local proxy), got %s", aRecord.A.String())
		}
	})

	t.Run("Offline Endpoint Query", func(t *testing.T) {
		rw := &dummyResponseWriter{}
		req := new(dns.Msg)
		req.SetQuestion("redis.interno.", dns.TypeA)

		resolver.ServeDNS(rw, req)

		if rw.lastMsg == nil || rw.lastMsg.Rcode != dns.RcodeServerFailure {
			t.Errorf("expected SERVFAIL (RcodeServerFailure), got %v", rw.lastMsg.Rcode)
		}
	})

	t.Run("Non-existent Endpoint Query", func(t *testing.T) {
		rw := &dummyResponseWriter{}
		req := new(dns.Msg)
		req.SetQuestion("nao-existe.interno.", dns.TypeA)

		resolver.ServeDNS(rw, req)

		if rw.lastMsg == nil || rw.lastMsg.Rcode != dns.RcodeNameError {
			t.Errorf("expected NXDOMAIN (RcodeNameError), got %v", rw.lastMsg.Rcode)
		}
	})

	t.Run("External Domain Query", func(t *testing.T) {
		rw := &dummyResponseWriter{}
		req := new(dns.Msg)
		req.SetQuestion("google.com.", dns.TypeA)

		resolver.ServeDNS(rw, req)

		if rw.lastMsg == nil || rw.lastMsg.Rcode != dns.RcodeRefused {
			t.Errorf("expected REFUSED (RcodeRefused) for external queries, got %v", rw.lastMsg.Rcode)
		}
	})
}
