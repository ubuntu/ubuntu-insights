package handlers

import (
	"fmt"
	"net/http"

	"github.com/ubuntu/ubuntu-insights/server/internal/common/constants"
	"github.com/ubuntu/ubuntu-insights/server/internal/webservice/metrics"
)

// VersionHandler handles requests to the /version endpoint.
func VersionHandler(w http.ResponseWriter, r *http.Request) {
	metrics.ApplyLabels(r)

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"version":"%s"}`, constants.Version)
}
