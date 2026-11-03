package bridge

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"cloud-client/internal/config"
	"cloud-client/internal/forwarding"
	"cloud-client/internal/session"
)

type mockSessionService struct {
	sessions []session.Session
}

func (m *mockSessionService) HasSession() bool           { return len(m.sessions) > 0 }
func (m *mockSessionService) Load() (*session.Session, error) {
	if len(m.sessions) > 0 {
		return &m.sessions[0], nil
	}
	return nil, session.ErrNoSession
}
func (m *mockSessionService) Save(token string) error { return nil }
func (m *mockSessionService) Delete() error {
	m.sessions = nil
	return nil
}
func (m *mockSessionService) Touch() error { return nil }
func (m *mockSessionService) List() ([]session.Session, error) {
	return m.sessions, nil
}
func (m *mockSessionService) Get(id string) (*session.Session, error) {
	for _, s := range m.sessions {
		if s.ID == id {
			return &s, nil
		}
	}
	return nil, fmt.Errorf("session not found")
}
func (m *mockSessionService) SaveWithDetails(token, containerName, hostname, tailscaleIP string) (*session.Session, error) {
	s := &session.Session{
		ID:            "sess-new",
		AccessToken:   token,
		ContainerName: containerName,
		Hostname:      hostname,
		TailscaleIP:   tailscaleIP,
		CreatedAt:     time.Now(),
		LastUsedAt:    time.Now(),
	}
	m.sessions = append(m.sessions, *s)
	return s, nil
}
func (m *mockSessionService) DeleteSession(id string) error {
	var filtered []session.Session
	for _, s := range m.sessions {
		if s.ID != id {
			filtered = append(filtered, s)
		}
	}
	m.sessions = filtered
	return nil
}

type mockForwardingService struct {
	items []forwarding.ForwardingState
}

func (m *mockForwardingService) Subscribe(l forwarding.ForwardingListener)   {}
func (m *mockForwardingService) Unsubscribe(l forwarding.ForwardingListener) {}
func (m *mockForwardingService) List() []forwarding.ForwardingState {
	return m.items
}
func (m *mockForwardingService) Get(id string) (forwarding.ForwardingState, error) {
	for _, item := range m.items {
		if item.Forwarding.ID == id {
			return item, nil
		}
	}
	return forwarding.ForwardingState{}, forwarding.ErrNotFound
}
func (m *mockForwardingService) Add(name string, remotePort, localPort int) (forwarding.Forwarding, error) {
	fwd := forwarding.Forwarding{
		ID:         "custom-id",
		Name:       name,
		RemotePort: remotePort,
		LocalPort:  localPort,
		Enabled:    true,
	}
	m.items = append(m.items, forwarding.ForwardingState{Forwarding: fwd, Running: true})
	return fwd, nil
}
func (m *mockForwardingService) Update(f forwarding.Forwarding) error { return nil }
func (m *mockForwardingService) Delete(id string) error {
	var remaining []forwarding.ForwardingState
	for _, item := range m.items {
		if item.Forwarding.ID != id {
			remaining = append(remaining, item)
		}
	}
	m.items = remaining
	return nil
}
func (m *mockForwardingService) Toggle(id string, enabled bool) error {
	for i := range m.items {
		if m.items[i].Forwarding.ID == id {
			m.items[i].Forwarding.Enabled = enabled
			m.items[i].Running = enabled
			return nil
		}
	}
	return forwarding.ErrNotFound
}
func (m *mockForwardingService) SwitchSession(sessionID string) error            { return nil }
func (m *mockForwardingService) DeleteSessionForwardings(sessionID string) error { return nil }
func (m *mockForwardingService) ActiveSessionID() string                         { return "default" }
func (m *mockForwardingService) StartAll(targetHost string, dialer forwarding.Dialer) error {
	return nil
}
func (m *mockForwardingService) StopAll() error     { return nil }
func (m *mockForwardingService) IsConnected() bool  { return true }

func TestBridgeApp_SessionAndForwardingDTOs(t *testing.T) {
	now := time.Now()
	mockSess := &mockSessionService{
		sessions: []session.Session{
			{
				ID:            "sess-1",
				AccessToken:   "token-abc",
				ContainerName: "test-container",
				Hostname:      "test-host",
				TailscaleIP:   "100.64.0.5",
				CreatedAt:     now,
				LastUsedAt:    now,
			},
		},
	}

	mockFwd := &mockForwardingService{
		items: []forwarding.ForwardingState{
			{
				Forwarding: forwarding.Forwarding{
					ID:         "ssh",
					Name:       "SSH",
					RemotePort: 22,
					LocalPort:  2222,
					Enabled:    true,
					IsDefault:  true,
				},
				Running: true,
			},
		},
	}

	app := NewApp(
		&config.Config{},
		nil,
		nil,
		nil,
		nil,
		mockFwd,
		nil,
		mockSess,
		nil,
		nil,
	)

	// 1. Session DTO mapping
	t.Run("ListSessions returns correctly formatted DTOs", func(t *testing.T) {
		assert.True(t, app.HasSession())
		sessions := app.ListSessions()
		require.Len(t, sessions, 1)
		assert.Equal(t, "sess-1", sessions[0].ID)
		assert.Equal(t, "test-container", sessions[0].ContainerName)
		assert.Equal(t, "100.64.0.5", sessions[0].TailscaleIP)
		assert.NotEmpty(t, sessions[0].CreatedAt)
	})

	// 2. Forwarding DTO mapping
	t.Run("ListForwardings returns correctly formatted DTOs", func(t *testing.T) {
		fwds := app.ListForwardings()
		require.Len(t, fwds, 1)
		assert.Equal(t, "ssh", fwds[0].ID)
		assert.Equal(t, "SSH", fwds[0].Name)
		assert.Equal(t, 2222, fwds[0].LocalPort)
		assert.True(t, fwds[0].Running)
	})

	// 3. Add Forwarding
	t.Run("AddForwarding converts inputs and returns DTO", func(t *testing.T) {
		fwdDTO, err := app.AddForwarding("Database", 5432, 15432)
		require.NoError(t, err)
		assert.Equal(t, "Database", fwdDTO.Name)
		assert.Equal(t, 5432, fwdDTO.RemotePort)
		assert.Equal(t, 15432, fwdDTO.LocalPort)
		assert.True(t, fwdDTO.Enabled)
	})

	// 4. Toggle Forwarding
	t.Run("ToggleForwarding updates state", func(t *testing.T) {
		err := app.ToggleForwarding("ssh", false)
		require.NoError(t, err)
	})

	// 5. Connect input validation
	t.Run("Connect with empty token returns descriptive error", func(t *testing.T) {
		err := app.Connect("")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "token")
	})

	// 6. Delete Session
	t.Run("DeleteSession empties sessions", func(t *testing.T) {
		err := app.DeleteSession()
		require.NoError(t, err)
		assert.False(t, app.HasSession())
	})
}
