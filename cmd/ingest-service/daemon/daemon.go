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
	"github.com/ubuntu/ubuntu-insights/internal/cli"
	"github.com/ubuntu/ubuntu-insights/internal/constants"
	"github.com/ubuntu/ubuntu-insights/internal/server/ingest"
	"github.com/ubuntu/ubuntu-insights/internal/server/ingest/database"
	"github.com/ubuntu/ubuntu-insights/internal/server/shared/config"
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
	Verbosity  int
	DBconfig   database.Config
	Daemon     ingest.StaticConfig // Configuration for the ingest service
	ConfigPath string
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
			if err := cli.InitViperConfig(constants.WebServiceCmdName, a.cmd, a.viper); err != nil {
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
	cli.InstallConfigFlag(a.cmd)

	if err := a.viper.BindPFlags(a.cmd.PersistentFlags()); err != nil {
		return nil, err
	}

	a.installVersion()

	return &a, nil
}

func installRootCmd(app *App) {
	cmd := app.cmd

	defaultConf := ingest.StaticConfig{
		ReportsDir: constants.DefaultServiceReportsDir,
		InvalidDir: constants.DefaultServiceInvalidReportsDir,
	}

	cmd.PersistentFlags().CountVarP(&app.config.Verbosity, "verbose", "v", "issue INFO (-v), DEBUG (-vv)")

	// Daemon flags
	cmd.PersistentFlags().StringVar(&app.config.DBconfig.Host, "db-host", "", "Database host")
	cmd.PersistentFlags().IntVarP(&app.config.DBconfig.Port, "db-port", "p", 5432, "Database port")
	cmd.PersistentFlags().StringVarP(&app.config.DBconfig.User, "db-user", "u", "", "Database user")
	cmd.PersistentFlags().StringVarP(&app.config.DBconfig.Password, "db-password", "P", "", "Database password")
	cmd.PersistentFlags().StringVarP(&app.config.DBconfig.DBName, "db-name", "n", "", "Database name")
	cmd.PersistentFlags().StringVarP(&app.config.DBconfig.SSLMode, "db-sslmode", "s", "", "Database SSL mode")

	cmd.PersistentFlags().StringVar(&app.config.Daemon.ReportsDir, "reports-dir", defaultConf.ReportsDir, "Directory to read reports from")
	cmd.PersistentFlags().StringVar(&app.config.Daemon.InvalidDir, "invalid-dir", defaultConf.InvalidDir, "Directory where invalid or partially invalid reports are stored for manual review")

	cmd.PersistentFlags().StringVarP(&app.config.ConfigPath, "daemon-config", "c", "", "Path to the configuration file")

	if err := cmd.MarkPersistentFlagDirname("reports-dir"); err != nil {
		panic(fmt.Errorf("failed to mark reports-dir flag as directory: %w", err))
	}

	if err := cmd.MarkPersistentFlagDirname("invalid-dir"); err != nil {
		panic(fmt.Sprintf("failed to mark invalid-dir flag as directory: %v", err))
	}

	if err := cmd.MarkPersistentFlagFilename("daemon-config"); err != nil {
		panic(fmt.Sprintf("failed to mark daemon-config flag as filename: %v", err))
	}
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
	a.daemon, err = ingest.New(context.Background(), cm, a.config.DBconfig, a.config.Daemon)
	close(a.ready)
	if err != nil {
		return fmt.Errorf("failed to create server: %v", err)
	}

	return a.daemon.Run()
}
