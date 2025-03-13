package commands

import (
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/ubuntu/ubuntu-insights/internal/constants"
	"github.com/ubuntu/ubuntu-insights/internal/uploader"
)

func installUploadCmd(app *App) {
	uploadCmd := &cobra.Command{
		Use:  "upload [sources](optional arguments)",
		Long: "Upload metrics to the Ubuntu Insights server",
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Persist viper config if no args passed
			if len(args) == 0 && len(app.config.Upload.Sources) > 0 {
				args = app.config.Upload.Sources
			}

			app.config.Upload.Sources = args

			slog.Info("Running upload command")
			return app.config.Upload.Run(app.config.consentDir, app.config.insightsDir, func(f *uploader.Factory) {
				*f = app.newUploader
			})
		},
	}

	uploadCmd.Flags().UintVar(&app.config.Upload.MinAge, "min-age", constants.DefaultMinAge, "the minimum age (in seconds) of a report before the uploader will attempt to upload it")
	uploadCmd.Flags().BoolVarP(&app.config.Upload.Force, "force", "f", false, "force an upload, ignoring min age and clashes between the collected file and a file in the uploaded folder, replacing the clashing uploaded report if it exists")
	uploadCmd.Flags().BoolVarP(&app.config.Upload.DryRun, "dry-run", "d", false, "go through the motions of doing an upload, but do not communicate with the server or send the payload")
	uploadCmd.Flags().BoolVarP(&app.config.Upload.Retry, "retry", "r", false, "enable a limited number of retries for failed uploads")

	app.cmd.AddCommand(uploadCmd)
}
