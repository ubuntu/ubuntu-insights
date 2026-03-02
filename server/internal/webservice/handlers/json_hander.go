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
	metrics.ApplyLabels(r)

	if r.Method != http.MethodPost {
		metrics.ApplyRejectReason(r, metrics.RejectReasonMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		slog.Debug("Request had invalid method", "req_id", reqID, "method", r.Method)
		return
	}

	if !h.config.IsAllowed(app) {
		metrics.ApplyRejectReason(r, metrics.RejectReasonForbidden)
		http.Error(w, "Invalid application name in URL", http.StatusForbidden)
		slog.Debug("Request had invalid application name in URL", "req_id", reqID, "app", app)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, h.maxUploadSize)
	jsonData, err := io.ReadAll(r.Body)
	if err != nil {
		metrics.ApplyRejectReason(r, metrics.RejectReasonUnreadablePayload)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		slog.Debug("Request had unreadable payload", "req_id", reqID, "app", app, "err", err)
		return
	}
	if !json.Valid(jsonData) {
		metrics.ApplyRejectReason(r, metrics.RejectReasonInvalidJSON)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		slog.Debug("Request had invalid JSON", "req_id", reqID, "app", app)
		return
	}

	targetDir := filepath.Join(h.reportsDir, app)
	if err := os.MkdirAll(targetDir, 0750); err != nil {
		metrics.ApplyRejectReason(r, metrics.RejectReasonInternalServerErr)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		slog.Error("Error creating directory", "req_id", reqID, "app", app, "target", targetDir, "err", err)
		return
	}

	safeFilename := fmt.Sprintf("%s.json", reqID)
	targetPath := filepath.Join(targetDir, safeFilename)

	if err := fileutils.AtomicWrite(targetPath, jsonData); err != nil {
		metrics.ApplyRejectReason(r, metrics.RejectReasonInternalServerErr)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		slog.Error("Error saving file", "req_id", reqID, "app", app, "target", targetPath, "err", err)
		return
	}

	slog.Debug("File successfully uploaded", "req_id", reqID, "app", app, "target", targetPath)
	w.WriteHeader(h.successStatus)
}
