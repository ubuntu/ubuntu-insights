// Package config provides configuration management for the ingest service.
package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// DBConfig represents the database configuration.
// It contains the necessary fields to connect to a PostgreSQL database.
type DBConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"db_name"`
	SSLMode  string `json:"sslmode"`
}

// ServiceConfig represents the configuration for the ingest service.
type ServiceConfig struct {
	InputDir string   `json:"input_dir"`
	DB       DBConfig `json:"db"`
	Interval *int     `json:"interval_seconds,omitempty"`
}

// Load reads the configuration from the specified JSON file.
// It returns a ServiceConfig struct populated with the values from the file.
func Load(path string) (*ServiceConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening config file: %w", err)
	}
	defer f.Close()

	var cfg ServiceConfig
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decoding config JSON: %w", err)
	}

	return &cfg, nil
}
