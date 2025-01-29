package commands

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ubuntu/ubuntu-insights/internal/consent"
)

type consentConfig struct {
	sources      []string
	consentState string
}

func installConsentCmd(app *App) {
	app.consentConfig = consentConfig{
		sources:      []string{""},
		consentState: "",
	}

	consentCmd := &cobra.Command{
		Use:   "consent [SOURCES](optional arguments)",
		Short: "Manage or get user consent state",
		Long:  "Manage or get user consent state for data collection and upload",
		Args:  cobra.ArbitraryArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			app.cmd.SilenceUsage = false

			if _, err := strconv.ParseBool(app.consentConfig.consentState); app.consentConfig.consentState != "" && err != nil {
				return fmt.Errorf("consent-state must be either true or false, or not set")
			}

			app.cmd.SilenceUsage = true
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Set Sources to Args
			app.consentConfig.sources = args

			// Ensure consent state is case insensitive
			app.consentConfig.consentState = strings.ToLower(app.consentConfig.consentState)

			// If insights-dir is set, warn the user that it is not used
			if app.rootConfig.InsightsDir != defaultRootConfig.InsightsDir {
				slog.Warn("The insights-dir flag was provided but it is not used in the consent command")
			}

			slog.Debug("Running consent command")
			return app.consentRun()
		},
	}

	consentCmd.Flags().StringVarP(&app.consentConfig.consentState, "consent-state", "c", "", "the consent state to set (true or false), the current consent state is displayed if not set")

	app.cmd.AddCommand(consentCmd)
}

func (a App) consentRun() error {
	cm := consent.New(a.rootConfig.ConsentDir)

	if len(a.consentConfig.sources) == 0 {
		// Global consent state to be changed
		a.consentConfig.sources = append(a.consentConfig.sources, "")
	}

	// Set consent state
	if a.consentConfig.consentState != "" {
		state, err := strconv.ParseBool(a.consentConfig.consentState)
		if err != nil {
			a.cmd.SilenceUsage = false
			return fmt.Errorf("consent-state must be either true or false, or not set")
		}

		for _, source := range a.consentConfig.sources {
			if err := cm.SetState(source, state); err != nil {
				return err
			}
		}
	}

	// Get consent state
	for _, source := range a.consentConfig.sources {
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
