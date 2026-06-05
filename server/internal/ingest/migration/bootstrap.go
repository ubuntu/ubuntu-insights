// Package migration provides utilities for database migration management,
// including bootstrapping from golang-migrate to goose.
package migration

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
)

// dbSchema is the PostgreSQL schema where migration tables reside.
const dbSchema = "public"

// expectedTables is the list of tables that should exist if schema_migrations
// reports version 8 (all initial migrations applied).
var expectedTables = []string{
	"invalid_reports",
	"linux",
	"windows",
	"darwin",
	"ubuntu_report",
	"ubuntu_desktop_provision",
	"ubuntu_release_upgrader",
	"wsl_setup",
}

// BootstrapFromGolangMigrate checks if the database was previously managed by
// golang-migrate and, if so, seeds goose's version table with the correct state.
// This function is idempotent and safe to call on every startup.
//
// Cases handled:
//   - Fresh database (no schema_migrations): no-op
//   - Already bootstrapped (schema_migrations gone): no-op
//   - Existing golang-migrate state: seeds goose_db_version, drops schema_migrations
//   - Dirty state: returns error requiring manual intervention
func BootstrapFromGolangMigrate(ctx context.Context, db *sql.DB) error {
	// Check if golang-migrate's tracking table exists
	var exists bool
	err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = $1
			  AND table_name = 'schema_migrations'
		)
	`, dbSchema).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check for schema_migrations table: %v", err)
	}

	if !exists {
		// Fresh install or already bootstrapped
		return nil
	}

	// Read current state
	var version int64
	var dirty bool
	err = db.QueryRowContext(ctx, `SELECT version, dirty FROM schema_migrations`).Scan(&version, &dirty)
	if err != nil {
		return fmt.Errorf("failed to read schema_migrations: %v", err)
	}

	if dirty {
		return fmt.Errorf("schema_migrations is dirty at version %d: manual intervention required to resolve the inconsistent state before migration tooling can proceed", version)
	}

	// Sanity check: verify expected tables exist
	if err := verifyExpectedTables(ctx, db, version); err != nil {
		return err
	}

	// Perform the bootstrap in a transaction
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin bootstrap transaction: %v", err)
	}
	defer func() {
		_ = tx.Rollback() // no-op if already committed
	}()

	// Create goose_db_version table if it doesn't exist
	_, err = tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS goose_db_version (
			id SERIAL PRIMARY KEY,
			version_id BIGINT NOT NULL,
			is_applied BOOLEAN NOT NULL,
			tstamp TIMESTAMP DEFAULT now()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create goose_db_version table: %v", err)
	}

	// Check if already seeded (handles partial bootstrap recovery)
	var count int
	err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM goose_db_version`).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check goose_db_version state: %v", err)
	}

	if count == 0 {
		// Seed version 0 (goose's initial entry) through current version
		for v := int64(0); v <= version; v++ {
			_, err = tx.ExecContext(ctx,
				`INSERT INTO goose_db_version (version_id, is_applied) VALUES ($1, true)`, v,
			)
			if err != nil {
				return fmt.Errorf("failed to seed goose_db_version at version %d: %v", v, err)
			}
		}
		slog.Info("Bootstrapped goose_db_version from golang-migrate state", "version", version)
	}

	// Drop the old tracking table
	_, err = tx.ExecContext(ctx, `DROP TABLE schema_migrations`)
	if err != nil {
		return fmt.Errorf("failed to drop schema_migrations: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit bootstrap transaction: %v", err)
	}

	slog.Info("Successfully migrated from golang-migrate to goose", "previous_version", version)
	return nil
}

// verifyExpectedTables checks that tables expected for the given migration version
// actually exist in the database as a sanity check before seeding goose state.
func verifyExpectedTables(ctx context.Context, db *sql.DB, version int64) error {
	// Each migration creates one table, versions 1-8 map to expectedTables[0-7]
	for i := int64(0); i < version && i < int64(len(expectedTables)); i++ {
		table := expectedTables[i]
		var exists bool
		err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT FROM information_schema.tables
				WHERE table_schema = $1
				  AND table_name = $2
			)
		`, dbSchema, table).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check existence of table %q: %v", table, err)
		}
		if !exists {
			return fmt.Errorf("schema_migrations reports version %d but expected table %q does not exist: manual intervention required", version, table)
		}
	}
	return nil
}
