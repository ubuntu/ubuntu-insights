// Package storage provides the database connection and upload functionality for the ingest service.
// It handles the connection to a PostgreSQL database and provides methods to upload data.
package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/lib/pq"
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
func Close(timeout time.Duration) error {
	if db == nil {
		return nil
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		db.Close()
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timeout while closing database")
	}
}

// UploadToPostgres uploads the provided FileData to the PostgreSQL database.
func UploadToPostgres(ctx context.Context, data *models.FileData) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	query := fmt.Sprintf(`INSERT INTO %s (generated, schema_version) VALUES ($1, $2)`, pq.QuoteIdentifier(data.AppID))
	_, err := db.ExecContext(ctx, query, data.Generated, data.SchemaVersion)

	if err != nil {
		if errors.Is(err, context.Canceled) {
			return fmt.Errorf("upload canceled: %w", err)
		}
		return fmt.Errorf("failed to upload data: %w", err)
	}
	return nil
}
