package commands

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ubuntu/ubuntu-insights/insights/internal/systemconfig"
)

func installSystemOptOutCmd(app *App) {
	systemOptOutCmd := &cobra.Command{
		Use:   "system-opt-out",
		Short: "Manage or get the system-wide opt-out state",
		Long: `Manage or get the system-wide opt-out state for data collection and upload.

When the system opt-out is active, all collection and upload operations behave as if consent is denied, regardless of per-user or per-source consent settings.

Setting the system opt-out state typically requires administrative privileges to write to the system configuration directory.`,
		Args: cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if _, err := strconv.ParseBool(app.config.SystemOptOut.State); app.config.SystemOptOut.State != "" && err != nil {
				app.cmd.SilenceUsage = false
				return fmt.Errorf("state must be either true or false, or not set: %v", err)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Ensure the state is case insensitive.
			app.config.SystemOptOut.State = strings.ToLower(app.config.SystemOptOut.State)

			// Warn about persistent flags that have no effect on this command.
			if cmd.Root().PersistentFlags().Changed("consent-dir") {
				slog.Warn("The consent-dir flag was provided but it is not used in the system-opt-out command")
			}
			if cmd.Root().PersistentFlags().Changed("insights-dir") {
				slog.Warn("The insights-dir flag was provided but it is not used in the system-opt-out command")
			}

			slog.Debug("Running system-opt-out command")

			sm := systemconfig.New(slog.Default(), app.config.systemConfigDir)

			// Set the system opt-out state.
			if app.config.SystemOptOut.State != "" {
				// PreRunE already validated the state value, so ParseBool cannot fail here.
				state, _ := strconv.ParseBool(app.config.SystemOptOut.State)
				if err := sm.SetOptOut(state); err != nil {
					return err
				}
			}

			// Get the system opt-out state.
			state, err := sm.IsOptedOut()
			if err != nil {
				return err
			}

			if !app.config.Quiet {
				fmt.Printf("%t\n", state)
			}

			return nil
		},
	}

	systemOptOutCmd.SetHelpFunc(func(command *cobra.Command, strings []string) {
		if err := command.Flags().MarkHidden("consent-dir"); err != nil {
			slog.Error("Failed to hide consent-dir flag", "error", err)
		}
		if err := command.Flags().MarkHidden("insights-dir"); err != nil {
			slog.Error("Failed to hide insights-dir flag", "error", err)
		}
		command.Parent().HelpFunc()(command, strings)
	})

	systemOptOutCmd.Flags().StringVarP(&app.config.SystemOptOut.State, "state", "s", "", "the system opt-out state to set (true or false)")

	app.cmd.AddCommand(systemOptOutCmd)
}
