package commands

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var sources []string
var server string
var minAge uint

var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload metrics to the Ubuntu Insights server",
	Long:  "Upload metrics to the Ubuntu Insights server",
	Run: func(cmd *cobra.Command, args []string) {
		setVerbosity()
		log.Info().Msg("Running upload command")
	},
}

func init() {
	uploadCmd.Flags().StringArrayVarP(&sources, "source", "s", []string{""}, "the name of the source application(s) or event(s)")
	uploadCmd.Flags().StringVar(&server, "server", "https://metrics.ubuntu.com", "the base URL of the server to upload the metrics to")
	uploadCmd.Flags().UintVar(&minAge, "min-age", 604800, "the minimum age of the metrics to upload in seconds")

	uploadCmd.Flags().BoolVarP(&force, "force", "f", false, "force upload even if the period has not elapsed, overriding any conflicts")
	uploadCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "perform a dry run of the upload without sending the data to the server")
	uploadCmd.Flags().StringVar(&dir, "dir", "", "the directory in which to look for the collected/uploaded folders or valid source folders")

	err := uploadCmd.MarkFlagDirname("dir")

	if err != nil {
		log.Fatal().Err(err).Msg("An error occurred while initializing the upload command.")
	}

	rootCmd.AddCommand(uploadCmd)
}
