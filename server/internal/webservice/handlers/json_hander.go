package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ubuntu/ubuntu-insights/common/fileutils"
	"github.com/ubuntu/ubuntu-insights/server/internal/common/config"
)

type jsonHandler struct {
	config        config.Provider
	reportsDir    string
	maxUploadSize int64
	successStatus int
}

func (h *jsonHandler) serveHTTP(w http.ResponseWriter, r *http.Request, reqID string, app string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !h.config.Allows(app) {
		http.Error(w, "Invalid application name in URL", http.StatusForbidden)
		slog.Error("Invalid application name in URL", "req_id", reqID, "app", app)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, h.maxUploadSize)
	jsonData, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		slog.Error("Error reading the file", "req_id", reqID, "app", app, "err", err)
		return
	}
	if !json.Valid(jsonData) {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		slog.Error("Invalid JSON in uploaded file", "req_id", reqID, "app", app)
		return
	}

	targetDir := filepath.Join(h.reportsDir, app)
	if err := os.MkdirAll(targetDir, 0750); err != nil {
		http.Error(w, "Error creating directory", http.StatusInternalServerError)
		slog.Error("Error creating directory", "req_id", reqID, "app", app, "target", targetDir, "err", err)
		return
	}

	safeFilename := fmt.Sprintf("%s.json", reqID)
	targetPath := filepath.Join(targetDir, safeFilename)

	if err := fileutils.AtomicWrite(targetPath, jsonData); err != nil {
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		slog.Error("Error saving file", "req_id", reqID, "app", app, "target", targetPath, "err", err)
		return
	}

	slog.Info("File successfully uploaded", "req_id", reqID, "app", app, "target", targetPath)
	w.WriteHeader(h.successStatus)
}
