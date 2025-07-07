// Package insights contains the Golang bindings: collect and upload system metrics.
package insights

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/ubuntu/ubuntu-insights/insights/internal/collector"
	"github.com/ubuntu/ubuntu-insights/insights/internal/consent"
	"github.com/ubuntu/ubuntu-insights/insights/internal/constants"
	"github.com/ubuntu/ubuntu-insights/insights/internal/uploader"
)

// Config represents the parameters needed for any call.
type Config struct {
	Source      string
	ConsentDir  string
	InsightsDir string

	Logger *slog.Logger // Optional logger, if not set, a new one will be created.
}

// CollectFlags represents optional parameters for Collect.
type CollectFlags struct {
	SourceMetricsPath string // Path to a JSON file a valid JSON object for source metrics.
	SourceMetricsJSON []byte // JSON object for source metrics.
	Period            uint
	Force             bool
	DryRun            bool
}

// UploadFlags represents optional parameters for Upload.
type UploadFlags struct {
	MinAge uint
	Force  bool
	DryRun bool
}

// Resolve returns a copy of the config with default values filled in where necessary.
//
// If ConsentDir is not set, a default path will be used.
// If InsightsDir is not set, a default path will be used.
// If Logger is not set, a default logger will be created.
func (c Config) Resolve() Config {
	if c.ConsentDir == "" {
		c.ConsentDir = constants.DefaultConfigPath
	}
	if c.InsightsDir == "" {
		c.InsightsDir = constants.DefaultCachePath
	}

	if c.Logger == nil {
		c.Logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
	return c
}

// Collect creates a report for Config.Source and writes it to Config.InsightsDir.
//
// The SourceMetricsPath and SourceMetricsJSON fields in flags are mutually exclusive.
// If both are set, an error will be returned.
// If SourceMetricsPath in flags is set, it must be a valid path to a JSON file with a valid JSON object.
// SourceMetricsJSON in flags if set must be a valid JSON object, not an array or primitive.
//
// This method calls Resolve() on the config before proceeding.
func (c Config) Collect(flags CollectFlags) error {
	r := c.Resolve()

	cConf := collector.Config{
		Source:            r.Source,
		Period:            flags.Period,
		CachePath:         r.InsightsDir,
		SourceMetricsPath: flags.SourceMetricsPath,
		SourceMetricsJSON: flags.SourceMetricsJSON,
	}

	cm := consent.New(r.Logger, r.ConsentDir)
	col, err := collector.New(r.Logger, cm, cConf)
	if err != nil {
		return err
	}

	insights, err := col.Compile(flags.Force)
	if err != nil {
		return err
	}

	return col.Write(insights, flags.DryRun)
}

// Upload uploads reports for Config.Source.
// If source is "", all reports found irregardless of the source are handled.
//
// This method calls Resolve() on the config before proceeding.
func (c Config) Upload(flags UploadFlags) error {
	r := c.Resolve()

	uConf := uploader.Config{
		Sources: []string{r.Source},
		MinAge:  flags.MinAge,
		Force:   flags.Force,
		DryRun:  flags.DryRun,
		Retry:   false,
	}
	err := uConf.Sanitize(r.Logger, r.ConsentDir)
	if err != nil {
		return err
	}

	cm := consent.New(r.Logger, r.ConsentDir)
	uploader, err := uploader.New(r.Logger, cm, r.InsightsDir, uConf.MinAge, uConf.DryRun)
	if err != nil {
		return fmt.Errorf("failed to create uploader: %v", err)
	}

	return uploader.UploadAll(uConf.Sources, uConf.Force, uConf.Retry)
}

// GetConsentState gets the state for Config.Source.
// If source is "", the global source is retrieved.
//
// This method calls Resolve() on the config before proceeding.
func (c Config) GetConsentState() (bool, error) {
	r := c.Resolve()

	cm := consent.New(r.Logger, r.ConsentDir)
	s, err := cm.GetState(r.Source)
	if err != nil {
		return false, fmt.Errorf("failed to get consent state: %v", err)
	}

	return s, nil
}

// SetConsentState sets the state for Config.Source to consent.
// If source is "", the global source is effected.
//
// This method calls Resolve() on the config before proceeding.
func (c Config) SetConsentState(consentState bool) error {
	r := c.Resolve()

	cm := consent.New(r.Logger, r.ConsentDir)
	return cm.SetState(r.Source, consentState)
}
