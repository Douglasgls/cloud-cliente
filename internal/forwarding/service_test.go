package forwarding

import (
	"context"
	"net"
	"path/filepath"
	"testing"
)

type MockDialer struct{}

func (m *MockDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	c1, _ := net.Pipe()
	return c1, nil
}

type TestListener struct {
	started []string
	stopped []string
	errors  map[string]error
}

func NewTestListener() *TestListener {
	return &TestListener{
		errors: make(map[string]error),
	}
}

func (tl *TestListener) OnForwardingStarted(id string) {
	tl.started = append(tl.started, id)
}

func (tl *TestListener) OnForwardingStopped(id string) {
	tl.stopped = append(tl.stopped, id)
}

func (tl *TestListener) OnForwardingError(id string, err error) {
	tl.errors[id] = err
}

func TestService_CRUD(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "forwardings.json")

	storage, _ := NewJSONStorage(filePath)
	service, err := NewService(storage, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	listener := NewTestListener()
	service.Subscribe(listener)

	// List initially
	items := service.List()
	if len(items) != 3 {
		t.Fatalf("Expected 3 default items, got %d", len(items))
	}

	// Add custom service
	added, err := service.Add("PostgreSQL", 5432, 15432)
	if err != nil {
		t.Fatalf("Failed to add custom service: %v", err)
	}
	if added.Name != "PostgreSQL" || added.RemotePort != 5432 || added.LocalPort != 15432 {
		t.Errorf("Unexpected added item: %+v", added)
	}

	// Cannot delete default service
	err = service.Delete("ssh")
	if err == nil || err != ErrDefaultServiceProtected {
		t.Errorf("Expected ErrDefaultServiceProtected, got %v", err)
	}

	// Toggle custom service
	err = service.Toggle(added.ID, false)
	if err != nil {
		t.Fatalf("Failed to toggle service: %v", err)
	}

	state, err := service.Get(added.ID)
	if err != nil || state.Forwarding.Enabled {
		t.Errorf("Expected service to be disabled, state: %+v, err: %v", state, err)
	}

	// Delete custom service
	err = service.Delete(added.ID)
	if err != nil {
		t.Fatalf("Failed to delete custom service: %v", err)
	}

	if len(service.List()) != 3 {
		t.Errorf("Expected 3 items after deletion, got %d", len(service.List()))
	}
}

func TestService_SwitchSession(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "forwardings.json")

	storage, _ := NewJSONStorage(filePath)
	service, _ := NewService(storage, nil)

	// Switch to container-a and add Redis
	_ = service.SwitchSession("container-a")
	_, err := service.Add("Redis", 6379, 6379)
	if err != nil {
		t.Fatalf("Failed to add Redis to container-a: %v", err)
	}

	if len(service.List()) != 4 {
		t.Errorf("Expected 4 items for container-a, got %d", len(service.List()))
	}

	// Switch to container-b
	_ = service.SwitchSession("container-b")

	// Container B should have only 3 default items
	itemsB := service.List()
	if len(itemsB) != 3 {
		t.Errorf("Expected 3 default items for container-b, got %d", len(itemsB))
	}

	// Switch back to container-a
	_ = service.SwitchSession("container-a")
	itemsA := service.List()
	if len(itemsA) != 4 {
		t.Errorf("Expected 4 items when switching back to container-a, got %d", len(itemsA))
	}
}

func TestService_StartStopAll(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "forwardings.json")

	storage, _ := NewJSONStorage(filePath)
	service, _ := NewService(storage, nil)

	listener := NewTestListener()
	service.Subscribe(listener)

	dialer := &MockDialer{}

	// Assign high random ports for testing to avoid conflicts with running dev servers
	_ = service.Update(Forwarding{ID: "ssh", Name: "SSH", RemotePort: 22, LocalPort: 42222, Enabled: true, IsDefault: true})
	_ = service.Update(Forwarding{ID: "http", Name: "HTTP", RemotePort: 80, LocalPort: 48080, Enabled: true, IsDefault: true})
	_ = service.Update(Forwarding{ID: "https", Name: "HTTPS", RemotePort: 443, LocalPort: 48443, Enabled: true, IsDefault: true})

	err := service.StartAll("127.0.0.1", dialer)
	if err != nil {
		t.Fatalf("Failed to start all: %v", err)
	}

	if !service.IsConnected() {
		t.Errorf("Expected service to be connected")
	}

	// All 3 defaults should be running
	items := service.List()
	for _, item := range items {
		if !item.Running {
			t.Errorf("Expected forwarding %s to be running", item.Forwarding.ID)
		}
	}

	err = service.StopAll()
	if err != nil {
		t.Fatalf("Failed to stop all: %v", err)
	}

	if service.IsConnected() {
		t.Errorf("Expected service to be disconnected")
	}
}
