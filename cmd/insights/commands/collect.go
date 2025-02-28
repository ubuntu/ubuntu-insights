package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/ubuntu/decorate"
	"github.com/ubuntu/ubuntu-insights/internal/collector"
	"github.com/ubuntu/ubuntu-insights/internal/consent"
)

func installCollectCmd(app *App) {
	collectCmd := &cobra.Command{
		Use:   "collect [SOURCE] [SOURCE-METRICS-PATH](required if source provided)",
		Short: "Collect system information",
		Long: `Collect system information and metrics and store it locally.
		If SOURCE is not provided, then it is the source is assumed to be the currently detected platform. Additionally, there should be no SOURCE-METRICS-PATH provided.
		If SOURCE is provided, then the SOURCE-METRICS-PATH should be provided as well.`,
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
			// Default source to platform
			app.config.collect.source = runtime.GOOS

			// Set Sources to Args
			if len(args) == 2 {
				app.config.collect.source = args[0]
				app.config.collect.sourceMetrics = args[1]
			}

			slog.Info("Running collect command")
			return app.collectRun()
		},
	}

	collectCmd.Flags().UintVarP(&app.config.collect.period, "period", "p", 1, "the minimum period between 2 collection periods for validation purposes in seconds")
	collectCmd.Flags().BoolVarP(&app.config.collect.force, "force", "f", false, "force a collection, override the report if there are any conflicts (doesn't ignore consent)")
	collectCmd.Flags().BoolVarP(&app.config.collect.dryRun, "dry-run", "d", false, "perform a dry-run where a report is collected, but not written to disk")

	app.cmd.AddCommand(collectCmd)
}

// collectRun runs the collect command.
func (a App) collectRun() (err error) {
	defer decorate.OnError(&err, "failed to collect insights")

	cConfig := a.config.collect
	cm := consent.New(a.config.consentDir)

	opts := []collector.Options{}
	if cConfig.sourceMetrics != "" {
		opts = append(opts, collector.WithSourceMetricsPath(cConfig.sourceMetrics))
	}

	c, err := a.newCollector(cm, a.config.insightsDir, cConfig.source, cConfig.period, cConfig.dryRun, opts...)
	if err != nil {
		return err
	}

	insights, err := c.Compile(cConfig.force)
	if err != nil {
		return err
	}

	// Pretty print insights
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, insights, "", "    "); err != nil {
		return err
	}
	fmt.Println(prettyJSON.String())

	err = c.Write(insights)

	return err
}
