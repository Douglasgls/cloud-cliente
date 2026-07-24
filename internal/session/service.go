package session

import (
	"errors"
	"fmt"
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
	if s.storage == nil {
		return false
	}
	return s.storage.Exists()
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
	s.mu.Lock()
	defer s.mu.Unlock()

	if token == "" {
		return errors.New("cannot save empty token")
	}

	now := time.Now()
	createdAt := now

	if s.storage.Exists() {
		existing, err := s.storage.Load()
		if err == nil && existing.AccessToken == token && !existing.CreatedAt.IsZero() {
			createdAt = existing.CreatedAt
		}
	}

	sess := &Session{
		AccessToken: token,
		CreatedAt:   createdAt,
		LastUsedAt:  now,
	}

	if err := s.storage.Save(sess); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
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
