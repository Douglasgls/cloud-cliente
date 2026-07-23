package app

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"cloud-client/internal/cloud"
	"cloud-client/pkg/logger"
)

type mockCloudClient struct {
	connectFunc func(ctx context.Context, token string) (*cloud.ConnectResponse, error)
	confirmFunc func(ctx context.Context, connectionID string) (*cloud.ConfirmResponse, error)
}

func (m *mockCloudClient) Connect(ctx context.Context, token string) (*cloud.ConnectResponse, error) {
	if m.connectFunc != nil {
		return m.connectFunc(ctx, token)
	}
	return nil, nil
}

func (m *mockCloudClient) Confirm(ctx context.Context, connectionID string) (*cloud.ConfirmResponse, error) {
	if m.confirmFunc != nil {
		return m.confirmFunc(ctx, connectionID)
	}
	return nil, nil
}

type mockTailscaleService struct {
	upFunc      func(ctx context.Context, loginServer, authKey, hostname string) error
	statusFunc  func(ctx context.Context) (string, error)
	versionFunc func(ctx context.Context) (string, error)
}

func (m *mockTailscaleService) Up(ctx context.Context, loginServer, authKey, hostname string) error {
	if m.upFunc != nil {
		return m.upFunc(ctx, loginServer, authKey, hostname)
	}
	return nil
}

func (m *mockTailscaleService) Status(ctx context.Context) (string, error) {
	if m.statusFunc != nil {
		return m.statusFunc(ctx)
	}
	return "", nil
}

func (m *mockTailscaleService) Version(ctx context.Context) (string, error) {
	if m.versionFunc != nil {
		return m.versionFunc(ctx)
	}
	return "", nil
}

type mockProxyService struct {
	startFunc func(target string) (string, error)
	stopFunc  func(ctx context.Context) error
}

func (m *mockProxyService) Start(target string) (string, error) {
	if m.startFunc != nil {
		return m.startFunc(target)
	}
	return "http://127.0.0.1:8080", nil
}

func (m *mockProxyService) Stop(ctx context.Context) error {
	if m.stopFunc != nil {
		return m.stopFunc(ctx)
	}
	return nil
}

func (m *mockProxyService) TargetURL() string {
	return "http://100.64.0.2"
}

func (m *mockProxyService) LocalURL() string {
	return "http://127.0.0.1:8080"
}

type mockBrowserOpener struct {
	openFunc func(url string) error
}

func (m *mockBrowserOpener) Open(url string) error {
	if m.openFunc != nil {
		return m.openFunc(url)
	}
	return nil
}

func TestConnectUseCase_Success(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	log := logger.NewWithWriters(out, errOut, false)

	mockCloud := &mockCloudClient{
		connectFunc: func(ctx context.Context, token string) (*cloud.ConnectResponse, error) {
			return &cloud.ConnectResponse{
				LoginServer:  "https://headscale.example.com",
				PreauthKey:   "key123",
				Hostname:     "host-a",
				ConnectionID: "conn-123",
			}, nil
		},
		confirmFunc: func(ctx context.Context, connectionID string) (*cloud.ConfirmResponse, error) {
			return &cloud.ConfirmResponse{Success: true}, nil
		},
	}

	tailscaleCalled := false
	mockTS := &mockTailscaleService{
		upFunc: func(ctx context.Context, loginServer, authKey, hostname string) error {
			tailscaleCalled = true
			if loginServer != "https://headscale.example.com" || authKey != "key123" || hostname != "host-a" {
				t.Errorf("unexpected tailscale args: %s %s %s", loginServer, authKey, hostname)
			}
			return nil
		},
	}

	mockProxy := &mockProxyService{}
	mockBrowser := &mockBrowserOpener{}

	uc := NewConnectUseCase(mockCloud, mockTS, mockProxy, mockBrowser, log)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := uc.Execute(ctx, "valid-token")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !tailscaleCalled {
		t.Error("expected Tailscale.Up to be called")
	}
}

func TestConnectUseCase_CloudConnectError(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	log := logger.NewWithWriters(out, errOut, false)

	expectedErr := errors.New("Invalid token.")
	mockCloud := &mockCloudClient{
		connectFunc: func(ctx context.Context, token string) (*cloud.ConnectResponse, error) {
			return nil, expectedErr
		},
	}

	tsCalled := false
	mockTS := &mockTailscaleService{
		upFunc: func(ctx context.Context, loginServer, authKey, hostname string) error {
			tsCalled = true
			return nil
		},
	}

	mockProxy := &mockProxyService{}
	mockBrowser := &mockBrowserOpener{}

	uc := NewConnectUseCase(mockCloud, mockTS, mockProxy, mockBrowser, log)
	err := uc.Execute(context.Background(), "bad-token")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected %v, got %v", expectedErr, err)
	}
	if tsCalled {
		t.Error("Tailscale.Up should NOT have been called when cloud connect fails")
	}
}
