package daemon

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx" // PGX driver for golang-migrate
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/spf13/cobra"
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
	m, err := migrate.New(
		fmt.Sprintf("file://%s", a.config.MigrationsDir),
		a.config.DBconfig.URI("pgx"),
	)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %v", err)
	}
	defer func() {
		if sErr, dbErr := m.Close(); sErr != nil || dbErr != nil {
			if sErr != nil {
				slog.Error("failed to close migration instance", "error", sErr)
			}
			if dbErr != nil {
				slog.Error("failed to close database connection", "error", dbErr)
			}
		}
	}()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			slog.Info("No new migrations to apply")
			return nil
		}

		return fmt.Errorf("failed to apply migrations: %v", err)
	}
	slog.Info("Migrations applied successfully")
	return nil
}
