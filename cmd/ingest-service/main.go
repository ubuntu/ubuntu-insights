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
		if err := storage.Close(5 * time.Second); err != nil {
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

	// Channel to signal when the last processing finishes
	idle := make(chan struct{})

	go func() {
		defer close(idle)

		for {
			select {
			case <-ctx.Done():
				// Stop processing new work immediately
				slog.Info("Shutdown signal received, waiting for current work to finish...")
				return

			case <-ticker.C:
				// Only process if context isn't already canceled
				if ctx.Err() != nil {
					return
				}

				if err := processor.ProcessFiles(ctx, cfg); err != nil {
					slog.Warn("Failed to process files", "err", err)
				}
			}
		}
	}()

	// Wait until either:
	// - Context is canceled
	// - Ongoing work finishes (whichever happens later)
	<-ctx.Done()
	<-idle

	slog.Info("Ingest service shutdown complete")
}
