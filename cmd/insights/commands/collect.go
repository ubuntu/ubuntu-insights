package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/ubuntu/decorate"
	"github.com/ubuntu/ubuntu-insights/internal/collector"
	"github.com/ubuntu/ubuntu-insights/internal/consent"
	"github.com/ubuntu/ubuntu-insights/internal/constants"
)

func installCollectCmd(app *App) {
	collectCmd := &cobra.Command{
		Use:   "collect [source] [source-metrics-path](required if source provided)",
		Short: "Collect system information",
		Long: `Collect system information and metrics and store it locally.

If source is not provided, then the source is assumed to be the currently detected platform. Additionally, there should be no source-metrics-path provided.
If source is provided, then the source-metrics-path should be provided as well.`,
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.MaximumNArgs(2)(cmd, args); err != nil {
				return err
			}

			if len(args) != 0 {
				if err := cobra.MatchAll(cobra.OnlyValidArgs, cobra.ExactArgs(2))(cmd, args); err != nil {
					return fmt.Errorf("accepts no args, or exactly 2 args, received 1")
				}

				fileInfo, err := os.Stat(args[1])
				if err != nil {
					return fmt.Errorf("the second argument, source-metrics-path, should be a valid JSON file. Error: %s", err.Error())
				}

				if fileInfo.IsDir() {
					return fmt.Errorf("the second argument, source-metrics-path, should be a valid JSON file, not a directory")
				}
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Set Sources to Args
			if len(args) == 2 {
				app.config.Collect.Source = args[0]
				app.config.Collect.SourceMetrics = args[1]
			}

			slog.Info("Running collect command")
			return app.collectRun()
		},
	}

	collectCmd.Flags().UintVarP(&app.config.Collect.Period, "period", "p", constants.DefaultPeriod, "the minimum period between 2 collection periods for validation purposes in seconds")
	collectCmd.Flags().BoolVarP(&app.config.Collect.Force, "force", "f", false, "force a collection, override the report if there are any conflicts (doesn't ignore consent)")
	collectCmd.Flags().BoolVarP(&app.config.Collect.DryRun, "dry-run", "d", false, "perform a dry-run where a report is collected, but not written to disk")

	app.cmd.AddCommand(collectCmd)
}

// collectRun runs the collect command.
func (a App) collectRun() (err error) {
	defer decorate.OnError(&err, "failed to collect insights")

	cConfig := a.config.Collect

	err = cConfig.Sanitize()
	if err != nil {
		return err
	}

	cm := consent.New(a.config.consentDir)
	c, err := a.newCollector(cm, a.config.insightsDir, cConfig.Source, cConfig.Period, cConfig.DryRun, collector.WithSourceMetricsPath(cConfig.SourceMetrics))
	if err != nil {
		return err
	}

	insights, err := c.Compile(cConfig.Force)
	if err != nil {
		return err
	}

	ib, err := json.MarshalIndent(insights, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal insights report for console printing: %v", err)
	}
	fmt.Println(string(ib))

	err = c.Write(insights)
	if errors.Is(err, consent.ErrConsentFileNotFound) {
		slog.Warn("Consent file not found, will not write insights report to disk or upload.")
		return nil
	}

	return err
}
