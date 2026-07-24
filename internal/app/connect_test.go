package app

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"cloud-client/internal/cloud"
	"cloud-client/internal/forwarding"
	"cloud-client/internal/session"
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
				TailscaleIP:  "100.64.0.2",
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

	tempDir := t.TempDir()
	storage, _ := forwarding.NewJSONStorage(filepath.Join(tempDir, "forwardings.json"))
	fwdSvc, _ := forwarding.NewService(storage, log)
	dialer := forwarding.NewDirectDialer()
	sStorage, _ := session.NewJSONStorage(filepath.Join(tempDir, "session.json"))
	sessionSvc := session.NewService(sStorage)

	uc := NewConnectUseCase(mockCloud, mockTS, fwdSvc, dialer, sessionSvc, log)

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

	tempDir := t.TempDir()
	storage, _ := forwarding.NewJSONStorage(filepath.Join(tempDir, "forwardings.json"))
	fwdSvc, _ := forwarding.NewService(storage, log)
	dialer := forwarding.NewDirectDialer()
	sStorage, _ := session.NewJSONStorage(filepath.Join(tempDir, "session.json"))
	sessionSvc := session.NewService(sStorage)

	uc := NewConnectUseCase(mockCloud, mockTS, fwdSvc, dialer, sessionSvc, log)
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
