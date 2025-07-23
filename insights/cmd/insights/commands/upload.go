package commands

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/ubuntu/ubuntu-insights/insights/internal/consent"
	"github.com/ubuntu/ubuntu-insights/insights/internal/constants"
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
			return app.uploadRun()
		},
	}

	uploadCmd.Flags().Uint32Var(&app.config.Upload.MinAge, "min-age", constants.DefaultMinAge, "the minimum age (in seconds) of a report before the uploader will attempt to upload it")
	uploadCmd.Flags().BoolVarP(&app.config.Upload.Force, "force", "f", false, "force an upload, ignoring min age and clashes between the collected file and a file in the uploaded folder, replacing the clashing uploaded report if it exists (doesn't ignore consent)")
	uploadCmd.Flags().BoolVarP(&app.config.Upload.DryRun, "dry-run", "d", false, "go through the motions of doing an upload, but do not communicate with the server, send the payload, or modify local files")
	uploadCmd.Flags().BoolVarP(&app.config.Upload.Retry, "retry", "r", false, "enable a limited number of retries for failed uploads")

	app.cmd.AddCommand(uploadCmd)
}

// uploadRun runs the upload command.
func (a App) uploadRun() error {
	l := slog.Default()
	uConfig := a.config.Upload
	err := uConfig.Sanitize(l, a.config.insightsDir)
	if err != nil {
		return err
	}

	cm := consent.New(l, a.config.consentDir)
	u, err := a.newUploader(l, cm, a.config.insightsDir, uConfig.MinAge, uConfig.DryRun)
	if err != nil {
		return fmt.Errorf("failed to create uploader: %v", err)
	}

	return u.UploadAll(uConfig.Sources, uConfig.Force, uConfig.Retry)
}
