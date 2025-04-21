package main

import (
	"flag"
	"log/slog"
)

func main() {
	cfgPath := flag.String("config", "config.json", "Path to configuration file")
	flag.Parse()

	if *cfgPath == "" {
		slog.Error("Configuration file path is required")
		return
	}
}
