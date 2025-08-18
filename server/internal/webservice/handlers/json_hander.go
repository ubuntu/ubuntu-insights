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
	"github.com/ubuntu/ubuntu-insights/server/internal/webservice/metrics"
)

type jsonHandler struct {
	config        ConfigProvider
	reportsDir    string
	maxUploadSize int64
	successStatus int
}

func (h *jsonHandler) serveHTTP(w http.ResponseWriter, r *http.Request, reqID string, app string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !h.config.IsAllowed(app) {
		http.Error(w, "Invalid application name in URL", http.StatusForbidden)
		slog.Debug("Request had invalid application name in URL", "req_id", reqID, "app", app)
		return
	}

	metrics.ApplyLabels(r)

	r.Body = http.MaxBytesReader(w, r.Body, h.maxUploadSize)
	jsonData, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		slog.Debug("Request had unreadable payload", "req_id", reqID, "app", app, "err", err)
		return
	}
	if !json.Valid(jsonData) {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		slog.Debug("Request had invalid JSON", "req_id", reqID, "app", app)
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
