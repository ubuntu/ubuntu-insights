package commands

import (
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/ubuntu/ubuntu-insights/internal/constants"
)

var consentDir string
var verbose bool

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

	if err != nil {
		err = rootCmd.MarkPersistentFlagRequired("consent-dir")
	}

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
