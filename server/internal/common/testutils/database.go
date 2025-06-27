// Package testutils package provides ingest-service specific test utilities.
package testutils

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx" // PGX driver for golang-migrate
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// PostgresContainer represents a PostgreSQL container for testing purposes.
type PostgresContainer struct {
	Container testcontainers.Container
	DSN       string

	User     string
	Password string
	Name     string
	Host     string
	Port     string
}

// StartPostgresContainer starts a PostgreSQL container for testing purposes.
func StartPostgresContainer(t *testing.T) *PostgresContainer {
	t.Helper()

	const (
		defaultUser     = "postgres"
		defaultPassword = "postgres"
		defaultName     = "testdb"
	)

	if runtime.GOOS != "linux" {
		t.Skip("Skipping PostgreSQL container test on non-Linux OS")
	}

	req := testcontainers.ContainerRequest{
		Image:        "postgres:latest",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     defaultUser,
			"POSTGRES_PASSWORD": defaultPassword,
			"POSTGRES_DB":       defaultName,
		},
		WaitingFor: wait.ForListeningPort("5432/tcp"),
	}
	ctx := t.Context()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err, "Setup: failed to start PostgreSQL container")
	host, err := container.Host(ctx)
	require.NoError(t, err, "Setup: failed to get container host")

	port, err := container.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err, "Setup: failed to get mapped port")

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		defaultUser,
		defaultPassword,
		host,
		port.Port(),
		defaultName,
	)

	return &PostgresContainer{
		Container: container,
		DSN:       dsn,

		User:     defaultUser,
		Password: defaultPassword,
		Name:     defaultName,
		Host:     host,
		Port:     port.Port(),
	}
}

// Stop stops the PostgreSQL container.
func (pc *PostgresContainer) Stop(ctx context.Context) error {
	return pc.Container.Terminate(ctx)
}

// IsReady checks if the PostgreSQL database is connectable.
// It will attempt to connect to the database multiple times, each attempt being timeout long at most.
func (pc PostgresContainer) IsReady(t *testing.T, timeout time.Duration, attempts int) error {
	t.Helper()

	config, err := pgx.ParseConfig(pc.DSN)
	if err != nil {
		return fmt.Errorf("failed to parse DSN: %w", err)
	}

	for i := range attempts {
		ctx, cancel := context.WithTimeout(t.Context(), timeout)
		conn, err := pgx.ConnectConfig(ctx, config)
		cancel()

		if err != nil {
			t.Logf("Attempt %d: failed to connect to database: %v", i+1, err)
			time.Sleep(1 * time.Second)
			continue
		}

		ctx, cancel = context.WithTimeout(t.Context(), 2*time.Second)
		defer cancel()
		return conn.Close(ctx)
	}

	return fmt.Errorf("database did not become ready after %d attempts: %v", attempts, err)
}

// ApplyMigrations applies migrations from the specified directory to the database using the PGX driver.
func ApplyMigrations(t *testing.T, dsn string, migrationsDir string) {
	t.Helper()
	m, err := migrate.New(
		fmt.Sprintf("file://%s", migrationsDir),
		fmt.Sprintf("pgx://%s", dsn[11:]), // Convert DSN to PGX-compatible format
	)
	require.NoError(t, err, "Setup: failed to create migration instance")
	if err := m.Up(); err != nil {
		require.ErrorIs(t, err, migrate.ErrNoChange, "Setup: failed to apply migrations")
	}
}

// DBListTables lists all the tables, excluding a blacklist.
func DBListTables(t *testing.T, dsn string, blacklist ...string) []string {
	t.Helper()

	blacklistMap := make(map[string]bool)
	for _, table := range blacklist {
		blacklistMap[table] = true
	}

	conn, err := pgx.Connect(t.Context(), dsn)
	require.NoError(t, err, "failed to connect to the database")
	defer func() {
		require.NoError(t, conn.Close(t.Context()), "failed to close the database connection")
	}()

	query := `
        SELECT table_name
        FROM information_schema.tables
        WHERE table_schema = 'public'
          AND table_type = 'BASE TABLE';`

	rows, err := conn.Query(t.Context(), query)
	require.NoError(t, err, "failed to execute query")

	var tables []string
	for rows.Next() {
		var tableName string
		require.NoError(t, rows.Scan(&tableName), "failed to scan table name")
		if !blacklistMap[tableName] {
			tables = append(tables, tableName)
		}
	}

	require.NoError(t, rows.Err(), "error occurred during rows iteration")
	return tables
}
