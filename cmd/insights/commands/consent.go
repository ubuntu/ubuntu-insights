package commands

import (
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var consentState string

var consentCmd = &cobra.Command{
	Use:   "consent",
	Short: "Manage or get user consent state",
	Long:  "Manage or get user consent state for data collection and upload",
	Run: func(cmd *cobra.Command, args []string) {
		setVerbosity()

		consentState = strings.ToLower(consentState)

		log.Info().Msg("Running consent command")
	},
}

func init() {
	consentCmd.Flags().StringVarP(&source, "source", "s", "", "the name of the source application or event where metrics are to be collected from. If not set, inferred to be global")
	consentCmd.Flags().StringVarP(&consentState, "consent-state", "c", "", "set the user consent state (true or false)")

	rootCmd.AddCommand(consentCmd)
}
