package commands

import (
	"log/slog"
	"strings"

	"github.com/spf13/cobra"
)

type consentConfig struct {
	sources      []string
	consentState string
}

var defaultConsentConfig = consentConfig{
	sources:      []string{""},
	consentState: "",
}

func installConsentCmd(app *App) {
	app.consentConfig = defaultConsentConfig

	consentCmd := &cobra.Command{
		Use:   "consent [SOURCES](optional arguments)",
		Short: "Manage or get user consent state",
		Long:  "Manage or get user consent state for data collection and upload",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Set Sources to Args
			app.consentConfig.sources = args

			// Ensure consent state is case insensitive
			app.consentConfig.consentState = strings.ToLower(app.consentConfig.consentState)

			// If insights-dir is set, warn the user that it is not used
			if app.rootConfig.InsightsDir != defaultRootConfig.InsightsDir {
				slog.Warn("The insights-dir flag was provided but it is not used in the consent command")
			}

			slog.Info("Running consent command")
			return nil
		},
	}

	consentCmd.Flags().StringVarP(&app.consentConfig.consentState, "consent-state", "c", "", "the consent state to set (true or false), the current consent state is displayed if not set")

	app.rootCmd.AddCommand(consentCmd)
}
