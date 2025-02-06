// Package commands contains the commands for the Ubuntu Insights CLI.
package commands

import (
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/ubuntu/ubuntu-insights/internal/constants"
	"github.com/ubuntu/ubuntu-insights/internal/uploader"
)

type newUploader func(cm uploader.ConsentManager, cachePath, source string, minAge uint, dryRun bool, args ...uploader.Options) (uploader.Uploader, error)

// App represents the application.
type App struct {
	cmd *cobra.Command

	config struct {
		verbose     bool
		consentDir  string
		insightsDir string
		upload      struct {
			sources []string
			minAge  uint
			force   bool
			dryRun  bool
		}
		collect struct {
			source       string
			period       uint
			force        bool
			dryRun       bool
			extraMetrics string
		}
		consent struct {
			sources []string
			state   string
		}
	}

	newUploader newUploader
}

type options struct {
	newUploader newUploader
}

// Options represents an optional function to override App default values.
type Options func(*options)

// New registers commands and returns a new App.
func New(args ...Options) (*App, error) {
	opts := options{
		newUploader: uploader.New,
	}
	for _, opt := range args {
		opt(&opts)
	}
	a := App{newUploader: opts.newUploader}
	a.cmd = &cobra.Command{
		Use:           constants.CmdName + " [COMMAND]",
		Short:         "",
		Long:          "",
		SilenceErrors: true,
		Version:       constants.Version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Command parsing has been successful. Returns to not print usage anymore.
			a.cmd.SilenceUsage = true

			setVerbosity(a.config.verbose)
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
	cmd := app.cmd

	cmd.PersistentFlags().BoolVarP(&app.config.verbose, "verbose", "v", false, "enable verbose logging")
	cmd.PersistentFlags().StringVar(&app.config.consentDir, "consent-dir", constants.DefaultConfigPath, "the base directory of the consent state files")
	cmd.PersistentFlags().StringVar(&app.config.insightsDir, "insights-dir", constants.DefaultCachePath, "the base directory of the insights report cache")

	if err := cmd.MarkPersistentFlagDirname("consent-dir"); err != nil {
		slog.Error("An error occurred while initializing Ubuntu Insights", "error", err.Error())
		return err
	}

	if err := cmd.MarkPersistentFlagDirname("insights-dir"); err != nil {
		slog.Error("An error occurred while initializing Ubuntu Insights", "error", err.Error())
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
	return a.cmd.Execute()
}

// UsageError returns if the error is a command parsing or runtime one.
func (a App) UsageError() bool {
	return !a.cmd.SilenceUsage
}
