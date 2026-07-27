package forwarding

import (
	"encoding/json"
	"fmt"
	"io"
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
	if err != nil || len(data) == 0 {
		bakData, bakErr := os.ReadFile(s.filePath + ".bak")
		if bakErr == nil && len(bakData) > 0 {
			data = bakData
		} else if err != nil {
			if os.IsNotExist(err) {
				return make(map[string][]Forwarding), nil
			}
			return nil, fmt.Errorf("failed to read forwardings file %s: %w", s.filePath, err)
		}
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

	// Try backup file if main unmarshal failed
	bakData, bakErr := os.ReadFile(s.filePath + ".bak")
	if bakErr == nil && len(bakData) > 0 && string(bakData) != string(data) {
		if err := json.Unmarshal(bakData, &allMap); err == nil {
			return allMap, nil
		}
	}

	return nil, fmt.Errorf("failed to parse forwardings JSON")
}

func (s *JSONStorage) saveAllMapLocked(allMap map[string][]Forwarding) error {
	data, err := json.MarshalIndent(allMap, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal forwardings map: %w", err)
	}

	return atomicWriteWithBackup(s.filePath, data, 0644)
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

func atomicWriteWithBackup(filePath string, data []byte, perm os.FileMode) error {
	bakFile := filePath + ".bak"
	tmpFile := filePath + ".tmp"

	f, err := os.OpenFile(tmpFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return fmt.Errorf("failed to open tmp file: %w", err)
	}

	if _, err := f.Write(data); err != nil {
		f.Close()
		_ = os.Remove(tmpFile)
		return fmt.Errorf("failed to write tmp file: %w", err)
	}

	if err := f.Sync(); err != nil {
		f.Close()
		_ = os.Remove(tmpFile)
		return fmt.Errorf("failed to sync tmp file to disk: %w", err)
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(tmpFile)
		return fmt.Errorf("failed to close tmp file: %w", err)
	}

	if info, err := os.Stat(filePath); err == nil && info.Size() > 0 {
		_ = copyFileContents(filePath, bakFile)
	}

	if err := os.Rename(tmpFile, filePath); err != nil {
		return fmt.Errorf("failed to rename tmp file: %w", err)
	}

	return nil
}

func copyFileContents(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
