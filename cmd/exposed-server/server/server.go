package server

import "github.com/ubuntu/ubuntu-insights/cmd/exposed-server/server/middleware"

const (
	rateLimitPerSecond = 0.1
	burstLimit         = 3
)

type Server struct {
	IPLimiter *middleware.IPLimiter
}

func New() Server {
	return Server{
		IPLimiter: middleware.New(rateLimitPerSecond, burstLimit),
	}
}
