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
)

func main() {
	var cfgPath string
	flag.StringVar(&cfgPath, "config", "config.json", "Path to configuration file")
	flag.Parse()

	configManager := server.NewConfigManager(cfgPath)
	if err := configManager.Load(); err != nil {
		slog.Error("Failed to load configuration", "err", err)
		return
	}
	go configManager.Watch()

	s := server.NewServer(configManager)

	mux := http.NewServeMux()
	mux.Handle("POST /upload/{app}", s.RateLimitMiddleware(http.HandlerFunc(s.UploadHandler)))
	mux.Handle("GET /version", http.HandlerFunc(s.VersionHandler))

	srv := &http.Server{
		Addr: ":8080",
		ReadTimeout: 5 * time.Second,
		WriteTimeout: 10 * time.Second,
		Handler: http.TimeoutHandler(mux, 1 * time.Second, ""),
		MaxHeaderBytes: 1 << 20,
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
