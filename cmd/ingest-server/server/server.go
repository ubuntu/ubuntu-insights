package server

import "github.com/ubuntu/ubuntu-insights/cmd/ingest-server/server/middleware"

const (
	rateLimitPerSecond = 0.1
	burstLimit         = 3
)

type Server struct {
	IPLimiter *middleware.IPLimiter
}

func NewServer() Server {
	return Server{
		IPLimiter: middleware.NewIPLimiter(rateLimitPerSecond, burstLimit),
	}
}
