package commands

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/ubuntu/ubuntu-insights/internal/consent"
	"github.com/ubuntu/ubuntu-insights/internal/uploader"
)

const defaultMinAge = 604800

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

			if len(args) == 0 {
				slog.Info("No sources provided, uploading all sources")
				var err error
				args, err = uploader.GetAllSources(app.config.insightsDir)
				if err != nil {
					return fmt.Errorf("failed to get all sources: %v", err)
				}
			}
			app.config.Upload.Sources = args

			slog.Info("Running upload command")
			return app.uploadRun()
		},
	}

	uploadCmd.Flags().UintVar(&app.config.Upload.MinAge, "min-age", defaultMinAge, "the minimum age (in seconds) of a report before the uploader will attempt to upload it")
	uploadCmd.Flags().BoolVarP(&app.config.Upload.Force, "force", "f", false, "force an upload, ignoring min age and clashes between the collected file and a file in the uploaded folder, replacing the clashing uploaded report if it exists")
	uploadCmd.Flags().BoolVarP(&app.config.Upload.DryRun, "dry-run", "d", false, "go through the motions of doing an upload, but do not communicate with the server or send the payload")

	app.cmd.AddCommand(uploadCmd)
}

func (a App) uploadRun() error {
	cm := consent.New(a.config.consentDir)

	for _, source := range a.config.Upload.Sources {
		u, err := a.newUploader(cm, a.config.insightsDir, source, a.config.Upload.MinAge, a.config.Upload.DryRun)
		if err != nil {
			return fmt.Errorf("failed to create uploader for source %s: %v", source, err)
		}

		if err := u.Upload(a.config.Upload.Force); err != nil {
			return fmt.Errorf("failed to upload reports for source %s: %v", source, err)
		}
	}

	return nil
}
