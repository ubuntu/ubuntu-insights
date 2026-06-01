package migration_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // PGX driver for database/sql
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/server/internal/ingest/migration"
	"github.com/ubuntu/ubuntu-insights/server/internal/ingest/testutils"
)

func TestBootstrapFromGolangMigrate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupDB func(t *testing.T, db *sql.DB)

		wantErr             bool
		wantGooseRows       int
		wantSchemaMigration bool
	}{
		"Fresh database no-ops": {
			setupDB:             func(_ *testing.T, _ *sql.DB) {},
			wantGooseRows:       0,
			wantSchemaMigration: false,
		},
		"Bootstraps from golang-migrate state": {
			setupDB:             setupGolangMigrateState(8, false),
			wantGooseRows:       9, // versions 0 through 8
			wantSchemaMigration: false,
		},
		"Partial version bootstraps correctly": {
			setupDB:             setupGolangMigrateState(3, false),
			wantGooseRows:       4, // versions 0 through 3
			wantSchemaMigration: false,
		},
		"Idempotent when run twice": {
			setupDB: func(t *testing.T, db *sql.DB) {
				t.Helper()
				setupGolangMigrateState(8, false)(t, db)

				// Run bootstrap once before the test runs it again.
				err := migration.BootstrapFromGolangMigrate(t.Context(), db)
				require.NoError(t, err)
			},
			wantGooseRows:       9,
			wantSchemaMigration: false,
		},
		"Errors on dirty state": {
			setupDB:             setupGolangMigrateState(8, true),
			wantErr:             true,
			wantSchemaMigration: true, // not dropped on error
		},
		"Errors when expected tables missing": {
			setupDB: func(t *testing.T, db *sql.DB) {
				t.Helper()
				// Create schema_migrations claiming version 8, but don't create any tables.
				_, err := db.ExecContext(t.Context(), `CREATE TABLE schema_migrations (version BIGINT NOT NULL PRIMARY KEY, dirty BOOLEAN NOT NULL)`)
				require.NoError(t, err)
				_, err = db.ExecContext(t.Context(), `INSERT INTO schema_migrations (version, dirty) VALUES (8, false)`)
				require.NoError(t, err)
			},
			wantErr:             true,
			wantSchemaMigration: true, // not dropped on error
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			db := openTestDB(t)
			tc.setupDB(t, db)

			err := migration.BootstrapFromGolangMigrate(t.Context(), db)

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// Verify goose_db_version state.
			gooseRows := countGooseRows(t, db)
			assert.Equal(t, tc.wantGooseRows, gooseRows, "unexpected goose_db_version row count")

			// Verify schema_migrations presence.
			hasSM := tableExists(t, db, "schema_migrations")
			assert.Equal(t, tc.wantSchemaMigration, hasSM, "unexpected schema_migrations existence")

			// If bootstrap succeeded and goose rows were seeded, verify all are marked applied.
			if !tc.wantErr && tc.wantGooseRows > 0 {
				verifyAllApplied(t, db, tc.wantGooseRows)
			}
		})
	}
}

// openTestDB starts a PostgreSQL container and returns an open *sql.DB.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	container := testutils.StartPostgresContainer(t)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := container.Stop(ctx); err != nil {
			t.Errorf("Teardown: failed to stop container: %v", err)
		}
	})
	require.NoError(t, container.IsReady(t, 5*time.Second, 10), "Setup: container was not ready in time")

	db, err := sql.Open("pgx", container.DSN)
	require.NoError(t, err, "Setup: failed to open database")
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	return db
}

// setupGolangMigrateState returns a setup function that creates the schema_migrations
// table and the expected application tables for the given version.
func setupGolangMigrateState(version int, dirty bool) func(*testing.T, *sql.DB) {
	return func(t *testing.T, db *sql.DB) {
		t.Helper()
		ctx := t.Context()

		_, err := db.ExecContext(ctx, `CREATE TABLE schema_migrations (version BIGINT NOT NULL PRIMARY KEY, dirty BOOLEAN NOT NULL)`)
		require.NoError(t, err, "Setup: failed to create schema_migrations")

		_, err = db.ExecContext(ctx, `INSERT INTO schema_migrations (version, dirty) VALUES ($1, $2)`, version, dirty)
		require.NoError(t, err, "Setup: failed to seed schema_migrations")

		// Create the expected application tables for the version.
		tables := []string{
			"invalid_reports",
			"linux",
			"windows",
			"darwin",
			"ubuntu_report",
			"ubuntu_desktop_provision",
			"ubuntu_release_upgrader",
			"wsl_setup",
		}
		for i := range version {
			if i >= len(tables) {
				break
			}
			_, err := db.ExecContext(ctx, fmt.Sprintf(`CREATE TABLE %s (id SERIAL PRIMARY KEY)`, tables[i]))
			require.NoError(t, err, "Setup: failed to create table %s", tables[i])
		}
	}
}

