package server

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/ubuntu/ubuntu-insights/internal/fileutils"
)

type Server struct {
	configManager *ConfigManager
}

func NewServer(configManager *ConfigManager) Server {
	return Server{
		configManager: configManager,
	}
}

func (h Server) UploadHandler(w http.ResponseWriter, r *http.Request) {
	app := r.PathValue("app")

	if len(app) < 1 {
		http.Error(w, "Invalid application name in URL", http.StatusBadRequest)
		return
	}

	h.configManager.lock.RLock()
	allowed := false
	for _, allowedApp := range h.configManager.GetAllowList() {
		if allowedApp == app {
			allowed = true
			break
		}
	}
	h.configManager.lock.RUnlock()

	if !allowed {
		http.Error(w, "Invalid application name in URL", http.StatusBadRequest)
		return
	}

	jsonFile, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving the file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer jsonFile.Close()

	jsonData, err := io.ReadAll(jsonFile)
	if err != nil {
		http.Error(w, "Error reading the file: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate file size (max 1MB)
	if header.Size > 100<<10 {
		http.Error(w, "File exceeds size limit", http.StatusRequestEntityTooLarge)
		return
	}

	h.configManager.lock.RLock()
	baseDir := h.configManager.GetBaseDir()
	h.configManager.lock.RUnlock()

	targetDir := filepath.Join(baseDir, app)
	if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
		http.Error(w, "Error creating directory: " + err.Error(), http.StatusInternalServerError)
		return
	}

	timestamp := time.Now().Format("20060102_150405")
	safeFilename := fmt.Sprintf("%s_%s", timestamp, header.Filename)
	targetPath := filepath.Join(targetDir, safeFilename)
	
	if err := fileutils.AtomicWrite(targetPath, jsonData); err != nil {
		http.Error(w, "Error saving file: " + err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("Saved file", "app", app, "target", targetPath)
	w.WriteHeader(http.StatusCreated)
}

func (h Server) VersionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"version":"1.0.0"}`)
}
