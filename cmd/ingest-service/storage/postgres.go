// Package storage provides the database connection and upload functionality for the ingest service.
// It handles the connection to a PostgreSQL database and provides methods to upload data.
package storage

import (
	"database/sql"
	"fmt"
	"log/slog"
	"sync"

	_ "github.com/lib/pq" // PostgreSQL driver.
	"github.com/ubuntu/ubuntu-insights/cmd/ingest-service/config"
	"github.com/ubuntu/ubuntu-insights/cmd/ingest-service/models"
)

var (
	db   *sql.DB
	once sync.Once
)

// Initialize sets up the database connection using the provided configuration.
func Initialize(cfg config.DBConfig) error {
	var err error
	once.Do(func() {
		dsn := fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
		)

		db, err = sql.Open("postgres", dsn)
		if err != nil {
			return
		}

		if pingErr := db.Ping(); pingErr != nil {
			err = pingErr
			return
		}
	})
	return err
}

// Get returns the initialized database connection.
func Get() *sql.DB {
	if db == nil {
		slog.Error("DB not initialized", "err", "Call storage.Initialize first")
	}
	return db
}

// Close closes the database connection.
func Close() error {
	if db != nil {
		return db.Close()
	}
	return nil
}

// UploadToPostgres uploads the provided FileData to the PostgreSQL database.
func UploadToPostgres(data *models.FileData) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	cmd := `INSERT INTO $1 (generated, schema_version) VALUES ($2, $3)`
	_, err := db.Exec(cmd, data.AppID, data.Generated, data.SchemaVersion)

	return err
}
