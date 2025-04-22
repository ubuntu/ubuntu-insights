// Package handlers provides HTTP handlers for the server.
package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"

	"github.com/google/uuid"
	"github.com/ubuntu/ubuntu-insights/internal/fileutils"
	"github.com/ubuntu/ubuntu-insights/internal/server/shared/config"
)

// Upload is a handler for uploading JSON files.
type Upload struct {
	config        config.Provider
	maxUploadSize int64
}

// NewUpload creates a new Upload handler.
func NewUpload(cfg config.Provider, maxUploadSize int64) *Upload {
	return &Upload{
		config:        cfg,
		maxUploadSize: maxUploadSize,
	}
}

// ServeHTTP handles the HTTP request for uploading JSON files.
func (h *Upload) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reqID := uuid.New().String()
	app := r.PathValue("app")

	slog.Info("Request recv'd", "req_id", reqID, "app", app)

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if len(app) < 1 {
		http.Error(w, "Invalid application name in URL", http.StatusForbidden)
		return
	}

	allowed := slices.Contains(h.config.AllowList(), app)

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

	if header.Size > h.maxUploadSize {
		http.Error(w, "File exceeds size limit", http.StatusRequestEntityTooLarge)
		slog.Error("File exceeds size limit", "req_id", reqID, "app", app, "size", header.Size)
		return
	}

	jsonData, err := io.ReadAll(jsonFile)
	if err != nil {
		http.Error(w, "Error reading the file: "+err.Error(), http.StatusBadRequest)
		slog.Error("Error reading the file", "req_id", reqID, "app", app, "err", err)
		return
	}

	if !json.Valid(jsonData) {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		slog.Error("Invalid JSON in uploaded file", "req_id", reqID, "app", app)
		return
	}

	baseDir := h.config.BaseDir()

	targetDir := filepath.Join(baseDir, app)
	if err := os.MkdirAll(targetDir, 0750); err != nil {
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
