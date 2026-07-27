package forwarding

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type ForwardingStorage interface {
	Load(sessionID string) ([]Forwarding, error)
	Save(sessionID string, items []Forwarding) error
	Delete(sessionID string) error
	LoadLegacy() ([]Forwarding, error)
	SaveLegacy(items []Forwarding) error
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

func (s *JSONStorage) readAllMapLocked() (map[string][]Forwarding, error) {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string][]Forwarding), nil
		}
		return nil, fmt.Errorf("failed to read forwardings file %s: %w", s.filePath, err)
	}

	if len(data) == 0 {
		return make(map[string][]Forwarding), nil
	}

	var allMap map[string][]Forwarding
	if err := json.Unmarshal(data, &allMap); err == nil {
		return allMap, nil
	}

	// Legacy format: JSON array
	var legacyItems []Forwarding
	if err := json.Unmarshal(data, &legacyItems); err == nil {
		allMap = make(map[string][]Forwarding)
		allMap["default"] = legacyItems
		return allMap, nil
	}

	return nil, fmt.Errorf("failed to parse forwardings JSON")
}

func (s *JSONStorage) saveAllMapLocked(allMap map[string][]Forwarding) error {
	data, err := json.MarshalIndent(allMap, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal forwardings map: %w", err)
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

func (s *JSONStorage) Load(sessionID string) ([]Forwarding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sessionID == "" {
		sessionID = "default"
	}

	allMap, err := s.readAllMapLocked()
	if err != nil {
		return nil, err
	}

	items, exists := allMap[sessionID]
	if !exists {
		items = DefaultForwardings()
		allMap[sessionID] = items
		_ = s.saveAllMapLocked(allMap)
		return items, nil
	}

	items = mergeDefaults(items)
	return items, nil
}

func (s *JSONStorage) Save(sessionID string, items []Forwarding) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sessionID == "" {
		sessionID = "default"
	}

	allMap, err := s.readAllMapLocked()
	if err != nil {
		allMap = make(map[string][]Forwarding)
	}

	allMap[sessionID] = items
	return s.saveAllMapLocked(allMap)
}

func (s *JSONStorage) Delete(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sessionID == "" {
		sessionID = "default"
	}

	allMap, err := s.readAllMapLocked()
	if err != nil {
		return nil
	}

	delete(allMap, sessionID)
	return s.saveAllMapLocked(allMap)
}

func (s *JSONStorage) LoadLegacy() ([]Forwarding, error) {
	return s.Load("default")
}

func (s *JSONStorage) SaveLegacy(items []Forwarding) error {
	return s.Save("default", items)
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
