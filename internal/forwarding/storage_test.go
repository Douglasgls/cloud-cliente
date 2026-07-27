package forwarding

import (
	"path/filepath"
	"testing"
)

func TestJSONStorage_LoadDefaults(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "forwardings.json")

	storage, err := NewJSONStorage(filePath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	items, err := storage.Load("container-a")
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	if len(items) != 3 {
		t.Errorf("Expected 3 default items, got %d", len(items))
	}

	if items[0].ID != "ssh" || items[1].ID != "http" || items[2].ID != "https" {
		t.Errorf("Unexpected default IDs: %v", items)
	}
}

func TestJSONStorage_PerSessionIsolation(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "forwardings.json")

	storage, err := NewJSONStorage(filePath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	itemsA := []Forwarding{
		{ID: "ssh", Name: "SSH", RemotePort: 22, LocalPort: 2222, Enabled: true, IsDefault: true},
		{ID: "custom-1", Name: "Redis", RemotePort: 6379, LocalPort: 6379, Enabled: true, IsDefault: false},
	}

	if err := storage.Save("container-a", itemsA); err != nil {
		t.Fatalf("Failed to save container-a: %v", err)
	}

	itemsB := []Forwarding{
		{ID: "ssh", Name: "SSH", RemotePort: 22, LocalPort: 2223, Enabled: true, IsDefault: true},
		{ID: "custom-2", Name: "Postgres", RemotePort: 5432, LocalPort: 5432, Enabled: true, IsDefault: false},
	}

	if err := storage.Save("container-b", itemsB); err != nil {
		t.Fatalf("Failed to save container-b: %v", err)
	}

	loadedA, err := storage.Load("container-a")
	if err != nil {
		t.Fatalf("Failed to load container-a: %v", err)
	}

	loadedB, err := storage.Load("container-b")
	if err != nil {
		t.Fatalf("Failed to load container-b: %v", err)
	}

	if len(loadedA) != 4 { // SSH, Redis + merged HTTP, HTTPS
		t.Errorf("Expected 4 items for container-a, got %d", len(loadedA))
	}

	if len(loadedB) != 4 { // SSH (port 2223), Postgres + merged HTTP, HTTPS
		t.Errorf("Expected 4 items for container-b, got %d", len(loadedB))
	}

	// Verify custom-1 (Redis) is only in Container A
	hasRedisInB := false
	for _, item := range loadedB {
		if item.Name == "Redis" {
			hasRedisInB = true
		}
	}
	if hasRedisInB {
		t.Errorf("Container B should not contain Redis service from Container A")
	}
}
