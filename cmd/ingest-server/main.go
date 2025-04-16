package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ubuntu/ubuntu-insights/cmd/ingest-server/server"
	"github.com/ubuntu/ubuntu-insights/cmd/ingest-server/server/config"
	"github.com/ubuntu/ubuntu-insights/cmd/ingest-server/server/handlers"
)

const (
	defaultConfigPath = "config.json"
	readTimeout       = 5 * time.Second
	writeTimeout      = 10 * time.Second
	requestTimeout    = 1 * time.Second
	maxHeaderBytes    = 1 << 20 // 1 MB
	listenAddr        = ":8080"
)

func main() {
	var cfgPath string
	flag.StringVar(&cfgPath, "config", defaultConfigPath, "Path to configuration file")
	flag.Parse()

	configManager := config.NewConfigManager(cfgPath)
	if err := configManager.Load(); err != nil {
		slog.Error("Failed to load configuration", "err", err)
		return
	}
	go configManager.Watch()

	s := server.NewServer()

	mux := http.NewServeMux()
	mux.Handle("POST /upload/{app}", s.IPLimiter.RateLimitMiddleware(http.HandlerFunc(handlers.UploadHandler)))
	mux.Handle("GET /version", http.HandlerFunc(handlers.VersionHandler))

	srv := &http.Server{
		Addr:           listenAddr,
		ReadTimeout:    readTimeout,
		WriteTimeout:   writeTimeout,
		Handler:        http.TimeoutHandler(mux, requestTimeout, ""),
		MaxHeaderBytes: maxHeaderBytes,
	}

	slog.Info("Server starting...")
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed", "err", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	slog.Info("Shutting down server...")
}
