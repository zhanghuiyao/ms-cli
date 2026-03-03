package config

import (
	"log"
	"os"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// Watcher handles configuration file changes.
type Watcher struct {
	path      string
	config    *Config
	mu        sync.RWMutex
	onChange  func(*Config)
	watcher   *fsnotify.Watcher
	stopCh    chan bool
}

// NewWatcher creates a new configuration watcher.
func NewWatcher(path string, onChange func(*Config)) (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	
	cw := &Watcher{
		path:     path,
		onChange: onChange,
		watcher:  watcher,
		stopCh:   make(chan bool),
	}
	
	// Load initial config
	cfg, err := LoadFromFile(path)
	if err != nil {
		cfg = DefaultConfig()
	}
	cw.config = cfg
	
	return cw, nil
}

// Start begins watching for configuration changes.
func (cw *Watcher) Start() error {
	if err := cw.watcher.Add(cw.path); err != nil {
		return err
	}
	
	go cw.watch()
	return nil
}

// Stop stops watching for changes.
func (cw *Watcher) Stop() {
	close(cw.stopCh)
	cw.watcher.Close()
}

// GetConfig returns the current configuration (thread-safe).
func (cw *Watcher) GetConfig() *Config {
	cw.mu.RLock()
	defer cw.mu.RUnlock()
	return cw.config
}

func (cw *Watcher) watch() {
	for {
		select {
		case event, ok := <-cw.watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				cw.reload()
			}
		case err, ok := <-cw.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("config watcher error: %v", err)
		case <-cw.stopCh:
			return
		}
	}
}

func (cw *Watcher) reload() {
	log.Printf("Reloading configuration from %s", cw.path)
	
	cfg, err := LoadFromFile(cw.path)
	if err != nil {
		log.Printf("Failed to reload config: %v", err)
		return
	}
	
	if err := cfg.Validate(); err != nil {
		log.Printf("Invalid config: %v", err)
		return
	}
	
	cw.mu.Lock()
	oldConfig := cw.config
	cw.config = cfg
	cw.mu.Unlock()
	
	log.Printf("Configuration reloaded successfully")
	
	// Notify callback
	if cw.onChange != nil {
		cw.onChange(oldConfig)
	}
}
