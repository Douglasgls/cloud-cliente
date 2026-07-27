package session

import (
	"os"
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

func TestSessionService_MultiSession(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "session.json")

	storage, err := NewJSONStorage(filePath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	service := NewService(storage)

	// Save Container A
	sessA, err := service.SaveWithDetails("token-a", "Container A", "ct-100", "100.64.0.1")
	if err != nil {
		t.Fatalf("Failed to save session A: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	// Save Container B
	sessB, err := service.SaveWithDetails("token-b", "Container B", "ct-101", "100.64.0.2")
	if err != nil {
		t.Fatalf("Failed to save session B: %v", err)
	}

	list, err := service.List()
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}

	if len(list) != 2 {
		t.Fatalf("Expected 2 sessions, got %d", len(list))
	}

	// Most recent session should be Container B
	if list[0].ID != sessB.ID {
		t.Errorf("Expected most recent session to be %s, got %s", sessB.ID, list[0].ID)
	}

	// Delete Container A
	if err := service.DeleteSession(sessA.ID); err != nil {
		t.Fatalf("Failed to delete session A: %v", err)
	}

	listAfterDelete, err := service.List()
	if err != nil {
		t.Fatalf("Failed to list sessions after delete: %v", err)
	}

	if len(listAfterDelete) != 1 {
		t.Fatalf("Expected 1 session after delete, got %d", len(listAfterDelete))
	}

	if listAfterDelete[0].ID != sessB.ID {
		t.Errorf("Expected remaining session to be %s, got %s", sessB.ID, listAfterDelete[0].ID)
	}
}

func TestSessionStorage_CorruptedFallback(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "session.json")

	storage, err := NewJSONStorage(filePath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	service := NewService(storage)
	_, err = service.SaveWithDetails("token-123", "Container Test", "ct-200", "100.64.0.10")
	if err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	// Second save to trigger backup creation
	if _, err := service.SaveWithDetails("token-456", "Container Test Updated", "ct-200", "100.64.0.10"); err != nil {
		t.Fatalf("Failed to update session: %v", err)
	}

	bakPath := filePath + ".bak"
	if _, err := os.Stat(bakPath); err != nil {
		t.Fatalf("Expected backup file %s to exist", bakPath)
	}

	// Corrupt main file
	if err := os.WriteFile(filePath, []byte("{ invalid json..."), 0644); err != nil {
		t.Fatalf("Failed to write corrupted file: %v", err)
	}

	// LoadAll should restore from .bak
	sessions, err := storage.LoadAll()
	if err != nil {
		t.Fatalf("Expected LoadAll to succeed using backup file, got error: %v", err)
	}

	if len(sessions) != 1 || sessions[0].AccessToken != "token-123" {
		t.Fatalf("Expected restored session token-123 from backup, got %+v", sessions)
	}
}
