package session

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

var (
	ErrNoSession = errors.New("no session found")
)

type SessionService interface {
	HasSession() bool
	Load() (*Session, error)
	Save(token string) error
	Delete() error
	Touch() error

	List() ([]Session, error)
	Get(id string) (*Session, error)
	SaveWithDetails(token string, containerName string, hostname string, tailscaleIP string) (*Session, error)
	DeleteSession(id string) error
}

type Service struct {
	storage Storage
	mu      sync.Mutex
}

func NewService(storage Storage) *Service {
	return &Service{
		storage: storage,
	}
}

func (s *Service) HasSession() bool {
	if s.storage == nil || !s.storage.Exists() {
		return false
	}
	sessions, err := s.storage.LoadAll()
	return err == nil && len(sessions) > 0
}

func (s *Service) Load() (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.storage == nil || !s.storage.Exists() {
		return nil, ErrNoSession
	}

	return s.storage.Load()
}

func (s *Service) Save(token string) error {
	_, err := s.SaveWithDetails(token, "", "", "")
	return err
}

func (s *Service) SaveWithDetails(token string, containerName string, hostname string, tailscaleIP string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if token == "" {
		return nil, errors.New("cannot save empty token")
	}

	if s.storage == nil {
		return nil, errors.New("storage not initialized")
	}

	sessions, _ := s.storage.LoadAll()
	if sessions == nil {
		sessions = []Session{}
	}

	now := time.Now()
	id := deriveSessionID(hostname, token)

	var target *Session
	idx := -1

	for i, sess := range sessions {
		if (sess.ID != "" && sess.ID == id) ||
			(hostname != "" && sess.Hostname == hostname) ||
			sess.AccessToken == token {
			idx = i
			target = &sessions[i]
			break
		}
	}

	if target != nil {
		target.AccessToken = token
		if containerName != "" {
			target.ContainerName = containerName
		}
		if hostname != "" {
			target.Hostname = hostname
		}
		if tailscaleIP != "" {
			target.TailscaleIP = tailscaleIP
		}
		if target.ID == "" {
			target.ID = id
		}
		target.LastUsedAt = now
		sessions[idx] = *target
	} else {
		name := containerName
		if name == "" {
			name = hostname
		}
		newSess := Session{
			ID:            id,
			AccessToken:   token,
			ContainerName: name,
			Hostname:      hostname,
			TailscaleIP:   tailscaleIP,
			CreatedAt:     now,
			LastUsedAt:    now,
		}
		sessions = append(sessions, newSess)
		target = &newSess
	}

	if err := s.storage.SaveAll(sessions); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return target, nil
}

func (s *Service) List() ([]Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.storage == nil || !s.storage.Exists() {
		return []Session{}, nil
	}

	sessions, err := s.storage.LoadAll()
	if err != nil {
		return []Session{}, err
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].LastUsedAt.After(sessions[j].LastUsedAt)
	})

	return sessions, nil
}

func (s *Service) Get(id string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.storage == nil {
		return nil, ErrNoSession
	}

	sessions, err := s.storage.LoadAll()
	if err != nil {
		return nil, err
	}

	for _, sess := range sessions {
		if sess.ID == id || sess.Hostname == id || sess.AccessToken == id {
			return &sess, nil
		}
	}

	return nil, ErrNoSession
}

func (s *Service) DeleteSession(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.storage == nil {
		return nil
	}

	sessions, err := s.storage.LoadAll()
	if err != nil {
		return nil
	}

	updated := make([]Session, 0, len(sessions))
	for _, sess := range sessions {
		if sess.ID != id && sess.Hostname != id && sess.AccessToken != id {
			updated = append(updated, sess)
		}
	}

	if len(updated) == 0 {
		return s.storage.Delete()
	}

	return s.storage.SaveAll(updated)
}

func (s *Service) Delete() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.storage == nil {
		return nil
	}

	return s.storage.Delete()
}

func (s *Service) Touch() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.storage == nil || !s.storage.Exists() {
		return ErrNoSession
	}

	sess, err := s.storage.Load()
	if err != nil {
		return err
	}

	sess.LastUsedAt = time.Now()
	return s.storage.Save(sess)
}

func deriveSessionID(hostname string, token string) string {
	if hostname != "" {
		return hostname
	}
	h := sha256.Sum256([]byte(token))
	return "sess-" + hex.EncodeToString(h[:8])
}
