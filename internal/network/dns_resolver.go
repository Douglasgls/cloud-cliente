package network

import (
	"net"
	"strings"

	"github.com/miekg/dns"

	"cloud-client/pkg/logger"
)

// localProxyIP is the IP returned for all .interno domains.
// All traffic is routed through local proxies (127.0.0.1:PORT) which
// forward to the real Tailscale IPs internally via SOCKS5.
var localProxyIP = net.ParseIP("127.0.0.1")

type DNSResolver struct {
	registry *EndpointRegistry
	logger   *logger.Logger
}

func NewDNSResolver(registry *EndpointRegistry, log *logger.Logger) *DNSResolver {
	return &DNSResolver{
		registry: registry,
		logger:   log,
	}
}

func (r *DNSResolver) ServeDNS(w dns.ResponseWriter, req *dns.Msg) {
	msg := new(dns.Msg)
	msg.SetReply(req)
	msg.Authoritative = true
	msg.RecursionAvailable = true

	if len(req.Question) == 0 {
		msg.Rcode = dns.RcodeFormatError
		_ = w.WriteMsg(msg)
		return
	}

	q := req.Question[0]
	rawName := q.Name
	normName := NormalizeFQDN(rawName)

	if !strings.HasSuffix(normName, InternalDomainSuffix) {
		if r.logger != nil {
			r.logger.Debug("[DNS] External query refused: name=%s qtype=%s", rawName, dns.TypeToString[q.Qtype])
		}
		msg.Rcode = dns.RcodeRefused
		_ = w.WriteMsg(msg)
		return
	}

	ep, found := r.registry.GetByFQDN(normName)
	if !found {
		if r.logger != nil {
			r.logger.Debug("[DNS] Query NXDOMAIN: name=%s (normalized=%s)", rawName, normName)
		}
		msg.Rcode = dns.RcodeNameError // NXDOMAIN
		_ = w.WriteMsg(msg)
		return
	}

	if ep.Status == "offline" {
		if r.logger != nil {
			r.logger.Warn("[DNS] Query SERVFAIL (endpoint offline): name=%s", normName)
		}
		msg.Rcode = dns.RcodeServerFailure // SERVFAIL for existing offline endpoints
		_ = w.WriteMsg(msg)
		return
	}

	msg.Rcode = dns.RcodeSuccess

	switch q.Qtype {
	case dns.TypeA:
		// Always return 127.0.0.1 so the browser connects to the local proxy.
		// The local proxy forwards traffic to the real Tailscale IP via SOCKS5.
		rr := &dns.A{
			Hdr: dns.RR_Header{
				Name:   q.Name,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    5,
			},
			A: localProxyIP.To4(),
		}
		msg.Answer = append(msg.Answer, rr)
		if r.logger != nil {
			r.logger.Info("[DNS] Resolved query: name=%s -> %s (real: %s)", normName, localProxyIP.String(), ep.TailscaleIP.String())
		}
	case dns.TypeAAAA:
		// No IPv6 loopback proxy support; return empty answer for AAAA.
		if r.logger != nil {
			r.logger.Debug("[DNS] AAAA query ignored for proxy mode: name=%s", normName)
		}
	default:
		if r.logger != nil {
			r.logger.Debug("[DNS] Query unsupported type: name=%s qtype=%s", normName, dns.TypeToString[q.Qtype])
		}
	}

	_ = w.WriteMsg(msg)
}
