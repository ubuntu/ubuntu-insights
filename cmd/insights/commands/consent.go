package commands

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ubuntu/ubuntu-insights/internal/consent"
	"github.com/ubuntu/ubuntu-insights/internal/constants"
)

func installConsentCmd(app *App) {
	consentCmd := &cobra.Command{
		Use:   "consent [SOURCES](optional arguments)",
		Short: "Manage or get user consent state",
		Long:  "Manage or get user consent state for data collection and upload",
		Args:  cobra.ArbitraryArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			app.cmd.SilenceUsage = false

			if _, err := strconv.ParseBool(app.config.consent.state); app.config.consent.state != "" && err != nil {
				return fmt.Errorf("consent-state must be either true or false, or not set")
			}

			app.cmd.SilenceUsage = true
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Set Sources to Args
			app.config.consent.sources = args

			// Ensure consent state is case insensitive
			app.config.consent.state = strings.ToLower(app.config.consent.state)

			// If insights-dir is set, warn the user that it is not used
			if app.config.insightsDir != constants.DefaultCachePath {
				slog.Warn("The insights-dir flag was provided but it is not used in the consent command")
			}

			slog.Debug("Running consent command")
			return app.consentRun()
		},
	}

	consentCmd.Flags().StringVarP(&app.config.consent.state, "consent-state", "c", "", "the consent state to set (true or false), the current consent state is displayed if not set")

	app.cmd.AddCommand(consentCmd)
}

func (a App) consentRun() error {
	cm := consent.New(a.config.consentDir)

	if len(a.config.consent.sources) == 0 {
		// Global consent state to be changed
		a.config.consent.sources = append(a.config.consent.sources, "")
	}

	// Set consent state
	if a.config.consent.state != "" {
		state, err := strconv.ParseBool(a.config.consent.state)
		if err != nil {
			a.cmd.SilenceUsage = false
			return fmt.Errorf("consent-state must be either true or false, or not set")
		}

		for _, source := range a.config.consent.sources {
			if err := cm.SetState(source, state); err != nil {
				return err
			}
		}
	}

	// Get consent state
	for _, source := range a.config.consent.sources {
		state, err := cm.GetState(source)
		if err != nil {
			return err
		}

		if source == "" {
			source = "Global"
		}
		fmt.Printf("%s: %t\n", source, state)
	}

	return nil
}
