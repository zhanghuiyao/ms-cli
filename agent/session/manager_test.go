package session

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vigo999/ms-cli/agent/context"
)

func TestManager(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test Create
	session := manager.Create("test-session", "/tmp")
	if session.Name != "test-session" {
		t.Errorf("expected name 'test-session', got '%s'", session.Name)
	}

	if session.WorkDir != "/tmp" {
		t.Errorf("expected workdir '/tmp', got '%s'", session.WorkDir)
	}

	// Test Save
	if err := manager.Save(session); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test Load
	loaded, err := manager.Load(session.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if loaded.Name != session.Name {
		t.Errorf("expected name '%s', got '%s'", session.Name, loaded.Name)
	}

	// Test List
	sessions, err := manager.List()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sessions) != 1 {
		t.Errorf("expected 1 session, got %d", len(sessions))
	}

	// Test Delete
	if err := manager.Delete(session.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify deletion
	_, err = manager.Load(session.ID)
	if err == nil {
		t.Error("expected error for deleted session")
	}
}

func TestManagerLoadNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	manager, _ := NewManager(tmpDir)

	_, err := manager.Load("nonexistent-id")
	if err == nil {
		t.Error("expected error for nonexistent session")
	}
}

func TestManagerDeleteNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	manager, _ := NewManager(tmpDir)

	err := manager.Delete("nonexistent-id")
	if err == nil {
		t.Error("expected error for nonexistent session")
	}
}

func TestSessionSaveContext(t *testing.T) {
	session := &Session{
		ID:       "test-id",
		Name:     "test",
		WorkDir:  "/tmp",
		Metadata: make(map[string]interface{}),
	}

	ctx := context.NewManager(1000)
	ctx.SetSystemMessage("System message")
	ctx.AddMessage("user", "Hello")

	if err := session.SaveContext(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(session.ContextData) == 0 {
		t.Error("expected context data to be saved")
	}
}

func TestSessionRestoreContext(t *testing.T) {
	session := &Session{
		ID:       "test-id",
		Name:     "test",
		WorkDir:  "/tmp",
		Metadata: make(map[string]interface{}),
	}

	// Save context
	ctx := context.NewManager(1000)
	ctx.SetSystemMessage("System message")
	ctx.AddMessage("user", "Hello")
	session.SaveContext(ctx)

	// Restore to new context
	newCtx := context.NewManager(1000)
	if err := session.RestoreContext(newCtx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify restored
	msgs := newCtx.BuildMessages()
	if len(msgs) == 0 {
		t.Error("expected messages to be restored")
	}
}

func TestSessionSummary(t *testing.T) {
	session := &Session{
		ID:   "123456789",
		Name: "test-session",
	}

	summary := session.Summary()
	if summary == "" {
		t.Error("expected non-empty summary")
	}

	// Should contain name and truncated ID
	if !contains(summary, "test-session") {
		t.Error("summary should contain session name")
	}
}

func TestDefaultManagerPath(t *testing.T) {
	manager, err := NewManager("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = manager // Avoid unused variable error

	// Should use default path
	home, _ := os.UserHomeDir()
	expectedPath := filepath.Join(home, ".config", "ms-cli", "sessions")

	// Verify directory was created
	if _, err := os.Stat(expectedPath); err != nil {
		t.Errorf("default sessions directory was not created: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsInternal(s, substr)))
}

func containsInternal(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
