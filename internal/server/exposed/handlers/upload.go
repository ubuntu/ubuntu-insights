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
	"strings"

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
	app := filepath.Clean(r.PathValue("app"))
	if app == "" || app == "." || strings.Contains(app, "..") {
		http.Error(w, "Invalid application name in URL", http.StatusForbidden)
		return
	}

	slog.Info("Request recv'd", "req_id", reqID, "app", app)

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	allowed := slices.Contains(h.config.AllowList(), app)
	if !allowed {
		http.Error(w, "Invalid application name in URL", http.StatusForbidden)
		slog.Error("Invalid application name in URL", "req_id", reqID, "app", app)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, h.maxUploadSize)
	jsonData, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body: "+err.Error(), http.StatusBadRequest)
		slog.Error("Error reading the file", "req_id", reqID, "app", app, "err", err)
		return
	}
	if !json.Valid(jsonData) {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		slog.Error("Invalid JSON in uploaded file", "req_id", reqID, "app", app)
		return
	}

	targetDir := filepath.Join(h.config.BaseDir(), app)
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
	w.WriteHeader(http.StatusAccepted)
}
