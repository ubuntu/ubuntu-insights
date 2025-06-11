// Package database provides the database connection and upload functionality for the ingest service.
// It handles the connection to a PostgreSQL database and provides methods to upload data.
package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ubuntu/ubuntu-insights/internal/server/ingest/models"
)

// Config holds the configuration for connecting to the PostgreSQL database.
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type dbPool interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Close()
}

// Manager manages the PostgreSQL database connection pool.
type Manager struct {
	dbpool dbPool
}

type options struct {
	newPool func(ctx context.Context, dsn string) (dbPool, error)
}

// Options represents an optional function to override Manager default values.
type Options func(*options)

// Connect establishes a connection to the PostgreSQL database using the provided configuration.
func Connect(ctx context.Context, cfg Config, args ...Options) (*Manager, error) {
	opts := options{
		newPool: func(ctx context.Context, dsn string) (dbPool, error) {
			return pgxpool.New(ctx, dsn)
		},
	}

	for _, opt := range args {
		opt(&opts)
	}

	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	dbpool, err := opts.newPool(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	slog.Info("Connected to PostgreSQL database", "host", cfg.Host, "port", cfg.Port)
	return &Manager{dbpool: dbpool}, nil
}

// Upload uploads the provided TargetModel to the PostgreSQL database.
func (db Manager) Upload(ctx context.Context, app string, report *models.TargetModel) error {
	return db.upload(ctx, app, func(ctx context.Context, table string) (pgconn.CommandTag, error) {
		if report.OptOut {
			query := fmt.Sprintf(
				`INSERT INTO %s (
					entry_time,
					optout
				) VALUES ($1, $2)`,
				table,
			)
			return db.dbpool.Exec(
				ctx,
				query,
				time.Now(),    // entry_time
				report.OptOut, // optout
			)
		}
		query := fmt.Sprintf(
			`INSERT INTO %s (
				entry_time, 
				insights_version, 
				collection_time,
				hardware, 
				software, 
				platform, 
				source_metrics,
				optout 
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			table,
		)

		return db.dbpool.Exec(
			ctx,
			query,
			time.Now(),                          // entry_time
			report.InsightsVersion,              // insights_version
			time.Unix(report.CollectionTime, 0), // collection_time
			report.SystemInfo.Hardware,          // hardware_info
			report.SystemInfo.Software,          // software_info
			report.SystemInfo.Platform,          // platform_info
			report.SourceMetrics,                // source_metrics
			report.OptOut,                       // optout
		)
	})
}

// UploadLegacy uploads the provided legacy report to the PostgreSQL database.
func (db Manager) UploadLegacy(ctx context.Context, distribution, version string, report *models.LegacyTargetModel) error {
	const table = "ubuntu_report"

	return db.upload(ctx, table, func(ctx context.Context, table string) (pgconn.CommandTag, error) {
		if report.OptOut {
			query := fmt.Sprintf(
				`INSERT INTO %s (
					entry_time,
					distribution,
					version,
					optout
				) VALUES ($1, $2, $3, $4)`,
				table,
			)

			return db.dbpool.Exec(ctx, query,
				time.Now(),    // entry_time
				distribution,  // distribution
				version,       // version
				report.OptOut, // optout
			)
		}

		query := fmt.Sprintf(
			`INSERT INTO %s (
					entry_time, 
					distribution,
					version, 
					report,
					optout
				) VALUES ($1, $2, $3, $4, $5)`,
			table,
		)

		return db.dbpool.Exec(ctx, query,
			time.Now(),    // entry_time
			distribution,  // distribution
			version,       // version
			report,        // report
			report.OptOut, // optout
		)
	})
}

func (db Manager) upload(ctx context.Context, table string, execFn func(context.Context, string) (pgconn.CommandTag, error)) error {
	if db.dbpool == nil {
		return fmt.Errorf("database not initialized")
	}

	table = pgx.Identifier{table}.Sanitize()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := execFn(ctx, table)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return fmt.Errorf("upload canceled: %v", err)
		}
		return fmt.Errorf("failed to upload data: %v", err)
	}
	return nil
}

// Close closes the database connection.
//
// If the connection is already closed, it does nothing.
// If the connection does not close within 10 seconds, it returns an error.
func (db *Manager) Close() error {
	if db.dbpool == nil {
		return nil
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		db.dbpool.Close()
	}()

	select {
	case <-done:
		db.dbpool = nil
		return nil
	case <-time.After(10 * time.Second):
		return fmt.Errorf("timeout while closing database, connection may still be open")
	}
}
