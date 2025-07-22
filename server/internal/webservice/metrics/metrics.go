// Package metrics provides middleware for collecting metrics in the web service, to be interpreted by Prometheus.
package metrics

import (
	"context"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type label string

// LabelPath is the label used for the path in metrics.
const LabelPath label = "path"

// Middleware is a middleware for collecting HTTP request metrics.
type Middleware struct {
	buckets  []float64
	registry prometheus.Registerer
}

// New creates a new Middleware instance with the provided registry and buckets.
func New(registry prometheus.Registerer) *Middleware {
	return &Middleware{
		// Mainly used for HTTP request durations which will skew small unless something is wrong. Max of 10.24.
		buckets:  prometheus.ExponentialBuckets(0.005, 2, 12),
		registry: registry,
	}
}

// Monitor is a middleware function that wraps an HTTP handler to collect metrics.
func (m *Middleware) Monitor(handlerName string, handler http.Handler) http.HandlerFunc {
	reg := prometheus.WrapRegistererWith(prometheus.Labels{"handler": handlerName}, m.registry)
	labels := []string{"method", "code", string(LabelPath)}

	requestsTotal := promauto.With(reg).NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Tracks the number of HTTP requests.",
		}, labels,
	)
	requestDuration := promauto.With(reg).NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Tracks the latencies for HTTP requests.",
			Buckets: m.buckets,
		},
		labels,
	)
	requestSize := promauto.With(reg).NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "http_request_size_bytes",
			Help: "Tracks the size of HTTP requests.",
		},
		labels,
	)

	base := promhttp.InstrumentHandlerCounter(
		requestsTotal,
		promhttp.InstrumentHandlerDuration(
			requestDuration,
			promhttp.InstrumentHandlerRequestSize(
				requestSize,
				handler,
				promhttp.WithLabelFromCtx("path", pathLabelFromCtx),
			),
			promhttp.WithLabelFromCtx("path", pathLabelFromCtx),
		),
		promhttp.WithLabelFromCtx("path", pathLabelFromCtx),
	)

	return base.ServeHTTP
}

func pathLabelFromCtx(ctx context.Context) string {
	if path, ok := ctx.Value(LabelPath).(string); ok {
		return path
	}
	return "unknown"
}

// ApplyLabels applies the path label to the request context.
func ApplyLabels(r *http.Request) {
	ctx := context.WithValue(r.Context(), LabelPath, r.URL.Path)
	*r = *r.WithContext(ctx)
}

// HandlerApplyLabels is a middleware helper function to apply labels to an HTTP handler.
func HandlerApplyLabels(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ApplyLabels(r)
		handler.ServeHTTP(w, r)
	})
}
