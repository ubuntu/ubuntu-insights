package commands

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/ubuntu/ubuntu-insights/internal/consent"
	"github.com/ubuntu/ubuntu-insights/internal/constants"
	"github.com/ubuntu/ubuntu-insights/internal/uploader"
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

type upload interface {
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
			if args == nil {
				slog.Info("No sources provided, uploading all sources")
				var err error
				args, err = uploader.GetAllSources(app.rootConfig.InsightsDir)
				if err != nil {
					return fmt.Errorf("failed to get all sources: %v", err)
				}
			}
			app.uploadConfig.sources = args

			slog.Info("Running upload command")
			return app.uploadRun()
		},
	}

	uploadCmd.Flags().StringVar(&app.uploadConfig.server, "server", app.uploadConfig.server, "the base URL of the server to upload the metrics to")
	uploadCmd.Flags().UintVar(&app.uploadConfig.minAge, "min-age", app.uploadConfig.minAge, "the minimum age (in seconds) of a report before the uploader will attempt to upload it")
	uploadCmd.Flags().BoolVarP(&app.uploadConfig.force, "force", "f", app.uploadConfig.force, "force an upload, ignoring min age and clashes between the collected file and a file in the uploaded folder, replacing the clashing uploaded report if it exists")
	uploadCmd.Flags().BoolVarP(&app.uploadConfig.dryRun, "dry-run", "d", app.uploadConfig.dryRun, "go through the motions of doing an upload, but do not communicate with the server or send the payload")

	app.cmd.AddCommand(uploadCmd)
}

func (a App) uploadRun() error {
	cm := consent.New(a.rootConfig.ConsentDir)
	opts := uploader.WithCachePath(a.rootConfig.InsightsDir)

	for _, source := range a.uploadConfig.sources {
		u, err := uploader.New(cm, source, a.uploadConfig.minAge, a.uploadConfig.dryRun, opts)
		if err != nil {
			return fmt.Errorf("failed to create uploader for source %s: %v", source, err)
		}

		if err := u.Upload(a.uploadConfig.force); err != nil {
			return fmt.Errorf("failed to upload reports for source %s: %v", source, err)
		}
	}

	return nil
}
