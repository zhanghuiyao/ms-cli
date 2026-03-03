package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/vigo999/ms-cli/agent/context"
)

// Session represents a persisted conversation session.
type Session struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	WorkDir     string                 `json:"work_dir"`
	ContextData []byte                 `json:"context_data"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// Manager handles session persistence.
type Manager struct {
	sessionsDir string
}

// NewManager creates a new session manager.
func NewManager(sessionsDir string) (*Manager, error) {
	if sessionsDir == "" {
		home, _ := os.UserHomeDir()
		sessionsDir = filepath.Join(home, ".config", "ms-cli", "sessions")
	}

	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		return nil, fmt.Errorf("create sessions directory: %w", err)
	}

	return &Manager{sessionsDir: sessionsDir}, nil
}

// Save saves a session to disk.
func (m *Manager) Save(session *Session) error {
	session.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	path := m.sessionPath(session.ID)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write session file: %w", err)
	}

	return nil
}

// Load loads a session from disk.
func (m *Manager) Load(sessionID string) (*Session, error) {
	path := m.sessionPath(sessionID)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("session not found: %s", sessionID)
		}
		return nil, fmt.Errorf("read session file: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}

	return &session, nil
}

// List returns all available sessions.
func (m *Manager) List() ([]*Session, error) {
	entries, err := os.ReadDir(m.sessionsDir)
	if err != nil {
		return nil, fmt.Errorf("read sessions directory: %w", err)
	}

	var sessions []*Session
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		sessionID := entry.Name()[:len(entry.Name())-5] // Remove .json
		session, err := m.Load(sessionID)
		if err != nil {
			continue // Skip invalid sessions
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// Delete removes a session.
func (m *Manager) Delete(sessionID string) error {
	path := m.sessionPath(sessionID)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("session not found: %s", sessionID)
		}
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// Create creates a new session with the given name.
func (m *Manager) Create(name, workDir string) *Session {
	return &Session{
		ID:        generateID(),
		Name:      name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		WorkDir:   workDir,
		Metadata:  make(map[string]interface{}),
	}
}

// SaveContext saves context manager state to session.
func (s *Session) SaveContext(ctx *context.Manager) error {
	data, err := ctx.Save()
	if err != nil {
		return err
	}
	s.ContextData = data
	return nil
}

// RestoreContext restores context manager state from session.
func (s *Session) RestoreContext(ctx *context.Manager) error {
	if len(s.ContextData) == 0 {
		return nil
	}
	return ctx.Load(s.ContextData)
}

// sessionPath returns the file path for a session.
func (m *Manager) sessionPath(sessionID string) string {
	return filepath.Join(m.sessionsDir, sessionID+".json")
}

// generateID generates a unique session ID.
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// Summary returns a human-readable session summary.
func (s *Session) Summary() string {
	return fmt.Sprintf("%s (%s) - %s", s.Name, s.ID[:8], s.UpdatedAt.Format("2006-01-02 15:04"))
}
