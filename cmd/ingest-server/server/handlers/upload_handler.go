package handlers

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/ubuntu/ubuntu-insights/cmd/ingest-server/server/config"
	"github.com/ubuntu/ubuntu-insights/internal/fileutils"
)

const maxUploadSize = 100 << 10 // 100 KB

var ConfigManager *config.ConfigManager

func UploadHandler(w http.ResponseWriter, r *http.Request) {
	app := r.PathValue("app")

	if len(app) < 1 {
		http.Error(w, "Invalid application name in URL", http.StatusBadRequest)
		return
	}

	ConfigManager.Lock.RLock()
	allowed := false
	for _, allowedApp := range ConfigManager.GetAllowList() {
		if allowedApp == app {
			allowed = true
			break
		}
	}
	ConfigManager.Lock.RUnlock()

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

	if header.Size > maxUploadSize {
		http.Error(w, "File exceeds size limit", http.StatusRequestEntityTooLarge)
		return
	}

	ConfigManager.Lock.RLock()
	baseDir := ConfigManager.GetBaseDir()
	ConfigManager.Lock.RUnlock()

	targetDir := filepath.Join(baseDir, app)
	if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
		http.Error(w, "Error creating directory: "+err.Error(), http.StatusInternalServerError)
		return
	}

	safeFilename := fmt.Sprintf("%s.json", uuid.New().String())
	targetPath := filepath.Join(targetDir, safeFilename)

	if err := fileutils.AtomicWrite(targetPath, jsonData); err != nil {
		http.Error(w, "Error saving file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("Saved file", "app", app, "target", targetPath)
	w.WriteHeader(http.StatusCreated)
}
