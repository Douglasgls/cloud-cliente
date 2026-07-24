package forwarding

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"cloud-client/pkg/logger"
)

var (
	ErrDefaultServiceProtected = errors.New("default service cannot be deleted")
	ErrInvalidPort             = errors.New("port must be between 1 and 65535")
	ErrEmptyName               = errors.New("name cannot be empty")
	ErrNotFound                = errors.New("forwarding not found")
)

type ForwardingService interface {
	Subscribe(l ForwardingListener)
	Unsubscribe(l ForwardingListener)

	List() []ForwardingState
	Get(id string) (ForwardingState, error)
	Add(name string, remotePort, localPort int) (Forwarding, error)
	Update(f Forwarding) error
	Delete(id string) error
	Toggle(id string, enabled bool) error

	StartAll(targetHost string, dialer Dialer) error
	StopAll() error
	IsConnected() bool
}

type Service struct {
	storage    ForwardingStorage
	log        *logger.Logger
	items      []Forwarding
	proxies    map[string]*TCPProxy
	lastErrors map[string]string

	targetHost string
	dialer     Dialer

	listeners []ForwardingListener
	mu        sync.RWMutex
}

func NewService(storage ForwardingStorage, log *logger.Logger) (*Service, error) {
	items, err := storage.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load forwardings: %w", err)
	}

	return &Service{
		storage:    storage,
		log:        log,
		items:      items,
		proxies:    make(map[string]*TCPProxy),
		lastErrors: make(map[string]string),
	}, nil
}

func (s *Service) Subscribe(l ForwardingListener) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listeners = append(s.listeners, l)
}

func (s *Service) Unsubscribe(l ForwardingListener) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, listener := range s.listeners {
		if listener == l {
			s.listeners = append(s.listeners[:i], s.listeners[i+1:]...)
			break
		}
	}
}

func (s *Service) notifyStarted(id string) {
	for _, l := range s.listeners {
		l.OnForwardingStarted(id)
	}
}

func (s *Service) notifyStopped(id string) {
	for _, l := range s.listeners {
		l.OnForwardingStopped(id)
	}
}

func (s *Service) notifyError(id string, err error) {
	for _, l := range s.listeners {
		l.OnForwardingError(id, err)
	}
}

func (s *Service) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.targetHost != "" && s.dialer != nil
}

func (s *Service) List() []ForwardingState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]ForwardingState, len(s.items))
	for i, item := range s.items {
		proxy := s.proxies[item.ID]
		running := proxy != nil && proxy.IsRunning()
		errStr := s.lastErrors[item.ID]

		result[i] = ForwardingState{
			Forwarding: item,
			Running:    running,
			LastError:  errStr,
		}
	}
	return result
}

func (s *Service) Get(id string) (ForwardingState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, item := range s.items {
		if item.ID == id {
			proxy := s.proxies[item.ID]
			running := proxy != nil && proxy.IsRunning()
			errStr := s.lastErrors[item.ID]
			return ForwardingState{
				Forwarding: item,
				Running:    running,
				LastError:  errStr,
			}, nil
		}
	}
	return ForwardingState{}, ErrNotFound
}

func (s *Service) Add(name string, remotePort, localPort int) (Forwarding, error) {
	if name == "" {
		return Forwarding{}, ErrEmptyName
	}
	if err := validatePort(remotePort); err != nil {
		return Forwarding{}, fmt.Errorf("invalid remote port: %w", err)
	}
	if err := validatePort(localPort); err != nil {
		return Forwarding{}, fmt.Errorf("invalid local port: %w", err)
	}

	s.mu.Lock()

	newID := generateCustomID()
	item := Forwarding{
		ID:         newID,
		Name:       name,
		RemotePort: remotePort,
		LocalPort:  localPort,
		Enabled:    true,
		IsDefault:  false,
	}

	s.items = append(s.items, item)
	if err := s.storage.Save(s.items); err != nil {
		s.items = s.items[:len(s.items)-1]
		s.mu.Unlock()
		return Forwarding{}, err
	}

	targetHost := s.targetHost
	dialer := s.dialer
	s.mu.Unlock()

	if targetHost != "" && dialer != nil {
		s.startProxyFor(item, targetHost, dialer)
	}

	return item, nil
}

