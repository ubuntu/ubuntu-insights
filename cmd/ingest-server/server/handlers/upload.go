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

type UploadHandler struct{
	Config config.ConfigProvider
}

func (h *UploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reqID := uuid.New().String()
	app := r.PathValue("app")

	slog.Info("Request recv'd", "req_id", reqID, "app", app)

	if len(app) < 1 {
		http.Error(w, "Invalid application name in URL", http.StatusBadRequest)
		return
	}

	allowed := false
	for _, allowedApp := range h.Config.GetAllowList() {
		if allowedApp == app {
			allowed = true
			break
		}
	}

	if !allowed {
		http.Error(w, "Invalid application name in URL", http.StatusForbidden)
		slog.Error("Invalid application name in URL", "req_id", reqID, "app", app)
		return
	}

	jsonFile, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving the file: "+err.Error(), http.StatusBadRequest)
		slog.Error("Error retrieving the file", "req_id", reqID, "app", app, "err", err)
		return
	}
	defer jsonFile.Close()

	jsonData, err := io.ReadAll(jsonFile)
	if err != nil {
		http.Error(w, "Error reading the file: "+err.Error(), http.StatusBadRequest)
		slog.Error("Error reading the file", "req_id", reqID, "app", app, "err", err)
		return
	}

	if header.Size > maxUploadSize {
		http.Error(w, "File exceeds size limit", http.StatusRequestEntityTooLarge)
		slog.Error("File exceeds size limit", "req_id", reqID, "app", app, "size", header.Size)
		return
	}

	baseDir := h.Config.GetBaseDir()

	targetDir := filepath.Join(baseDir, app)
	if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
		http.Error(w, "Error creating directory: "+err.Error(), http.StatusInternalServerError)
		slog.Error("Error creating directory", "req_id", reqID, "app", app, "target", targetDir, "err", err)
		return
	}

	safeFilename := fmt.Sprintf("%s.json", reqID)
	targetPath := filepath.Join(targetDir, safeFilename)

	if err := fileutils.AtomicWrite(targetPath, jsonData); err != nil {
		http.Error(w, "Error saving file: "+err.Error(), http.StatusInternalServerError)
		slog.Error("Error saving file", "req_id", reqID, "app", app, "target", targetPath, "err", err)
		return
	}

	slog.Info("File successfully uploaded", "req_id", reqID, "app", app, "target", targetPath)
	w.WriteHeader(http.StatusCreated)
}