// countGooseRows returns the number of rows in goose_db_version, or 0 if the table doesn't exist.
func countGooseRows(t *testing.T, db *sql.DB) int {
	t.Helper()

	if !tableExists(t, db, "goose_db_version") {
		return 0
	}

	var count int
	err := db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM goose_db_version`).Scan(&count)
	require.NoError(t, err)
	return count
}

// tableExists checks if a table exists in the public schema.
func tableExists(t *testing.T, db *sql.DB, table string) bool {
	t.Helper()

	var exists bool
	err := db.QueryRowContext(t.Context(), `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = $1
		)
	`, table).Scan(&exists)
	require.NoError(t, err)
	return exists
}

// verifyAllApplied checks that all goose_db_version rows have is_applied = true
// and versions are sequential starting from 0.
func verifyAllApplied(t *testing.T, db *sql.DB, expectedCount int) {
	t.Helper()

	rows, err := db.QueryContext(t.Context(), `SELECT version_id, is_applied FROM goose_db_version ORDER BY version_id`)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()

	var i int64
	for rows.Next() {
		var versionID int64
		var isApplied bool
		require.NoError(t, rows.Scan(&versionID, &isApplied))
		assert.Equal(t, i, versionID, "version_id should be sequential")
		assert.True(t, isApplied, "version %d should be marked as applied", versionID)
		i++
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, int64(expectedCount), i, "unexpected number of goose version rows")
}

func TestBootstrapPreservesData(t *testing.T) {
	t.Parallel()

	db := openTestDB(t)
	ctx := t.Context()

	// Load the real golang-migrate dump to get exact production schema.
	dumpPath := filepath.Join(testutils.ModuleRoot(), "internal", "ingest", "migration", "testdata", "golang_migrate_dump.sql")
	dumpSQL, err := os.ReadFile(dumpPath)
	require.NoError(t, err, "Setup: failed to read golang_migrate_dump.sql")

	_, err = db.ExecContext(ctx, string(dumpSQL))
	require.NoError(t, err, "Setup: failed to apply golang_migrate_dump.sql")

	// Insert sample data across multiple tables.
	_, err = db.ExecContext(ctx, `
		INSERT INTO invalid_reports (report_id, entry_time, app_name, raw_report)
		VALUES ('11111111-1111-1111-1111-111111111111', '2025-01-15 10:00:00', 'test-app', '{"invalid": true}')
	`)
	require.NoError(t, err, "Setup: failed to insert invalid_reports row")

	_, err = db.ExecContext(ctx, `
		INSERT INTO linux (report_id, entry_time, insights_version, collection_time, hardware, software, platform, source_metrics, optout)
		VALUES ('22222222-2222-2222-2222-222222222222', '2025-02-20 12:00:00', '1.0.0', '2025-02-20 11:55:00', '{"cpu": "x86"}', '{"os": "ubuntu"}', '{"arch": "amd64"}', '{"src": "apt"}', false)
	`)
	require.NoError(t, err, "Setup: failed to insert linux row")

	_, err = db.ExecContext(ctx, `
		INSERT INTO ubuntu_report (report_id, entry_time, distribution, version, report, optout)
		VALUES ('33333333-3333-3333-3333-333333333333', '2025-03-10 08:00:00', 'Ubuntu', '24.04', '{"legacy": true}', false)
	`)
	require.NoError(t, err, "Setup: failed to insert ubuntu_report row")

	// Run bootstrap.
	err = migration.BootstrapFromGolangMigrate(ctx, db)
	require.NoError(t, err, "Bootstrap should succeed")

	// Run goose migrations to confirm goose considers DB up-to-date.
	migrationsDir := filepath.Join(testutils.ModuleRoot(), "migrations")
	require.NoError(t, goose.SetDialect("postgres"))
	err = goose.Up(db, migrationsDir)
	require.NoError(t, err, "Goose Up should succeed (no new migrations)")

	// Verify data is intact.
	var count int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM invalid_reports WHERE report_id = '11111111-1111-1111-1111-111111111111'`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "invalid_reports data should be preserved")

	var hw, sw, platform string
	err = db.QueryRowContext(ctx, `SELECT hardware::text, software::text, platform::text FROM linux WHERE report_id = '22222222-2222-2222-2222-222222222222'`).Scan(&hw, &sw, &platform)
	require.NoError(t, err)
	assert.JSONEq(t, `{"cpu": "x86"}`, hw, "linux hardware data should be preserved")
	assert.JSONEq(t, `{"os": "ubuntu"}`, sw, "linux software data should be preserved")
	assert.JSONEq(t, `{"arch": "amd64"}`, platform, "linux platform data should be preserved")

	var dist, ver string
	err = db.QueryRowContext(ctx, `SELECT distribution, version FROM ubuntu_report WHERE report_id = '33333333-3333-3333-3333-333333333333'`).Scan(&dist, &ver)
	require.NoError(t, err)
	assert.Equal(t, "Ubuntu", dist, "ubuntu_report distribution should be preserved")
	assert.Equal(t, "24.04", ver, "ubuntu_report version should be preserved")
}
