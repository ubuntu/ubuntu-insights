// Package handlers provides HTTP handlers for the server.
package handlers

import (
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/ubuntu/ubuntu-insights/server/internal/shared/config"
)

// Upload is a handler for uploading standard Ubuntu-Insights JSON reports.
type Upload struct {
	jsonHandler *jsonHandler
}

// NewUpload creates a new Upload handler.
func NewUpload(cfg config.Provider, reportsDir string, maxUploadSize int64) *Upload {
	return &Upload{
		jsonHandler: &jsonHandler{
			config:        cfg,
			reportsDir:    reportsDir,
			maxUploadSize: maxUploadSize,
			successStatus: http.StatusAccepted,
		}}
}

// ServeHTTP handles incoming HTTP requests for JSON report uploads.
func (h *Upload) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reqID := uuid.New().String()
	app := filepath.Clean(r.PathValue("app"))
	if app == "" || app == "." || strings.Contains(app, "..") {
		http.Error(w, "Invalid application name in URL", http.StatusForbidden)
		return
	}

	slog.Info("Request recv'd", "req_id", reqID, "app", app)
	h.jsonHandler.serveHTTP(w, r, reqID, app)
}
