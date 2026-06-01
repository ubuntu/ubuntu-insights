package daemon

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib" // PGX driver for database/sql
	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
	"github.com/ubuntu/ubuntu-insights/server/internal/ingest/migration"
)

func installMigrateCmd(app *App) {
	migrateCmd := &cobra.Command{
		Use:   "migrate [path-to-migration-scripts]",
		Short: "Run migration scripts",
		Long: `Run migration scripts to update the database schema or data.
If no path is provided, the default path is used.`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("migrate command accepts exactly one argument")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			app.cmd.SilenceUsage = false

			// Set Migrations directory
			app.config.MigrationsDir = args[0]

			fileInfo, err := os.Stat(app.config.MigrationsDir)
			if err != nil {
				return fmt.Errorf("the provided path to migration scripts is not valid: %v", err)
			}
			if !fileInfo.IsDir() {
				return fmt.Errorf("the provided path to migration scripts should be a directory, not a file")
			}

			app.cmd.SilenceUsage = true

			slog.Info("Running migrate command")
			return app.migrateRun()
		},
	}

	addDBFlags(migrateCmd, &app.config.DBconfig)

	app.cmd.AddCommand(migrateCmd)
}

func (a App) migrateRun() error {
	db, err := sql.Open("pgx", a.config.DBconfig.URI("postgres"))
	if err != nil {
		return fmt.Errorf("failed to open database connection: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("failed to close database connection", "error", err)
		}
	}()

	ctx := context.Background()

	// Bootstrap from golang-migrate if needed (idempotent)
	if err := migration.BootstrapFromGolangMigrate(ctx, db); err != nil {
		return fmt.Errorf("failed to bootstrap from golang-migrate: %v", err)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %v", err)
	}

	if err := goose.UpContext(ctx, db, a.config.MigrationsDir); err != nil {
		return fmt.Errorf("failed to apply migrations: %v", err)
	}

	slog.Info("Migrations applied successfully")
	return nil
}
