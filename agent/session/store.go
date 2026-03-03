package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vigo999/ms-cli/agent/context"
)

// Session represents a conversation session.
type Session struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Context   *context.Manager  `json:"-"` // Not serialized directly
	Metadata  map[string]string `json:"metadata"`
}

// Store handles session persistence.
type Store struct {
	baseDir string
}

// NewStore creates a new session store.
func NewStore(baseDir string) (*Store, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create session directory: %w", err)
	}
	
	return &Store{baseDir: baseDir}, nil
}

// DefaultStore returns a store in the default location (~/.ms-cli/sessions).
func DefaultStore() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return NewStore(filepath.Join(home, ".ms-cli", "sessions"))
}

// Save persists a session to disk.
func (s *Store) Save(session *Session) error {
	session.UpdatedAt = time.Now()
	
	// Save metadata
	metaPath := filepath.Join(s.baseDir, session.ID+".json")
	metaData := struct {
		ID        string            `json:"id"`
		Name      string            `json:"name"`
		CreatedAt time.Time         `json:"created_at"`
		UpdatedAt time.Time         `json:"updated_at"`
		Metadata  map[string]string `json:"metadata"`
	}{
		ID:        session.ID,
		Name:      session.Name,
		CreatedAt: session.CreatedAt,
		UpdatedAt: session.UpdatedAt,
		Metadata:  session.Metadata,
	}
	
	data, err := json.MarshalIndent(metaData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session metadata: %w", err)
	}
	
	if err := os.WriteFile(metaPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write session metadata: %w", err)
	}
	
	// Save context if available
	if session.Context != nil {
		ctxPath := filepath.Join(s.baseDir, session.ID+"-context.json")
		ctxData, err := session.Context.Save()
		if err != nil {
			return fmt.Errorf("failed to save context: %w", err)
		}
		if err := os.WriteFile(ctxPath, ctxData, 0644); err != nil {
			return fmt.Errorf("failed to write context: %w", err)
		}
	}
	
	return nil
}

// Load loads a session from disk.
func (s *Store) Load(id string) (*Session, error) {
	metaPath := filepath.Join(s.baseDir, id+".json")
	
	data, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("session not found: %s", id)
		}
		return nil, fmt.Errorf("failed to read session: %w", err)
	}
	
	var meta struct {
		ID        string            `json:"id"`
		Name      string            `json:"name"`
		CreatedAt time.Time         `json:"created_at"`
		UpdatedAt time.Time         `json:"updated_at"`
		Metadata  map[string]string `json:"metadata"`
	}
	
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("failed to parse session: %w", err)
	}
	
	session := &Session{
		ID:        meta.ID,
		Name:      meta.Name,
		CreatedAt: meta.CreatedAt,
		UpdatedAt: meta.UpdatedAt,
		Metadata:  meta.Metadata,
	}
	
	// Load context if exists
	ctxPath := filepath.Join(s.baseDir, id+"-context.json")
	if _, err := os.Stat(ctxPath); err == nil {
		ctxData, err := os.ReadFile(ctxPath)
		if err == nil {
			ctx := context.NewManager(128000)
			if err := ctx.Load(ctxData); err == nil {
				session.Context = ctx
			}
		}
	}
	
	return session, nil
}

// List returns all session IDs.
func (s *Store) List() ([]string, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	
	var ids []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) == ".json" && !strings.HasSuffix(name, "-context.json") {
			ids = append(ids, strings.TrimSuffix(name, ".json"))
		}
	}
	
	return ids, nil
}

// Delete removes a session from disk.
func (s *Store) Delete(id string) error {
	metaPath := filepath.Join(s.baseDir, id+".json")
	ctxPath := filepath.Join(s.baseDir, id+"-context.json")
	
	os.Remove(metaPath)
	os.Remove(ctxPath)
	
	return nil
}

// Manager provides high-level session management.
type Manager struct {
	store       *Store
	current     *Session
	autoSave    bool
	saveInterval time.Duration
}

// NewManager creates a new session manager.
func NewManager(store *Store) *Manager {
	return &Manager{
		store:        store,
		autoSave:     true,
		saveInterval: 30 * time.Second,
	}
}

// Create creates a new session.
func (m *Manager) Create(name string) (*Session, error) {
	session := &Session{
		ID:        generateID(),
		Name:      name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Context:   context.NewManager(128000),
		Metadata:  make(map[string]string),
	}
	
	if err := m.store.Save(session); err != nil {
		return nil, err
	}
	
	m.current = session
	return session, nil
}

// Load loads an existing session.
func (m *Manager) Load(id string) (*Session, error) {
	session, err := m.store.Load(id)
	if err != nil {
		return nil, err
	}
	
	m.current = session
	return session, nil
}

// GetCurrent returns the current session.
func (m *Manager) GetCurrent() *Session {
	return m.current
}

// SaveCurrent saves the current session.
func (m *Manager) SaveCurrent() error {
	if m.current == nil {
		return fmt.Errorf("no current session")
	}
	return m.store.Save(m.current)
}

// generateID generates a unique session ID.
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
