package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type DBConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"db_name"`
	SSLMode  string `json:"sslmode"`
}

type ServiceConfig struct {
	InputDir string   `json:"input_dir"`
	DB       DBConfig `json:"db"`
}

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
