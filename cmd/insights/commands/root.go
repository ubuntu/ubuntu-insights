// Package commands contains the commands for the Ubuntu Insights CLI.
package commands

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ubuntu/ubuntu-insights/internal/collector"
	"github.com/ubuntu/ubuntu-insights/internal/constants"
	"github.com/ubuntu/ubuntu-insights/internal/uploader"
)

type newUploader func(cm uploader.Consent, cachePath, source string, minAge uint, dryRun bool, args ...uploader.Options) (uploader.Uploader, error)
type newCollector func(cm collector.Consent, cachePath, source string, period uint, dryRun bool, args ...collector.Options) (collector.Collector, error)

// App represents the application.
type App struct {
	cmd   *cobra.Command
	viper *viper.Viper

	config struct {
		Verbose     int
		consentDir  string
		insightsDir string
		Upload      struct {
			Sources []string
			MinAge  uint `mapstructure:"minAge"`
			Force   bool
			DryRun  bool `mapstructure:"dryRun"`
			Retry   bool `mapstructure:"retry"`
		}
		Collect struct {
			Source        string
			Period        uint
			Force         bool
			DryRun        bool
			SourceMetrics string
		}
		Consent struct {
			Sources []string
			State   string
		}
	}

	newUploader  newUploader
	newCollector newCollector
}

type options struct {
	newUploader  newUploader
	newCollector newCollector
}

// Options represents an optional function to override App default values.
type Options func(*options)

// New registers commands and returns a new App.
func New(args ...Options) (*App, error) {
	opts := options{
		newUploader:  uploader.New,
		newCollector: collector.New,
	}
	for _, opt := range args {
		opt(&opts)
	}
	a := App{
		newUploader:  opts.newUploader,
		newCollector: opts.newCollector,
	}
	a.cmd = &cobra.Command{
		Use:   constants.CmdName,
		Short: "A transparent tool to collect and share anonymous insights about your system",
		Long: `A transparent tool to collect and share anonymous insights about your system.
		
If consent is given, this tool collects non-personally identifying hardware, software, and platform information, and shares it with the Ubuntu Development team.
The information collected can't be used to identify a single machine. All reports are cached on the local machine and can be reviewed before and after uploading.`,
		SilenceErrors: true,
		Version:       constants.Version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Command parsing has been successful. Returns to not print usage anymore.
			a.cmd.SilenceUsage = true

			if err := initViperConfig(constants.CmdName, a.cmd, a.viper); err != nil {
				return err
			}
			if err := a.viper.Unmarshal(&a.config); err != nil {
				return fmt.Errorf("unable to decode configuration into struct: %w", err)
			}

			setVerbosity(a.config.Verbose)
			return nil
		},
	}
	a.viper = viper.New()

	if err := installRootCmd(&a); err != nil {
		return nil, err
	}
	installConfigFlag(&a)
	installCollectCmd(&a)
	installUploadCmd(&a)
	installConsentCmd(&a)

	if err := a.viper.BindPFlags(a.cmd.PersistentFlags()); err != nil {
		return nil, err
	}

	return &a, nil
}

func installRootCmd(app *App) error {
	cmd := app.cmd

	cmd.PersistentFlags().CountVarP(&app.config.Verbose, "verbose", "v", "issue INFO (-v), DEBUG (-vv)")
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

// setVerbosity sets the global logging level based on the verbose flag count.
func setVerbosity(level int) {
	switch level {
	case 0:
		slog.SetLogLoggerLevel(constants.DefaultLogLevel)
	case 1:
		slog.SetLogLoggerLevel(slog.LevelInfo)
	default:
		slog.SetLogLoggerLevel(slog.LevelDebug)
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
