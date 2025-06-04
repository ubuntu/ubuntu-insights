package handlers

import (
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/ubuntu/ubuntu-insights/internal/server/shared/config"
)

// LegacyReport is a handler for old Ubuntu-Report JSON reports.
//
// Reports of this style have the path values {distribution} and {version},
// and expect a http status code of 200 OK on success.
//
// The reports will be matched against apps whitelist in the format of
// "ubuntu-report/{distribution}/desktop/{version}".
type LegacyReport struct {
	jsonHandler *jsonHandler
}

// NewLegacyReport creates a new LegacyReport handler.
func NewLegacyReport(cfg config.Provider, reportsDir string, maxUploadSize int64) *LegacyReport {
	return &LegacyReport{
		jsonHandler: &jsonHandler{
			config:        cfg,
			reportsDir:    reportsDir,
			maxUploadSize: maxUploadSize,
			successStatus: http.StatusOK,
		}}
}

// ServeHTTP handles incoming HTTP requests for JSON report uploads.
func (h *LegacyReport) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reqID := uuid.New().String()
	distribution := filepath.Clean(r.PathValue("distribution"))
	if distribution == "" || distribution == "." || strings.Contains(distribution, "..") {
		http.Error(w, "Invalid distribution name in URL", http.StatusForbidden)
		return
	}

	version := filepath.Clean(r.PathValue("version"))
	if version == "" || version == "." || strings.Contains(version, "..") {
		http.Error(w, "Invalid version name in URL", http.StatusForbidden)
		return
	}

	app := "ubuntu-report/" + distribution + "/desktop/" + version
	slog.Info("Request recv'd", "req_id", reqID, "app", app)
	h.jsonHandler.serveHTTP(w, r, reqID, app)
}
