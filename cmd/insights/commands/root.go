package commands

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/ubuntu/ubuntu-insights/internal/constants"
)

// Root command flags
var consentDir string
var verbose bool

// Shared flags but different configurations
var source string

// Command flags shared between collect and upload
var force bool
var dryRun bool
var dir string

var rootCmd = &cobra.Command{
	Use:   "ubuntu-insights",
	Short: "",
	Long:  "",
}

func init() {
	userConfigDir, err := os.UserConfigDir()
	defaultConsentDir := ""
	if err != nil {
		log.Error().Err(err).Msg("Failed to get user config directory, expecting consent-dir flag to be set")

	} else {
		defaultConsentDir = userConfigDir + string(os.PathSeparator) + constants.DefaultAppDirName
	}

	rootCmd.PersistentFlags().StringVar(&consentDir, "consent-dir", defaultConsentDir, "directory to look for and to store user consent")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose logging")

	err = rootCmd.MarkPersistentFlagDirname("consent-dir")

	if err != nil {
		log.Fatal().Err(err).Msg("An error occurred while initializing Ubuntu Insights.")
		os.Exit(1)
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal().Err(err).Msg("An error occurred while executing Ubuntu Insights.")
		os.Exit(1)
	}
}

func setVerbosity() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Debug().Msg("Verbose logging enabled")
	}
}
