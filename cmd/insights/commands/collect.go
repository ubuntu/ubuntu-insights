package commands

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

type collectConfig struct {
	source       string
	period       uint
	force        bool
	dryRun       bool
	extraMetrics string
}

var defaultCollectConfig = collectConfig{
	source:       "",
	period:       1,
	force:        false,
	dryRun:       false,
	extraMetrics: "",
}

func installCollectCmd(app *App) error {
	app.collectConfig = defaultCollectConfig

	collectCmd := &cobra.Command{
		Use:   "collect [SOURCE](required argument)",
		Short: "Collect system information",
		Long:  "Collect system information and metrics and store it locally",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Set Sources to Args
			app.collectConfig.source = args[0]

			log.Info().Msg("Running collect command")

			return nil
		},
	}

	collectCmd.Flags().UintVarP(&app.collectConfig.period, "period", "p", 1, "The minimum period between 2 collection periods for validation purposes in seconds")
	collectCmd.Flags().BoolVarP(&app.collectConfig.force, "force", "f", false, "Force a collection, override the report if there are any conflicts. (Doesn't ignore consent)")
	collectCmd.Flags().BoolVarP(&app.collectConfig.dryRun, "dry-run", "d", false, "Perform a dry-run where a report is collected, but not written to disk")
	collectCmd.Flags().StringVarP(&app.collectConfig.extraMetrics, "extra-metrics", "e", "", "Path to JSON file to append extra metrics from")

	if err := collectCmd.MarkFlagFilename("extra-metrics", "json"); err != nil {
		log.Fatal().Msg("An error occurred while initializing the collect command.")
		return err
	}

	app.rootCmd.AddCommand(collectCmd)
	return nil
}
