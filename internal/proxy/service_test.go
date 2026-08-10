package proxy

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestProxyService_StartAndForward(t *testing.T) {
	// Create mock backend container HTTP server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/test" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("backend response ok"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer backend.Close()

	service := NewService(nil, "")
	localURL, err := service.Start(backend.URL)
	if err != nil {
		t.Fatalf("failed to start proxy service: %v", err)
	}

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = service.Stop(ctx)
	}()

	// Send HTTP request to local proxy URL
	resp, err := http.Get(localURL + "/test")
	if err != nil {
		t.Fatalf("failed to send request to local proxy: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected HTTP 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "backend response ok" {
		t.Errorf("unexpected body from proxy: %s", string(body))
	}
}
