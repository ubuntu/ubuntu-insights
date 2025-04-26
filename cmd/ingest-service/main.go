// Package main provides the entry point for the ingest service.
package main

import (
	"context"
	"flag"
	"log/slog"
	"os/signal"
	"syscall"
	"time"

	"github.com/ubuntu/ubuntu-insights/cmd/ingest-service/config"
	"github.com/ubuntu/ubuntu-insights/cmd/ingest-service/processor"
	"github.com/ubuntu/ubuntu-insights/cmd/ingest-service/storage"
)

func main() {
	cfgPath := flag.String("config", "config.json", "Path to configuration file")
	flag.Parse()

	if *cfgPath == "" {
		slog.Error("Configuration file path is required")
		return
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		slog.Error("Failed to load configuration", "err", err)
		return
	}

	if err := storage.Initialize(cfg.DB); err != nil {
		slog.Error("Failed to initialize database connection", "err", err)
		return
	}
	defer func() {
		if err := storage.Close(); err != nil {
			slog.Error("Error closing database", "err", err)
		} else {
			slog.Info("Database connection closed successfully")
		}
	}()

	// Setup graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	slog.Info("Ingest service started")

	for {
		select {
		case <-ctx.Done():
			slog.Info("Shutdown signal received, exiting...")
			return

		case <-ticker.C:
			if err := processor.ProcessFiles(cfg); err != nil {
				slog.Warn("Failed to process files", "err", err)
			}
		}
	}
}
