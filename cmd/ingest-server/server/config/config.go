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

type Config struct {
	BaseDir     string   `json:"base_dir"`
	AllowedList []string `json:"allowList"`
}

type ConfigManager struct {
	config     Config
	Lock       sync.RWMutex
	configPath string
}

func New(path string) *ConfigManager {
	return &ConfigManager{configPath: path}
}

var (
	config     Config
	lock       sync.RWMutex
	configPath = "config.json"
)

func (cm *ConfigManager) Load() error {
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

func (cm *ConfigManager) Watch() {
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

func (cm *ConfigManager) GetBaseDir() string {
	cm.Lock.RLock()
	defer cm.Lock.RUnlock()
	return cm.config.BaseDir
}

func (cm *ConfigManager) GetAllowList() []string {
	cm.Lock.RLock()
	defer cm.Lock.RUnlock()
	return cm.config.AllowedList
}
