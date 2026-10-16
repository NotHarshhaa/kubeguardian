package config

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Reloader watches a config file for changes and reloads it automatically
type Reloader struct {
	mu        sync.RWMutex
	config    *Config
	configPath string
	watcher   *fsnotify.Watcher
	callbacks []func(*Config)
	stopCh    chan struct{}
	running   bool
}

// NewReloader creates a new config reloader that watches the given config file
func NewReloader(configPath string) (*Reloader, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	// Watch the directory containing the config file
	dir := filepath.Dir(configPath)
	if err := watcher.Add(dir); err != nil {
		watcher.Close()
		return nil, fmt.Errorf("failed to watch config directory: %w", err)
	}

	return &Reloader{
		configPath: configPath,
		watcher:    watcher,
		callbacks:  make([]func(*Config), 0),
		stopCh:     make(chan struct{}),
	}, nil
}

// SetConfig sets the current configuration
func (r *Reloader) SetConfig(cfg *Config) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.config = cfg
}

// GetConfig returns the current configuration
func (r *Reloader) GetConfig() *Config {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.config
}

// RegisterCallback registers a callback to be called when config is reloaded
func (r *Reloader) RegisterCallback(callback func(*Config)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.callbacks = append(r.callbacks, callback)
}

// Start starts watching the config file for changes
func (r *Reloader) Start() error {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return fmt.Errorf("reloader is already running")
	}
	r.running = true
	r.mu.Unlock()

	go r.watch()
	return nil
}

// Stop stops watching the config file
func (r *Reloader) Stop() error {
	r.mu.Lock()
	if !r.running {
		r.mu.Unlock()
		return nil
	}
	r.running = false
	close(r.stopCh)
	r.mu.Unlock()

	return r.watcher.Close()
}

// watch watches for config file changes and reloads when detected
func (r *Reloader) watch() {
	debounce := make(chan struct{}, 1)
	debounceTimer := time.NewTimer(0)
	if !debounceTimer.Stop() {
		select {
		case <-debounceTimer.C:
		default:
		}
	}

	for {
		select {
		case <-r.stopCh:
			debounceTimer.Stop()
			return

		case event, ok := <-r.watcher.Events:
			if !ok {
				return
			}

			// Check if the event is for our config file
			if filepath.Clean(event.Name) == filepath.Clean(r.configPath) {
				if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
					// Debounce rapid file changes
					select {
					case debounce <- struct{}{}:
					default:
					}

					// Reset debounce timer
					if !debounceTimer.Stop() {
						select {
						case <-debounceTimer.C:
						default:
						}
					}
					debounceTimer.Reset(500 * time.Millisecond)
				}
			}

		case <-debounceTimer.C:
			select {
			case <-debounce:
				r.reload()
			default:
			}

		case err, ok := <-r.watcher.Errors:
			if !ok {
				return
			}
			// Log watcher error but continue
			_ = err
		}
	}
}

// reload reloads the configuration from disk and invokes callbacks
func (r *Reloader) reload() {
	r.mu.RLock()
	configPath := r.configPath
	r.mu.RUnlock()

	// Load the new configuration
	newConfig, err := LoadConfig(configPath)
	if err != nil {
		// Log error but keep the old config
		fmt.Printf("Failed to reload config: %v\n", err)
		return
	}

	r.mu.Lock()
	oldConfig := r.config
	r.config = newConfig
	callbacks := make([]func(*Config), len(r.callbacks))
	copy(callbacks, r.callbacks)
	r.mu.Unlock()

	// Invoke callbacks with the new config
	for _, callback := range callbacks {
		callback(newConfig)
	}

	fmt.Printf("Configuration reloaded from %s\n", configPath)

	_ = oldConfig
}

// IsRunning returns true if the reloader is running
func (r *Reloader) IsRunning() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.running
}

// ValidateCurrentConfig validates the current configuration
func (r *Reloader) ValidateCurrentConfig() *ValidationResult {
	r.mu.RLock()
	cfg := r.config
	r.mu.RUnlock()

	if cfg == nil {
		return &ValidationResult{
			Valid:  false,
			Errors: []string{"no configuration loaded"},
		}
	}

	return cfg.Validate()
}
