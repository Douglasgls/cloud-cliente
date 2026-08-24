package network

import (
	"fmt"
	"net"
	"strings"
	"sync"

	"cloud-client/internal/cloud"
	"cloud-client/pkg/logger"
)

const InternalDomainSuffix = ".interno"

type Endpoint struct {
	Hostname    string
	FQDN        string
	TailscaleIP net.IP
	Status      string
	Ports       []cloud.NetworkPort
}

type EndpointRegistry struct {
	mu        sync.RWMutex
	endpoints map[string]Endpoint
	version   int64
	logger    *logger.Logger
}

func NewEndpointRegistry(log *logger.Logger) *EndpointRegistry {
	return &EndpointRegistry{
		endpoints: make(map[string]Endpoint),
		version:   -1,
		logger:    log,
	}
}

// NormalizeFQDN converts a domain name to lowercase and strips trailing dot.
// Examples:
// "POSTGRES.INTERNO." -> "postgres.interno"
// "Postgres.interno"  -> "postgres.interno"
func NormalizeFQDN(fqdn string) string {
	s := strings.ToLower(strings.TrimSpace(fqdn))
	s = strings.TrimSuffix(s, ".")
	return s
}

// ValidateEndpoint checks if FQDN ends with .interno and IP is a valid IP address.
func ValidateEndpoint(ep cloud.NetworkEndpoint) (Endpoint, error) {
	normFQDN := NormalizeFQDN(ep.FQDN)
	if normFQDN == "" {
		if ep.Hostname != "" {
			normFQDN = NormalizeFQDN(ep.Hostname + InternalDomainSuffix)
		}
	}

	if !strings.HasSuffix(normFQDN, InternalDomainSuffix) {
		return Endpoint{}, fmt.Errorf("invalid FQDN '%s': must end with '%s'", ep.FQDN, InternalDomainSuffix)
	}

	rawIPStr := strings.TrimSpace(ep.TailscaleIP)
	var ip net.IP
	for _, candidate := range strings.Fields(rawIPStr) {
		if parsed := net.ParseIP(candidate); parsed != nil {
			ip = parsed
			break
		}
	}
	if ip == nil {
		return Endpoint{}, fmt.Errorf("invalid IP address '%s' for FQDN '%s'", ep.TailscaleIP, normFQDN)
	}

	status := strings.ToLower(strings.TrimSpace(ep.Status))
	if status == "" {
		status = "online"
	}

	return Endpoint{
		Hostname:    ep.Hostname,
		FQDN:        normFQDN,
		TailscaleIP: ip,
		Status:      status,
		Ports:       ep.Ports,
	}, nil
}

// Update replaces the registry state with the new endpoints snapshot and version.
func (r *EndpointRegistry) Update(version int64, rawEndpoints []cloud.NetworkEndpoint) {
	r.mu.Lock()
	defer r.mu.Unlock()

	newMap := make(map[string]Endpoint)
	for _, raw := range rawEndpoints {
		ep, err := ValidateEndpoint(raw)
		if err != nil {
			if r.logger != nil {
				r.logger.Warn("[EndpointRegistry] Skipping invalid endpoint: %v", err)
			}
			continue
		}
		newMap[ep.FQDN] = ep
		if r.logger != nil {
			r.logger.Debug("[EndpointRegistry] Registered endpoint: FQDN=%s IP=%s Status=%s", ep.FQDN, ep.TailscaleIP.String(), ep.Status)
		}
	}

	r.endpoints = newMap
	r.version = version
	if r.logger != nil {
		r.logger.Info("[EndpointRegistry] Updated endpoints registry (version=%d, total=%d)", version, len(newMap))
	}
}

// GetByFQDN retrieves an endpoint by its FQDN (normalized internally).
func (r *EndpointRegistry) GetByFQDN(fqdn string) (Endpoint, bool) {
	norm := NormalizeFQDN(fqdn)

	r.mu.RLock()
	defer r.mu.RUnlock()

	ep, ok := r.endpoints[norm]
	return ep, ok
}

// Version returns the current registry version.
func (r *EndpointRegistry) Version() int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.version
}

// All returns a slice of all stored endpoints.
func (r *EndpointRegistry) All() []Endpoint {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]Endpoint, 0, len(r.endpoints))
	for _, ep := range r.endpoints {
		list = append(list, ep)
	}
	return list
}
