package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ubuntu/ubuntu-insights/server/internal/webservice/handlers"
)

func TestVersionSuccess(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(handlers.VersionHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code 200 OK, got %d", status)
	}

	expectedContentType := "application/json"
	if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("Expected Content-Type %s, got %s", expectedContentType, contentType)
	}

	var js map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &js); err != nil {
		t.Errorf("Expected valid JSON response, got error: %v", err)
	}
}

func TestVersionMethodNotAllowed(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		method string
	}{
		"HEAD":    {method: http.MethodHead},
		"POST":    {method: http.MethodPost},
		"PUT":     {method: http.MethodPut},
		"PATCH":   {method: http.MethodPatch},
		"DELETE":  {method: http.MethodDelete},
		"CONNECT": {method: http.MethodConnect},
		"OPTIONS": {method: http.MethodOptions},
		"TRACE":   {method: http.MethodTrace},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(test.method, "/version", nil)
			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(handlers.VersionHandler)
			handler.ServeHTTP(rr, req)
			if status := rr.Code; status != http.StatusMethodNotAllowed {
				t.Errorf("Expected status code 405 Method Not Allowed, got %d", status)
			}
		})
	}
}
