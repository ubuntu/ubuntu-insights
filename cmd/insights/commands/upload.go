package commands

import (
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/ubuntu/ubuntu-insights/internal/constants"
	"github.com/ubuntu/ubuntu-insights/internal/uploader"
)

func installUploadCmd(app *App) {
	uploadCmd := &cobra.Command{
		Use:   "upload [sources](optional arguments)",
		Short: "Upload metrics to the Ubuntu Insights server",
		Long: `Upload metrics to the Ubuntu Insights server.
		
If no sources are provided, all detected sources at the configured reports directory will be uploaded.
If consent is not given for a source, an opt-out notification will be sent regardless of the locally cached insights report's contents.`,
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

	uploadCmd.Flags().UintVar(&app.config.Upload.MinAge, "min-age", constants.defaultMinAge, "the minimum age (in seconds) of a report before the uploader will attempt to upload it")
	uploadCmd.Flags().BoolVarP(&app.config.Upload.Force, "force", "f", false, "force an upload, ignoring min age and clashes between the collected file and a file in the uploaded folder, replacing the clashing uploaded report if it exists (doesn't ignore consent)")
	uploadCmd.Flags().BoolVarP(&app.config.Upload.DryRun, "dry-run", "d", false, "go through the motions of doing an upload, but do not communicate with the server, send the payload, or modify local files")
	uploadCmd.Flags().BoolVarP(&app.config.Upload.Retry, "retry", "r", false, "enable a limited number of retries for failed uploads")

	app.cmd.AddCommand(uploadCmd)
}
