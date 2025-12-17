package commands

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ubuntu/ubuntu-insights/insights/internal/consent"
	"github.com/ubuntu/ubuntu-insights/insights/internal/constants"
)

func installConsentCmd(app *App) {
	consentCmd := &cobra.Command{
		Use:   "consent [sources](optional arguments)",
		Short: "Manage or get user consent state",
		Long: `Manage or get user consent state for data collection and upload.
		
If no sources are provided, the default consent state is managed.`,
		Args: cobra.ArbitraryArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if _, err := strconv.ParseBool(app.config.Consent.State); app.config.Consent.State != "" && err != nil {
				app.cmd.SilenceUsage = false
				return fmt.Errorf("consent-state must be either true or false, or not set: %v", err)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Set Sources to Args
			if len(args) > 0 {
				// Persist viper config if no args passed
				app.config.Consent.Sources = args
			}

			// Ensure consent state is case insensitive
			app.config.Consent.State = strings.ToLower(app.config.Consent.State)

			// If insights-dir is set, warn the user that it is not used
			if app.config.insightsDir != constants.DefaultCachePath {
				slog.Warn("The insights-dir flag was provided but it is not used in the consent command")
			}

			slog.Debug("Running consent command")
			return app.consentRun()
		},
	}

	consentCmd.SetHelpFunc(func(command *cobra.Command, strings []string) {
		if err := command.Flags().MarkHidden("insights-dir"); err != nil {
			slog.Error("Failed to hide insights-dir flag", "error", err)
		}
		command.Parent().HelpFunc()(command, strings)
	})

	consentCmd.Flags().StringVarP(&app.config.Consent.State, "state", "s", "", "the consent state to set (true or false)")

	app.cmd.AddCommand(consentCmd)
}

func (a App) consentRun() error {
	cm := consent.New(slog.Default(), a.config.consentDir)

	if len(a.config.Consent.Sources) == 0 {
		// Change default consent state
		a.config.Consent.Sources = append(a.config.Consent.Sources, "")
	}

	// Set consent state
	if a.config.Consent.State != "" {
		state, err := strconv.ParseBool(a.config.Consent.State)
		if err != nil {
			a.cmd.SilenceUsage = false
			return fmt.Errorf("consent-state must be either true or false, or not set")
		}

		for _, source := range a.config.Consent.Sources {
			if err := cm.SetState(source, state); err != nil {
				return err
			}
		}
	}

	// Get consent state
	var failedSources []string
	for _, source := range a.config.Consent.Sources {
		state, err := cm.GetState(source)
		if source == "" {
			source = "Default"
		}
		if err != nil {
			slog.Error("Failed to get consent state for source", "source", source, "error", err)
			failedSources = append(failedSources, source)
			continue
		}

		if !a.config.Quiet {
			fmt.Printf("%s: %t\n", source, state)
		}
	}

	if len(failedSources) > 0 {
		return fmt.Errorf("failed to get consent state for sources: %s", strings.Join(failedSources, ", "))
	}
	return nil
}
