// Package config provides configuration management for the server.
package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// Provider defines the interface for configuration management.
type Provider interface {
	GetBaseDir() string
	GetAllowList() []string
}

// Config represents the configuration structure for the server.
type Config struct {
	BaseDir     string   `json:"base_dir"`
	AllowedList []string `json:"allowList"`
}

// Manager manages the configuration for the server.
type Manager struct {
	config     Config
	Lock       sync.RWMutex
	configPath string
}

// New creates a new ConfigManager with the specified configuration file path.
func New(path string) *Manager {
	return &Manager{configPath: path}
}

// Load reads the configuration from the specified JSON file.
func (cm *Manager) Load() error {
	file, err := os.Open(cm.configPath)
	if err != nil {
		return fmt.Errorf("opening config file: %w", err)
	}
	defer file.Close()

	var newConfig Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&newConfig); err != nil {
		return fmt.Errorf("decoding config JSON: %w", err)
	}

	cm.Lock.Lock()
	cm.config = newConfig
	cm.Lock.Unlock()

	slog.Info("Configuration loaded", "config", cm.config)
	return nil
}

// Watch starts watching the configuration file for changes.
// When a change is detected, it reloads the configuration.
func (cm *Manager) Watch() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Warn("Failed to create watcher", "err", err)
		return
	}
	defer watcher.Close()

	configDir, _ := filepath.Split(cm.configPath)
	if configDir == "" {
		configDir = "."
	}
	if err := watcher.Add(configDir); err != nil {
		slog.Warn("Failed to add directory to watcher", "dir", configDir, "err", err)
		return
	}

	slog.Info("Watching configuration directory", "dir", configDir)

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op != fsnotify.Write && event.Op != fsnotify.Create {
				continue
			}

			slog.Debug("Configuration file changed. Reloading...")

			if err := cm.Load(); err != nil {
				slog.Warn("Error reloading config", "err", err)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			slog.Warn("Watcher error", "err", err)
		}
	}
}

// GetBaseDir returns the base directory for file storage.
func (cm *Manager) GetBaseDir() string {
	cm.Lock.RLock()
	defer cm.Lock.RUnlock()
	return cm.config.BaseDir
}

// GetAllowList returns the allowed list of applications.
func (cm *Manager) GetAllowList() []string {
	cm.Lock.RLock()
	defer cm.Lock.RUnlock()
	return cm.config.AllowedList
}
