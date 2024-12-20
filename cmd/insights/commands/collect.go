package commands

import (
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var period uint

var collectCmd = &cobra.Command{
	Use:   "collect",
	Short: "Collect system information",
	Long:  "Collect system information and metrics and store it locally",
	Run: func(cmd *cobra.Command, args []string) {
		setVerbosity()
		log.Info().Msg("Running collect command")
	},
}

func init() {
	collectCmd.Flags().UintVarP(&period, "period", "p", 1, "the periodicity of the collection in seconds")
	collectCmd.Flags().BoolVarP(&force, "force", "f", false, "force collection even if the period has not elapsed, overriding any conflicts")
	collectCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "perform a dry run of the collection without storing the data on disk")
	collectCmd.Flags().StringVar(&dir, "dir", "", "the directory to store the collected data and to check for existing data")
	collectCmd.Flags().StringVarP(&source, "source", "s", "", "the name of the source application or event where metrics are to be collected from")

	err := collectCmd.MarkFlagRequired("source")

	if err != nil {
		log.Fatal().Err(err).Msg("An error occurred while initializing the collect command.")
		os.Exit(1)
	}

	err = collectCmd.MarkFlagDirname("dir")

	if err != nil {
		log.Fatal().Err(err).Msg("An error occurred while initializing the collect command.")
		os.Exit(1)
	}

	rootCmd.AddCommand(collectCmd)
}