func (s *Service) Update(f Forwarding) error {
	if f.Name == "" {
		return ErrEmptyName
	}
	if err := validatePort(f.RemotePort); err != nil {
		return fmt.Errorf("invalid remote port: %w", err)
	}
	if err := validatePort(f.LocalPort); err != nil {
		return fmt.Errorf("invalid local port: %w", err)
	}

	s.mu.Lock()
	idx := -1
	for i, item := range s.items {
		if item.ID == f.ID {
			idx = i
			break
		}
	}

	if idx == -1 {
		s.mu.Unlock()
		return ErrNotFound
	}

	existing := s.items[idx]
	if existing.IsDefault {
		// Keep default fields fixed
		f.Name = existing.Name
		f.RemotePort = existing.RemotePort
		f.IsDefault = true
	}

	s.items[idx] = f
	if err := s.storage.Save(s.items); err != nil {
		s.items[idx] = existing
		s.mu.Unlock()
		return err
	}

	targetHost := s.targetHost
	dialer := s.dialer
	s.mu.Unlock()

	s.stopProxyFor(f.ID)
	if targetHost != "" && dialer != nil && f.Enabled {
		s.startProxyFor(f, targetHost, dialer)
	}

	return nil
}

func (s *Service) Delete(id string) error {
	s.mu.Lock()
	idx := -1
	for i, item := range s.items {
		if item.ID == id {
			idx = i
			break
		}
	}

	if idx == -1 {
		s.mu.Unlock()
		return ErrNotFound
	}

	if s.items[idx].IsDefault {
		s.mu.Unlock()
		return ErrDefaultServiceProtected
	}

	deletedItem := s.items[idx]
	s.items = append(s.items[:idx], s.items[idx+1:]...)

	if err := s.storage.Save(s.items); err != nil {
		s.items = append(s.items[:idx], append([]Forwarding{deletedItem}, s.items[idx:]...)...)
		s.mu.Unlock()
		return err
	}

	s.mu.Unlock()
	s.stopProxyFor(id)
	return nil
}

func (s *Service) Toggle(id string, enabled bool) error {
	s.mu.Lock()
	idx := -1
	for i, item := range s.items {
		if item.ID == id {
			idx = i
			break
		}
	}

	if idx == -1 {
		s.mu.Unlock()
		return ErrNotFound
	}

	s.items[idx].Enabled = enabled
	item := s.items[idx]

	if err := s.storage.Save(s.items); err != nil {
		s.items[idx].Enabled = !enabled
		s.mu.Unlock()
		return err
	}

	targetHost := s.targetHost
	dialer := s.dialer
	s.mu.Unlock()

	if enabled {
		if targetHost != "" && dialer != nil {
			s.startProxyFor(item, targetHost, dialer)
		}
	} else {
		s.stopProxyFor(id)
	}

	return nil
}

func (s *Service) StartAll(targetHost string, dialer Dialer) error {
	cleanHost := strings.TrimSpace(targetHost)
	if lines := strings.Split(cleanHost, "\n"); len(lines) > 0 {
		cleanHost = strings.TrimSpace(lines[0])
	}
	s.mu.Lock()
	s.targetHost = cleanHost
	s.dialer = dialer
	items := make([]Forwarding, len(s.items))
	copy(items, s.items)
	s.mu.Unlock()

	for _, item := range items {
		if item.Enabled {
			s.startProxyFor(item, cleanHost, dialer)
		}
	}
	return nil
}

func (s *Service) StopAll() error {
	s.mu.Lock()
	s.targetHost = ""
	s.dialer = nil
	ids := make([]string, 0, len(s.proxies))
	for id := range s.proxies {
		ids = append(ids, id)
	}
	s.mu.Unlock()

	for _, id := range ids {
		s.stopProxyFor(id)
	}
	return nil
}

func (s *Service) startProxyFor(item Forwarding, targetHost string, dialer Dialer) {
	s.stopProxyFor(item.ID)

	proxy := NewTCPProxy(item.ID, item.LocalPort, targetHost, item.RemotePort, dialer, s.log)
	if err := proxy.Start(); err != nil {
		s.mu.Lock()
		s.lastErrors[item.ID] = err.Error()
		s.mu.Unlock()
		s.notifyError(item.ID, err)
		return
	}

	s.mu.Lock()
	s.proxies[item.ID] = proxy
	delete(s.lastErrors, item.ID)
	s.mu.Unlock()

	s.notifyStarted(item.ID)
}

func (s *Service) stopProxyFor(id string) {
	s.mu.Lock()
	proxy := s.proxies[id]
	delete(s.proxies, id)
	delete(s.lastErrors, id)
	s.mu.Unlock()

	if proxy != nil {
		_ = proxy.Stop()
		s.notifyStopped(id)
	}
}

func validatePort(port int) error {
	if port < 1 || port > 65535 {
		return ErrInvalidPort
	}
	return nil
}

func generateCustomID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("custom-%d-%s", time.Now().Unix(), hex.EncodeToString(b))
}
