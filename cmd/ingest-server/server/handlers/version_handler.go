package handlers

import (
	"fmt"
	"net/http"
)

func (h Server) VersionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"version":"1.0.0"}`)
}
