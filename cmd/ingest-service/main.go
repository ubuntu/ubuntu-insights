package main

import (
	"flag"
	"log/slog"
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

	for {
		if err := processor.ProcessFiles(cfg); err != nil {
			slog.Warn("Failed to process files", "err", err)
		}

		time.Sleep(30 * time.Second)
	}

}
