// Package database provides the database connection and upload functionality for the ingest service.
// It handles the connection to a PostgreSQL database and provides methods to upload data.
package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ubuntu/ubuntu-insights/server/internal/ingest/models"
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
	Ping(ctx context.Context) error
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

// New creates database manager with a PostgreSQL connection pool using the provided configuration.
// Note: The connection is validated with a ping, but it is not maintained.
func New(ctx context.Context, cfg Config, args ...Options) (*Manager, error) {
	opts := options{
		newPool: func(ctx context.Context, dsn string) (dbPool, error) {
			return pgxpool.New(ctx, dsn)
		},
	}

	for _, opt := range args {
		opt(&opts)
	}

	dbpool, err := opts.newPool(ctx, cfg.URI("postgres"))
	if err != nil {
		return nil, fmt.Errorf("unable to create database connection pool: %w", err)
	}

	slog.Debug("Testing database connection", "host", cfg.Host, "port", cfg.Port)
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := dbpool.Ping(pingCtx); err != nil {
		dbpool.Close()
		return nil, fmt.Errorf("unable to ping database: %v", err)
	}

	slog.Info("Successfully pinged PostgreSQL database", "host", cfg.Host, "port", cfg.Port)
	return &Manager{dbpool: dbpool}, nil
}

// Upload uploads the provided TargetModel to the PostgreSQL database.
func (db Manager) Upload(ctx context.Context, id, app string, report *models.TargetModel) error {
	return db.upload(ctx, app, func(ctx context.Context, table string) (pgconn.CommandTag, error) {
		if report.OptOut {
			query := fmt.Sprintf(
				`INSERT INTO %s (
					report_id,
					entry_time,
					optout
				) VALUES ($1, $2, $3)`,
				table,
			)
			return db.dbpool.Exec(
				ctx,
				query,
				id,            // report_id
				time.Now(),    // entry_time
				report.OptOut, // optout
			)
		}
		query := fmt.Sprintf(
			`INSERT INTO %s (
				report_id,
				entry_time, 
				insights_version, 
				collection_time,
				hardware, 
				software, 
				platform, 
				source_metrics,
				optout 
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			table,
		)

		return db.dbpool.Exec(
			ctx,
			query,
			id,                                  // report_id
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
func (db Manager) UploadLegacy(ctx context.Context, id, distribution, version string, report *models.LegacyTargetModel) error {
	const table = "ubuntu_report"

	return db.upload(ctx, table, func(ctx context.Context, table string) (pgconn.CommandTag, error) {
		if report.OptOut {
			query := fmt.Sprintf(
				`INSERT INTO %s (
					report_id,
					entry_time,
					distribution,
					version,
					optout
				) VALUES ($1, $2, $3, $4, $5)`,
				table,
			)

			return db.dbpool.Exec(ctx, query,
				id,            // report_id
				time.Now(),    // entry_time
				distribution,  // distribution
				version,       // version
				report.OptOut, // optout
			)
		}

		query := fmt.Sprintf(
			`INSERT INTO %s (
					report_id,
					entry_time, 
					distribution,
					version, 
					report,
					optout
				) VALUES ($1, $2, $3, $4, $5, $6)`,
			table,
		)

		return db.dbpool.Exec(ctx, query,
			id,            // report_id
			time.Now(),    // entry_time
			distribution,  // distribution
			version,       // version
			report,        // report
			report.OptOut, // optout
		)
	})
}

// UploadInvalid uploads the invalid report to the invalid_reports table as a string.
func (db Manager) UploadInvalid(ctx context.Context, id, app, rawReport string) error {
	const table = "invalid_reports"

	return db.upload(ctx, table, func(ctx context.Context, table string) (pgconn.CommandTag, error) {
		query := fmt.Sprintf(
			`INSERT INTO %s (
				report_id,
				entry_time,
				app_name,
				raw_report
			) VALUES ($1, $2, $3, $4)`,
			table,
		)

		return db.dbpool.Exec(ctx, query,
			id,         // report_id
			time.Now(), // entry_time
			app,        // app
			rawReport,  // raw_report
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

// URI is a helper method that returns a connection URI for PostgreSQL.
// It does not check the validity of the configuration values.
//
// Security warning: the returned string may include credentials.
func (c Config) URI(scheme string) string {
	host := c.Host
	if c.Port != 0 {
		host = fmt.Sprintf("%s:%d", c.Host, c.Port)
	}

	user := url.User(c.User)
	if c.Password != "" {
		user = url.UserPassword(c.User, c.Password)
	}

	u := &url.URL{
		Scheme: scheme,
		User:   user,
		Host:   host,
		Path:   c.DBName,
	}

	q := u.Query()
	if c.SSLMode != "" {
		q.Set("sslmode", c.SSLMode)
	}
	u.RawQuery = q.Encode()
	return u.String()
}
