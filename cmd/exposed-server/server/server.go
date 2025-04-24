// Package server provides the main server structure and initialization.
package server

import "github.com/ubuntu/ubuntu-insights/cmd/exposed-server/server/middleware"

const (
	rateLimitPerSecond = 0.1
	burstLimit         = 3
)

// Server represents the main server structure.
type Server struct {
	IPLimiter *middleware.IPLimiter
}

// New creates a new Server instance with an IP limiter.
func New() Server {
	return Server{
		IPLimiter: middleware.New(rateLimitPerSecond, burstLimit),
	}
}
