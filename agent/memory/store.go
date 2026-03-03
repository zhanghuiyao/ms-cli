package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Item represents a single memory item.
type Item struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	Category  string    `json:"category"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
	Accessed  time.Time `json:"accessed"`
	AccessCount int     `json:"access_count"`
}

// FileStore implements Store interface with JSON file persistence.
type FileStore struct {
	path      string
	maxItems  int
	maxBytes  int
	ttlHours  int

	data map[string]*Item
	mu   sync.RWMutex
}

// NewFileStore creates a new file-based memory store.
func NewFileStore(path string) (*FileStore, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create memory directory: %w", err)
	}

	s := &FileStore{
		path:     path,
		maxItems: 1000,
		maxBytes: 10 * 1024 * 1024, // 10MB
		ttlHours: 168,              // 7 days
		data:     make(map[string]*Item),
	}

	// Load existing data
	if _, err := os.Stat(path); err == nil {
		if err := s.load(); err != nil {
			return nil, fmt.Errorf("load memory: %w", err)
		}
	}

	return s, nil
}

// SetMaxItems sets maximum number of items.
func (s *FileStore) SetMaxItems(n int) {
	s.maxItems = n
}

// SetMaxBytes sets maximum storage size.
func (s *FileStore) SetMaxBytes(n int) {
	s.maxBytes = n
}

// SetTTL sets item TTL in hours.
func (s *FileStore) SetTTL(hours int) {
	s.ttlHours = hours
}

// Put stores a value by key.
func (s *FileStore) Put(key, value string) error {
	return s.PutWithCategory(key, value, "general", nil)
}

// PutWithCategory stores a value with category and tags.
func (s *FileStore) PutWithCategory(key, value, category string, tags []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	s.data[key] = &Item{
		Key:         key,
		Value:       value,
		Category:    category,
		Tags:        tags,
		CreatedAt:   now,
		Accessed:    now,
		AccessCount: 0,
	}

	// Enforce limits
	s.enforceLimits()

	// Persist
	return s.save()
}

// Get retrieves a value by key.
func (s *FileStore) Get(key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.data[key]
	if !ok {
		return "", fmt.Errorf("key not found: %s", key)
	}

	// Check TTL
	if s.ttlHours > 0 {
		maxAge := time.Duration(s.ttlHours) * time.Hour
		if time.Since(item.CreatedAt) > maxAge {
			delete(s.data, key)
			s.save()
			return "", fmt.Errorf("key expired: %s", key)
		}
	}

	// Update access stats
	item.Accessed = time.Now()
	item.AccessCount++

	return item.Value, nil
}

// Search finds items matching query.
func (s *FileStore) Search(query string, limit int) ([]RetrieveResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query = strings.ToLower(query)
	var matches []RetrieveResult

	for _, item := range s.data {
		// Check TTL
		if s.ttlHours > 0 {
			maxAge := time.Duration(s.ttlHours) * time.Hour
			if time.Since(item.CreatedAt) > maxAge {
				continue // Skip expired
			}
		}

		score := 0
		if strings.Contains(strings.ToLower(item.Key), query) {
			score += 10
		}
		if strings.Contains(strings.ToLower(item.Value), query) {
			score += 5
		}
		if strings.Contains(strings.ToLower(item.Category), query) {
			score += 3
		}
		for _, tag := range item.Tags {
			if strings.Contains(strings.ToLower(tag), query) {
				score += 7
			}
		}

		if score > 0 {
			matches = append(matches, RetrieveResult{
				Key:   item.Key,
				Value: item.Value,
				Score: score,
			})
		}
	}

	// Sort by score descending
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	if limit > 0 && len(matches) > limit {
		matches = matches[:limit]
	}

	return matches, nil
}

// Delete removes a key.
func (s *FileStore) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, key)
	return s.save()
}

// List returns all keys.
func (s *FileStore) List() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.data))
	for key := range s.data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// enforceLimits removes old items if limits exceeded.
func (s *FileStore) enforceLimits() {
	// Remove expired items first
	if s.ttlHours > 0 {
		maxAge := time.Duration(s.ttlHours) * time.Hour
		for key, item := range s.data {
			if time.Since(item.CreatedAt) > maxAge {
				delete(s.data, key)
			}
		}
	}

	// Remove oldest if too many items
	if s.maxItems > 0 && len(s.data) > s.maxItems {
		type kv struct {
			key   string
			item  *Item
		}
		items := make([]kv, 0, len(s.data))
		for k, v := range s.data {
			items = append(items, kv{k, v})
		}
		sort.Slice(items, func(i, j int) bool {
			return items[i].item.Accessed.Before(items[j].item.Accessed)
		})

		toRemove := len(s.data) - s.maxItems
		for i := 0; i < toRemove; i++ {
			delete(s.data, items[i].key)
		}
	}
}

// save persists to disk.
func (s *FileStore) save() error {
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

// load reads from disk.
func (s *FileStore) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.data)
}

// RetrieveResult is defined in retrieve.go
