package network

import (
	"context"
	"testing"
	"time"

	"cloud-client/internal/cloud"
)

type mockCloudClient struct {
	endpoints *cloud.EndpointsResponse
	err       error
}

func (m *mockCloudClient) Connect(ctx context.Context, token string) (*cloud.ConnectResponse, error) {
	return nil, nil
}

func (m *mockCloudClient) Confirm(ctx context.Context, connectionID string) (*cloud.ConfirmResponse, error) {
	return nil, nil
}

func (m *mockCloudClient) GetNetworkEndpoints(ctx context.Context, token string) (*cloud.EndpointsResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.endpoints != nil {
		return m.endpoints, nil
	}
	return &cloud.EndpointsResponse{
		Version: 1,
		Endpoints: []cloud.NetworkEndpoint{
			{
				Hostname:    "postgres",
				FQDN:        "postgres.interno",
				TailscaleIP: "100.64.0.16",
				Status:      "online",
			},
		},
	}, nil
}

type mockOSIntegration struct {
	enabled  bool
	disabled bool
}

func (m *mockOSIntegration) Enable(listenAddr string) error {
	m.enabled = true
	return nil
}

func (m *mockOSIntegration) Disable() error {
	m.disabled = true
	return nil
}

func TestNetworkManager(t *testing.T) {
	mockCloud := &mockCloudClient{}
	mockOS := &mockOSIntegration{}

	// Listen on high port for unit test to avoid needing root/admin
	listenAddr := "127.0.0.1:15353"
	registry := NewEndpointRegistry(nil)
	cloudSync := NewCloudSync(mockCloud, registry, "test-token", 100*time.Millisecond, nil)
	resolver := NewDNSResolver(registry, nil)
	dnsServer := NewDNSServer(listenAddr, resolver, nil)

	nm := NewNetworkManagerWithComponents(registry, cloudSync, dnsServer, mockOS, listenAddr, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := nm.Start(ctx, "test-token"); err != nil {
		t.Fatalf("unexpected error starting NetworkManager: %v", err)
	}

	if !mockOS.enabled {
		t.Error("expected OS integration to be enabled")
	}

	// Verify registry populated
	ep, ok := nm.Registry().GetByFQDN("postgres.interno")
	if !ok || ep.TailscaleIP.String() != "100.64.0.16" {
		t.Errorf("unexpected endpoint in registry: ok=%v ep=%+v", ok, ep)
	}

	if err := nm.Stop(); err != nil {
		t.Fatalf("unexpected error stopping NetworkManager: %v", err)
	}

	if !mockOS.disabled {
		t.Error("expected OS integration to be disabled")
	}
}
