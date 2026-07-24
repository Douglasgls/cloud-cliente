package session

import (
	"path/filepath"
	"testing"
	"time"
)

func TestSessionService_SaveLoadTouchDelete(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "session.json")

	storage, err := NewJSONStorage(filePath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	service := NewService(storage)

	// 1. Initially no session
	if service.HasSession() {
		t.Errorf("Expected HasSession() to be false initially")
	}

	_, err = service.Load()
	if err == nil {
		t.Errorf("Expected Load() to fail when no session exists")
	}

	// 2. Save token
	testToken := "my-secret-access-token"
	if err := service.Save(testToken); err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	if !service.HasSession() {
		t.Errorf("Expected HasSession() to be true after Save()")
	}

	sess, err := service.Load()
	if err != nil {
		t.Fatalf("Failed to load session: %v", err)
	}

	if sess.AccessToken != testToken {
		t.Errorf("Expected token %s, got %s", testToken, sess.AccessToken)
	}

	if sess.CreatedAt.IsZero() || sess.LastUsedAt.IsZero() {
		t.Errorf("Expected non-zero timestamps: %+v", sess)
	}

	// 3. Touch session
	time.Sleep(10 * time.Millisecond)
	if err := service.Touch(); err != nil {
		t.Fatalf("Failed to touch session: %v", err)
	}

	updated, err := service.Load()
	if err != nil {
		t.Fatalf("Failed to load updated session: %v", err)
	}

	if !updated.LastUsedAt.After(sess.LastUsedAt) {
		t.Errorf("Expected LastUsedAt to be updated: old=%v, new=%v", sess.LastUsedAt, updated.LastUsedAt)
	}

	// 4. Delete session
	if err := service.Delete(); err != nil {
		t.Fatalf("Failed to delete session: %v", err)
	}

	if service.HasSession() {
		t.Errorf("Expected HasSession() to be false after Delete()")
	}
}
