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

	items, err := storage.Load()
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

func TestJSONStorage_SaveAndLoad(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "forwardings.json")

	storage, err := NewJSONStorage(filePath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	customItems := []Forwarding{
		{ID: "ssh", Name: "SSH", RemotePort: 22, LocalPort: 2222, Enabled: true, IsDefault: true},
		{ID: "custom-1", Name: "Redis", RemotePort: 6379, LocalPort: 6379, Enabled: true, IsDefault: false},
	}

	if err := storage.Save(customItems); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	loaded, err := storage.Load()
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// Loading should also merge missing default services (http, https)
	if len(loaded) != 4 {
		t.Errorf("Expected 4 items after merging defaults, got %d", len(loaded))
	}
}
