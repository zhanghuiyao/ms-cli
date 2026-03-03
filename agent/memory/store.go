package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Item represents a stored memory item.
type Item struct {
	ID        string            `json:"id"`
	Content   string            `json:"content"`
	Type      string            `json:"type"` // e.g., "fact", "preference", "context"
	Tags      []string          `json:"tags"`
	Metadata  map[string]string `json:"metadata"`
	CreatedAt time.Time         `json:"created_at"`
	ExpiresAt *time.Time        `json:"expires_at,omitempty"`
}

// IsExpired checks if the memory item has expired.
func (i *Item) IsExpired() bool {
	if i.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*i.ExpiresAt)
}

// Store handles memory persistence.
type Store struct {
	baseDir string
	maxSize int
}

// NewStore creates a new memory store.
func NewStore(baseDir string) (*Store, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create memory directory: %w", err)
	}
	
	return &Store{
		baseDir: baseDir,
		maxSize: 1000, // Default max items
	}, nil
}

// Save stores a memory item.
func (s *Store) Save(item *Item) error {
	if item.ID == "" {
		item.ID = generateMemoryID()
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}
	
	path := filepath.Join(s.baseDir, item.ID+".json")
	
	data, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal memory: %w", err)
	}
	
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write memory: %w", err)
	}
	
	return nil
}

// Load retrieves a memory item by ID.
func (s *Store) Load(id string) (*Item, error) {
	path := filepath.Join(s.baseDir, id+".json")
	
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("memory not found: %s", id)
		}
		return nil, fmt.Errorf("failed to read memory: %w", err)
	}
	
	var item Item
	if err := json.Unmarshal(data, &item); err != nil {
		return nil, fmt.Errorf("failed to parse memory: %w", err)
	}
	
	return &item, nil
}

// Search finds memories matching the query.
func (s *Store) Search(query string, tags []string) ([]*Item, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to list memories: %w", err)
	}
	
	var results []*Item
	
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		
		item, err := s.Load(entry.Name()[:len(entry.Name())-5])
		if err != nil {
			continue
		}
		
		// Skip expired items
		if item.IsExpired() {
			continue
		}
		
		// Match query
		if query != "" {
			if !contains(item.Content, query) {
				continue
			}
		}
		
		// Match tags
		if len(tags) > 0 {
			if !hasAnyTag(item.Tags, tags) {
				continue
			}
		}
		
		results = append(results, item)
	}
	
	return results, nil
}

// Delete removes a memory item.
func (s *Store) Delete(id string) error {
	path := filepath.Join(s.baseDir, id+".json")
	return os.Remove(path)
}

// Cleanup removes expired memory items.
func (s *Store) Cleanup() error {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return err
	}
	
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		id := entry.Name()[:len(entry.Name())-5]
		item, err := s.Load(id)
		if err != nil {
			continue
		}
		
		if item.IsExpired() {
			s.Delete(id)
		}
	}
	
	return nil
}

// Helper functions

func generateMemoryID() string {
	return fmt.Sprintf("mem-%d", time.Now().UnixNano())
}

func contains(s, substr string) bool {
	return len(substr) == 0 || len(s) > 0 && containsIgnoreCase(s, substr)
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    findSubstrIgnoreCase(s, substr) >= 0)
}

func findSubstrIgnoreCase(s, substr string) int {
	// Simple case-insensitive search
	sLower := toLower(s)
	subLower := toLower(substr)
	
	for i := 0; i <= len(sLower)-len(subLower); i++ {
		if sLower[i:i+len(subLower)] == subLower {
			return i
		}
	}
	return -1
}

func toLower(s string) string {
	// Simple ASCII lowercase
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c = c + ('a' - 'A')
		}
		result[i] = c
	}
	return string(result)
}

func hasAnyTag(itemTags, queryTags []string) bool {
	for _, qt := range queryTags {
		for _, it := range itemTags {
			if it == qt {
				return true
			}
		}
	}
	return false
}
