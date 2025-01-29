package commands

import (
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/ubuntu/ubuntu-insights/internal/constants"
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
	server:  constants.DefaultServerURL,
	minAge:  604800,
	force:   false,
	dryRun:  false,
}

func installUploadCmd(app *App) {
	app.uploadConfig = defaultUploadConfig

	uploadCmd := &cobra.Command{
		Use:   "upload [sources](optional arguments)",
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
	uploadCmd.Flags().UintVar(&app.uploadConfig.minAge, "min-age", app.uploadConfig.minAge, "the minimum age (in seconds) of a report before the uploader will attempt to upload it")
	uploadCmd.Flags().BoolVarP(&app.uploadConfig.force, "force", "f", app.uploadConfig.force, "force an upload, ignoring min age and clashes between the collected file and a file in the uploaded folder, replacing the clashing uploaded report if it exists")
	uploadCmd.Flags().BoolVarP(&app.uploadConfig.dryRun, "dry-run", "d", app.uploadConfig.dryRun, "go through the motions of doing an upload, but do not communicate with the server or send the payload")

	app.cmd.AddCommand(uploadCmd)
}
