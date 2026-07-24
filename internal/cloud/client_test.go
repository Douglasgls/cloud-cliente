package cloud

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_Connect_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/client/connect" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var req ConnectRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed decoding request: %v", err)
		}
		if req.Token != "valid-token" {
			t.Errorf("unexpected token: %s", req.Token)
		}

		resp := ConnectResponse{
			LoginServer:   "https://headscale.example.com",
			PreauthKey:    "key12345",
			Hostname:      "host-01",
			TailscaleIP:   "100.64.0.2",
			ConnectionID:  "conn-999",
			ExpiresAt:     "2026-12-31T23:59:59Z",
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClientWithHTTP(server.URL, server.Client(), "test-agent", nil)
	resp, err := client.Connect(context.Background(), "valid-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.LoginServer != "https://headscale.example.com" {
		t.Errorf("unexpected LoginServer: %s", resp.LoginServer)
	}
	if resp.ConnectionID != "conn-999" {
		t.Errorf("unexpected ConnectionID: %s", resp.ConnectionID)
	}
}

func TestClient_Connect_InvalidToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewClientWithHTTP(server.URL, server.Client(), "test-agent", nil)
	_, err := client.Connect(context.Background(), "bad-token")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errorsIs(err, ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestClient_Confirm_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/client/confirm" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var req ConfirmRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed decoding request: %v", err)
		}
		if req.ConnectionID != "conn-999" {
			t.Errorf("unexpected connection ID: %s", req.ConnectionID)
		}

		resp := ConfirmResponse{
			Success: true,
			Message: "Confirmed",
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClientWithHTTP(server.URL, server.Client(), "test-agent", nil)
	resp, err := client.Confirm(context.Background(), "conn-999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.Success {
		t.Errorf("expected success to be true")
	}
}

func errorsIs(err, target error) bool {
	return errors.Is(err, target)
}
