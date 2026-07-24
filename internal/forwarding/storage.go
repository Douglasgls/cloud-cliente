package forwarding

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type ForwardingStorage interface {
	Load() ([]Forwarding, error)
	Save(items []Forwarding) error
}

type JSONStorage struct {
	filePath string
	mu       sync.Mutex
}

func NewJSONStorage(filePath string) (*JSONStorage, error) {
	if filePath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		filePath = filepath.Join(home, ".cloud-client", "forwardings.json")
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory %s: %w", dir, err)
	}

	return &JSONStorage{filePath: filePath}, nil
}

func (s *JSONStorage) FilePath() string {
	return s.filePath
}

func (s *JSONStorage) Load() ([]Forwarding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			defaults := DefaultForwardings()
			if err := s.saveLocked(defaults); err != nil {
				return nil, err
			}
			return defaults, nil
		}
		return nil, fmt.Errorf("failed to read forwardings file %s: %w", s.filePath, err)
	}

	var items []Forwarding
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("failed to parse forwardings JSON: %w", err)
	}

	// Ensure default services (ssh, http, https) are present
	items = mergeDefaults(items)
	return items, nil
}

func (s *JSONStorage) Save(items []Forwarding) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked(items)
}

func (s *JSONStorage) saveLocked(items []Forwarding) error {
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal forwardings: %w", err)
	}

	tmpFile := s.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	if err := os.Rename(tmpFile, s.filePath); err != nil {
		return fmt.Errorf("failed to replace forwardings file: %w", err)
	}

	return nil
}

func mergeDefaults(existing []Forwarding) []Forwarding {
	defaults := DefaultForwardings()
	existingMap := make(map[string]bool)

	for _, item := range existing {
		existingMap[item.ID] = true
	}

	result := make([]Forwarding, len(existing))
	copy(result, existing)

	for _, def := range defaults {
		if !existingMap[def.ID] {
			result = append(result, def)
		}
	}

	return result
}
