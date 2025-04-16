package handlers

import (
	"github.com/ubuntu/ubuntu-insights/cmd/ingest-server/server"
	"github.com/ubuntu/ubuntu-insights/cmd/ingest-server/server/middleware"
)

const (
	rateLimitPerSecond = 0.1
	burstLimit         = 3
)

type Server struct {
	configManager *server.ConfigManager
	IPLimiter     *middleware.IPLimiter
}

func NewServer(configManager *server.ConfigManager) Server {
	return Server{
		configManager: configManager,
		IPLimiter:     middleware.NewIPLimiter(rateLimitPerSecond, burstLimit),
	}
}
