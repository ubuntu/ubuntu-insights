// Package daemon provides the ingest service daemon for Ubuntu Insights.
package daemon

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ubuntu/ubuntu-insights/common/cli"
	"github.com/ubuntu/ubuntu-insights/server/internal/common/config"
	"github.com/ubuntu/ubuntu-insights/server/internal/common/constants"
	"github.com/ubuntu/ubuntu-insights/server/internal/ingest"
	"github.com/ubuntu/ubuntu-insights/server/internal/ingest/database"
	"github.com/ubuntu/ubuntu-insights/server/internal/ingest/processor"
)

// App represents the application.
type App struct {
	cmd    *cobra.Command
	viper  *viper.Viper
	config appConfig

	daemon *ingest.Service

	ready chan struct{}
}

// appConfig holds the configuration for the application.
type appConfig struct {
	Verbosity     int
	DBconfig      database.Config
	ReportsDir    string // Base directory for reports
	ConfigPath    string
	MigrationsDir string
}

// New creates a new App instance with default values.
func New() (*App, error) {
	a := App{ready: make(chan struct{})}

	a.cmd = &cobra.Command{
		Use:           constants.IngestServiceCmdName,
		Short:         "Ubuntu Insights ingest service",
		Long:          "Ubuntu Insights ingest service uses validated received reports and inserts them into a PostgreSQL database.",
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Command parsing has been successful. Returns to not print usage anymore.
			a.cmd.SilenceUsage = true
			cli.SetVerbosity(a.config.Verbosity) // Set verbosity before loading config
			if err := cli.InitViperConfig(constants.IngestServiceCmdName, a.cmd, a.viper); err != nil {
				return err
			}
			if err := a.viper.Unmarshal(&a.config); err != nil {
				return fmt.Errorf("unable to strictly decode configuration into struct: %w", err)
			}
			slog.Info("got app config", "config", a.config)

			cli.SetVerbosity(a.config.Verbosity)
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			a.cmd.SilenceUsage = true

			return a.run()
		},
	}
	a.viper = viper.New()
	a.cmd.CompletionOptions.HiddenDefaultCmd = true

	installRootCmd(&a)
	installMigrateCmd(&a)
	cli.InstallConfigFlag(a.cmd)

	if err := a.viper.BindPFlags(a.cmd.PersistentFlags()); err != nil {
		return nil, err
	}

	a.installVersion()

	return &a, nil
}

func installRootCmd(app *App) {
	cmd := app.cmd

	cmd.PersistentFlags().CountVarP(&app.config.Verbosity, "verbose", "v", "issue INFO (-v), DEBUG (-vv)")

	// Daemon flags
	cmd.Flags().StringVar(&app.config.ReportsDir, "reports-dir", constants.DefaultServiceReportsDir, "base directory to read reports from")
	cmd.Flags().StringVarP(&app.config.ConfigPath, "daemon-config", "c", "", "path to the configuration file")

	addDBFlags(cmd, &app.config.DBconfig)

	if err := cmd.MarkFlagDirname("reports-dir"); err != nil {
		panic(fmt.Errorf("failed to mark reports-dir flag as directory: %w", err))
	}

	if err := cmd.MarkFlagDirname("daemon-config"); err != nil {
		panic(fmt.Sprintf("failed to mark daemon-config flag as filename: %v", err))
	}
}

func addDBFlags(cmd *cobra.Command, config *database.Config) {
	cmd.Flags().StringVar(&config.Host, "db-host", "", "database host")
	cmd.Flags().IntVarP(&config.Port, "db-port", "p", 5432, "database port")
	cmd.Flags().StringVarP(&config.User, "db-user", "u", "", "database user")
	cmd.Flags().StringVarP(&config.Password, "db-password", "P", "", "database password")
	cmd.Flags().StringVarP(&config.DBName, "db-name", "n", "", "database name")
	cmd.Flags().StringVarP(&config.SSLMode, "db-sslmode", "s", "", "database SSL mode")
}

// Run executes the command and associated process, returning an error if any.
func (a App) Run() error {
	return a.cmd.Execute()
}

// UsageError returns if the error is a command parsing or runtime one.
func (a App) UsageError() bool {
	return !a.cmd.SilenceUsage
}

// Hup prints all goroutine stack traces and return false to signal you shouldn't quit.
func (a App) Hup() (shouldQuit bool) {
	buf := make([]byte, 1<<16)
	runtime.Stack(buf, true)
	fmt.Printf("%s", buf)
	return false
}

// Quit gracefully shuts down the daemon.
func (a *App) Quit() {
	a.WaitReady()
	if a.daemon != nil {
		a.daemon.Quit(false)
	}
}

// WaitReady waits for the daemon to be ready.
func (a *App) WaitReady() {
	<-a.ready
}

// RootCmd returns the root command.
func (a App) RootCmd() cobra.Command {
	return *a.cmd
}

func (a *App) run() (err error) {
	a.config.ConfigPath, err = filepath.Abs(a.config.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for config file: %v", err)
	}
	cm := config.New(a.config.ConfigPath)
	db, err := database.Connect(context.Background(), a.config.DBconfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	proc, err := processor.New(a.config.ReportsDir, db)
	if err != nil {
		return fmt.Errorf("failed to create report processor: %v", err)
	}

	a.daemon = ingest.New(context.Background(), cm, proc)
	close(a.ready)

	return a.daemon.Run()
}
