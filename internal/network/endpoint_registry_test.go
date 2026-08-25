package network

import (
	"fmt"
	"sync"
	"testing"

	"cloud-client/internal/cloud"
)

func TestNormalizeFQDN(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"POSTGRES.INTERNO.", "postgres.interno"},
		{"Postgres.interno", "postgres.interno"},
		{"  meu-app.interno.  ", "meu-app.interno"},
		{"redis.interno", "redis.interno"},
	}

	for _, tt := range tests {
		got := NormalizeFQDN(tt.input)
		if got != tt.expected {
			t.Errorf("NormalizeFQDN(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestValidateEndpoint(t *testing.T) {
	valid := cloud.NetworkEndpoint{
		Hostname:    "postgres",
		FQDN:        "postgres.interno",
		TailscaleIP: "100.64.0.16",
		Status:      "online",
	}

	ep, err := ValidateEndpoint(valid)
	if err != nil {
		t.Fatalf("expected valid endpoint, got error: %v", err)
	}
	if ep.FQDN != "postgres.interno" || ep.TailscaleIP.String() != "100.64.0.16" {
		t.Errorf("unexpected endpoint result: %+v", ep)
	}

	invalidDomain := cloud.NetworkEndpoint{
		Hostname:    "google",
		FQDN:        "google.com",
		TailscaleIP: "100.64.0.16",
	}
	if _, err := ValidateEndpoint(invalidDomain); err == nil {
		t.Error("expected error for non-.interno domain, got nil")
	}

	invalidIP := cloud.NetworkEndpoint{
		Hostname:    "invalid-ip",
		FQDN:        "invalid.interno",
		TailscaleIP: "http://100.64.0.16",
	}
	if _, err := ValidateEndpoint(invalidIP); err == nil {
		t.Error("expected error for invalid IP format, got nil")
	}

	multilineIP := cloud.NetworkEndpoint{
		Hostname:    "meu-arquivos",
		FQDN:        "meu-arquivos.interno",
		TailscaleIP: "100.64.0.6\nfd7a:115c:a1e0::6",
		Status:      "online",
	}
	epMulti, err := ValidateEndpoint(multilineIP)
	if err != nil {
		t.Fatalf("expected multiline IP endpoint to be valid, got error: %v", err)
	}
	if epMulti.TailscaleIP.String() != "100.64.0.6" {
		t.Errorf("expected extracted IP '100.64.0.6', got %q", epMulti.TailscaleIP.String())
	}
}

func TestEndpointRegistryConcurrency(t *testing.T) {
	registry := NewEndpointRegistry(nil)

	var wg sync.WaitGroup
	// Writers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(v int) {
			defer wg.Done()
			eps := []cloud.NetworkEndpoint{
				{
					Hostname:    "postgres",
					FQDN:        "postgres.interno",
					TailscaleIP: fmt.Sprintf("100.64.0.%d", v+1),
					Status:      "online",
				},
				{
					Hostname:    "redis",
					FQDN:        "redis.interno",
					TailscaleIP: "100.64.0.17",
					Status:      "offline",
				},
			}
			registry.Update(int64(v), eps)
		}(i)
	}

	// Readers
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = registry.GetByFQDN("POSTGRES.INTERNO.")
			_, _ = registry.GetByFQDN("redis.interno")
			_ = registry.All()
			_ = registry.Version()
		}()
	}

	wg.Wait()
}
