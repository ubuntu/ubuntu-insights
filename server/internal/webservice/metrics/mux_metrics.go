package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MuxMiddleware is a middleware for collecting metrics on HTTP request paths.
type MuxMiddleware struct {
	registry prometheus.Registerer
}

// NewMuxMiddleware creates a new MuxMiddleware instance with the provided registry.
func NewMuxMiddleware(registry prometheus.Registerer) *MuxMiddleware {
	return &MuxMiddleware{
		registry: registry,
	}
}

// Wrap is a middleware function that wraps an HTTP handler to collect metrics on request paths.
func (m *MuxMiddleware) Wrap(handlerName string, handler http.Handler) http.HandlerFunc {
	reg := prometheus.WrapRegistererWith(prometheus.Labels{"handler": handlerName}, m.registry)

	requestsTotal := promauto.With(reg).NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_mux_requests_total",
			Help: "Tracks the number of HTTP requests to the mux.",
		}, []string{"method", "code"},
	)

	return promhttp.InstrumentHandlerCounter(
		requestsTotal,
		handler,
	)
}
