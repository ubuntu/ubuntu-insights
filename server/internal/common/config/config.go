// Package config provides a configuration manager that loads and watches a JSON configuration file.
package config

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
)

var reservedNames = map[string]struct{}{
	"ubuntu_report":     {},
	"schema_migrations": {},
}

// Conf represents the configuration structure.
type Conf struct {
	AllowedList []string `json:"allowList"`
}

// Manager is a struct that manages the configuration.
type Manager struct {
	config     Conf
	allowSet   map[string]struct{}
	lock       sync.RWMutex
	configPath string

	log *slog.Logger
}

type options struct {
	Logger *slog.Logger
}

// Options represents an optional function to override Manager default values.
type Options func(*options)

// New creates a new configuration manager with the specified path.
func New(path string, args ...Options) *Manager {
	opts := options{
		Logger: slog.Default(),
	}

	for _, opt := range args {
		opt(&opts)
	}

	return &Manager{
		configPath: path,
		log:        opts.Logger,
	}
}

// Load reads the configuration from the specified file and updates the internal state.
// It filters out any reserved names from the allow list, logging a warning for each reserved name found.
func (cm *Manager) Load() error {
	file, err := os.Open(cm.configPath)
	if err != nil {
		return fmt.Errorf("opening config file: %w", err)
	}
	defer file.Close()

	var newConfig Conf
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&newConfig); err != nil {
		return fmt.Errorf("decoding config JSON: %w", err)
	}

	newConfig.AllowedList = cm.filterAllowList(newConfig.AllowedList)

	cm.lock.Lock()
	cm.config = newConfig
	cm.allowSet = cm.buildAllowSet()
	cm.lock.Unlock()

	cm.log.Info("Configuration loaded", "config", cm.config)
	return nil
}

// Watch starts watching the configuration file for changes.
//
// It returns two channels: one for configuration changes which result in a successful load and another for unrecoverable watcher errors.
func (cm *Manager) Watch(ctx context.Context) (changes <-chan struct{}, errors <-chan error, err error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create watcher: %v", err)
	}

	configDir, _ := filepath.Split(cm.configPath)
	if configDir == "" {
		configDir = "."
	}
	if err := watcher.Add(configDir); err != nil {
		watcher.Close()
		return nil, nil, fmt.Errorf("failed to add directory %s to watcher: %v", configDir, err)
	}

	cm.log.Info("Watching configuration directory", "dir", configDir)
	changesCh := make(chan struct{}, 1)
	errorsCh := make(chan error, 1)

	// Initial load of the configuration
	if err := cm.Load(); err != nil {
		cm.log.Warn("Error loading initial config", "err", err)
	}

	go func() {
		defer close(changesCh)
		defer close(errorsCh)
		defer watcher.Close()

		for {
			select {
			case <-ctx.Done():
				cm.log.Info("Configuration watcher stopped")
				return
			case event, ok := <-watcher.Events:
				if !ok {
					errorsCh <- fmt.Errorf("watcher events channel closed unexpectedly")
					return
				}
				if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
					continue
				}

				if event.Name != cm.configPath {
					continue
				}

				cm.log.Debug("Configuration file changed. Reloading...")
				if err := cm.Load(); err != nil {
					cm.log.Warn("Error reloading config", "err", err)
					continue
				}

				select {
				case changesCh <- struct{}{}:
				default:
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					errorsCh <- fmt.Errorf("watcher errors channel closed unexpectedly")
					return
				}
				cm.log.Warn("Watcher error", "err", err)
			}
		}
	}()

	return changesCh, errorsCh, nil
}

// AllowList returns a copy of the allow list from the configuration.
func (cm *Manager) AllowList() []string {
	cm.lock.RLock()
	defer cm.lock.RUnlock()
	allowListCopy := make([]string, len(cm.config.AllowedList))
	copy(allowListCopy, cm.config.AllowedList)
	return allowListCopy
}

// IsAllowed checks if the given value is in the allow set.
func (cm *Manager) IsAllowed(value string) bool {
	cm.lock.RLock()
	defer cm.lock.RUnlock()
	_, exists := cm.allowSet[value]
	return exists
}

// filterAllowList filters out reserved names from the allow list.
func (cm *Manager) filterAllowList(allowList []string) []string {
	filteredAllowList := make([]string, 0, len(allowList))
	for _, name := range allowList {
		if _, reserved := reservedNames[name]; reserved {
			cm.log.Warn("Reserved name in allow list, ignoring", "name", name)
			continue
		}
		filteredAllowList = append(filteredAllowList, name)
	}
	return filteredAllowList
}

// buildAllowSet builds a set from the allow list for faster lookups.
func (cm *Manager) buildAllowSet() map[string]struct{} {
	allowSet := make(map[string]struct{}, len(cm.config.AllowedList))
	for _, name := range cm.config.AllowedList {
		allowSet[name] = struct{}{}
	}
	return allowSet
}
