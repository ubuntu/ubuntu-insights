package commands

import (
	"log/slog"

	"github.com/spf13/cobra"
)

type uploadConfig struct {
	sources []string
	server  string
	minAge  uint
	force   bool
	dryRun  bool
}

var defaultUploadConfig = uploadConfig{
	sources: []string{""},
	server:  "https://metrics.ubuntu.com",
	minAge:  604800,
	force:   false,
	dryRun:  false,
}

func installUploadCmd(app *App) {
	app.uploadConfig = defaultUploadConfig

	uploadCmd := &cobra.Command{
		Use:   "upload upload [sources](optional arguments)",
		Short: "Upload metrics to the Ubuntu Insights server",
		Long:  "Upload metrics to the Ubuntu Insights server",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Set Sources to Args
			app.uploadConfig.sources = args

			slog.Info("Running upload command")

			return nil
		},
	}

	uploadCmd.Flags().StringVar(&app.uploadConfig.server, "server", app.uploadConfig.server, "the base URL of the server to upload the metrics to")
	uploadCmd.Flags().UintVar(&app.uploadConfig.minAge, "min-age", app.uploadConfig.minAge, "the minimum age of the metrics to upload in seconds")
	uploadCmd.Flags().BoolVarP(&app.uploadConfig.force, "force", "f", app.uploadConfig.force, "force upload even if the period has not elapsed, overriding any conflicts")
	uploadCmd.Flags().BoolVarP(&app.uploadConfig.dryRun, "dry-run", "d", app.uploadConfig.dryRun, "perform a dry run of the upload without sending the data to the server")

	app.rootCmd.AddCommand(uploadCmd)
}
