package session

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

type Storage interface {
	LoadAll() ([]Session, error)
	SaveAll(sessions []Session) error
	Load() (*Session, error)
	Save(sess *Session) error
	Delete() error
	Exists() bool
}

type JSONStorage struct {
	filePath string
	mu       sync.Mutex
}

func NewJSONStorage(filePath string) (*JSONStorage, error) {
	if filePath == "" {
		var err error
		filePath, err = getDefaultSessionPath()
		if err != nil {
			return nil, fmt.Errorf("failed to determine default session path: %w", err)
		}
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create session directory %s: %w", dir, err)
	}

	return &JSONStorage{filePath: filePath}, nil
}

func (s *JSONStorage) FilePath() string {
	return s.filePath
}

func (s *JSONStorage) Exists() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	info, err := os.Stat(s.filePath)
	if err != nil {
		bakInfo, bakErr := os.Stat(s.filePath + ".bak")
		if bakErr != nil {
			return false
		}
		return !bakInfo.IsDir() && bakInfo.Size() > 0
	}
	return !info.IsDir() && info.Size() > 0
}

func (s *JSONStorage) LoadAll() ([]Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil || len(data) == 0 {
		// Attempt to load from backup file if main file is missing or empty
		bakData, bakErr := os.ReadFile(s.filePath + ".bak")
		if bakErr == nil && len(bakData) > 0 {
			data = bakData
		} else if err != nil {
			return nil, fmt.Errorf("failed to read session file: %w", err)
		}
	}

	if len(data) == 0 {
		return []Session{}, nil
	}

	var sessions []Session
	if err := json.Unmarshal(data, &sessions); err == nil {
		return sessions, nil
	}

	// Try legacy single session unmarshal for backward compatibility
	var single Session
	if err := json.Unmarshal(data, &single); err == nil && single.AccessToken != "" {
		if single.ID == "" {
			if single.Hostname != "" {
				single.ID = single.Hostname
			} else {
				single.ID = "default"
			}
		}
		return []Session{single}, nil
	}

	// If unmarshal failed, attempt fallback to .bak if we haven't tried it yet
	bakData, bakErr := os.ReadFile(s.filePath + ".bak")
	if bakErr == nil && len(bakData) > 0 && string(bakData) != string(data) {
		if err := json.Unmarshal(bakData, &sessions); err == nil {
			return sessions, nil
		}
	}

	return nil, fmt.Errorf("failed to parse session JSON")
}

func (s *JSONStorage) SaveAll(sessions []Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(sessions, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal sessions: %w", err)
	}

	return atomicWriteWithBackup(s.filePath, data, 0600)
}

func (s *JSONStorage) Load() (*Session, error) {
	sessions, err := s.LoadAll()
	if err != nil || len(sessions) == 0 {
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("no sessions found")
	}

	// Find most recently used session
	var mostRecent *Session
	for i := range sessions {
		if mostRecent == nil || sessions[i].LastUsedAt.After(mostRecent.LastUsedAt) {
			mostRecent = &sessions[i]
		}
	}
	return mostRecent, nil
}

func (s *JSONStorage) Save(sess *Session) error {
	if sess == nil || sess.AccessToken == "" {
		return fmt.Errorf("cannot save nil or empty session")
	}

	sessions, _ := s.LoadAll()
	if sessions == nil {
		sessions = []Session{}
	}

	updated := false
	for i, existing := range sessions {
		if (sess.ID != "" && existing.ID == sess.ID) ||
			(sess.Hostname != "" && existing.Hostname == sess.Hostname) ||
			existing.AccessToken == sess.AccessToken {
			sessions[i] = *sess
			updated = true
			break
		}
	}

	if !updated {
		sessions = append(sessions, *sess)
	}

	return s.SaveAll(sessions)
}

func (s *JSONStorage) Delete() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_ = os.Remove(s.filePath + ".bak")
	if err := os.Remove(s.filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete session file: %w", err)
	}
	return nil
}

func getDefaultSessionPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	if runtime.GOOS == "windows" {
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData != "" {
			return filepath.Join(localAppData, "CloudClient", "session.json"), nil
		}
	}

	return filepath.Join(home, ".cloud-client", "session.json"), nil
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
