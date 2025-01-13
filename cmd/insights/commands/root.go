// Package commands contains the commands for the Ubuntu Insights CLI.
package commands

import (
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/ubuntu/ubuntu-insights/internal/constants"
)

// App represents the application.
type App struct {
	rootCmd *cobra.Command

	rootConfig    rootConfig
	collectConfig collectConfig
	uploadConfig  uploadConfig
	consentConfig consentConfig
}

type rootConfig struct {
	Verbose     bool
	ConsentDir  string
	InsightsDir string
}

var defaultRootConfig = rootConfig{
	Verbose:     false,
	ConsentDir:  constants.GetDefaultConfigPath(),
	InsightsDir: constants.GetDefaultCachePath(),
}

// New registers commands and returns a new App.
func New() (*App, error) {
	a := App{}
	a.rootCmd = &cobra.Command{
		Use:           constants.CmdName + " [COMMAND]",
		Short:         "",
		Long:          "",
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Command parsing has been successful. Returns to not print usage anymore.
			a.rootCmd.SilenceUsage = true

			setVerbosity(a.rootConfig.Verbose)
			return nil
		},
	}

	err := installRootCmd(&a)
	installCollectCmd(&a)
	installUploadCmd(&a)
	installConsentCmd(&a)

	return &a, err
}

func installRootCmd(app *App) error {
	cmd := app.rootCmd

	app.rootConfig = defaultRootConfig

	cmd.PersistentFlags().BoolVarP(&app.rootConfig.Verbose, "verbose", "v", app.rootConfig.Verbose, "enable verbose logging")
	cmd.PersistentFlags().StringVar(&app.rootConfig.ConsentDir, "consent-dir", app.rootConfig.ConsentDir, "the base directory of the consent state files")
	cmd.PersistentFlags().StringVar(&app.rootConfig.InsightsDir, "insights-dir", app.rootConfig.InsightsDir, "the base directory of the insights report cache")

	if err := cmd.MarkPersistentFlagDirname("consent-dir"); err != nil {
		slog.Error("An error occurred while initializing Ubuntu Insights", "error", err.Error())
		return err
	}

	if err := cmd.MarkPersistentFlagDirname("insights-dir"); err != nil {
		slog.Error("An error occurred while initializing Ubuntu Insights.", "error", err.Error())
		return err
	}

	return nil
}

// setVerbosity sets the global logging level based on the verbose flag. If verbose is true, it sets the logging level to debug, otherwise it sets it to info.
func setVerbosity(verbose bool) {
	if verbose {
		slog.SetLogLoggerLevel(slog.LevelDebug)
		slog.Debug("Verbose logging enabled")
	} else {
		slog.SetLogLoggerLevel(constants.DefaultLogLevel)
	}
}

// Run executes the command and associated process, returning an error if any.
func (a *App) Run() error {
	return a.rootCmd.Execute()
}

// UsageError returns if the error is a command parsing or runtime one.
func (a App) UsageError() bool {
	return !a.rootCmd.SilenceUsage
}

// Quit gracefully exits the application.
func (a App) Quit() {
	// Not implemented
}
