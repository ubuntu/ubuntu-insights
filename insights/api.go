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
	Verbose     bool
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

// Collect creates a report for Config.Source and writes it to Config.InsightsDir.
//
// The SourceMetricsPath and SourceMetricsJSON fields in flags are mutually exclusive.
// If both are set, an error will be returned.
// If SourceMetricsPath in flags is set, it must be a valid path to a JSON file with a valid JSON object.
// SourceMetricsJSON in flags if set must be a valid JSON object, not an array or primitive.
//
// returns an error if collection fails.
func (c Config) Collect(flags CollectFlags) error {
	l := c.setup()

	cConf := collector.Config{
		Source:            c.Source,
		Period:            flags.Period,
		CachePath:         c.InsightsDir,
		SourceMetricsPath: flags.SourceMetricsPath,
		SourceMetricsJSON: flags.SourceMetricsJSON,
	}

	cm := consent.New(l, c.ConsentDir)
	col, err := collector.New(l, cm, cConf)
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
// if Config.Source is "", all reports are uploaded.
// returns an error if uploading fails.
func (c Config) Upload(flags UploadFlags) error {
	l := c.setup()

	uConf := uploader.Config{
		Sources: []string{c.Source},
		MinAge:  flags.MinAge,
		Force:   flags.Force,
		DryRun:  flags.DryRun,
		Retry:   false,
	}
	err := uConf.Sanitize(l, c.ConsentDir)
	if err != nil {
		return err
	}

	cm := consent.New(l, c.ConsentDir)
	uploader, err := uploader.New(l, cm, c.InsightsDir, uConf.MinAge, uConf.DryRun)
	if err != nil {
		return fmt.Errorf("failed to create uploader: %v", err)
	}

	return uploader.UploadAll(uConf.Sources, uConf.Force, uConf.Retry)
}

// GetConsentState gets the state for Config.Source.
// if Config.Source is "", the global source is retrieved.
func (c Config) GetConsentState() (bool, error) {
	l := c.setup()

	cm := consent.New(l, c.ConsentDir)
	s, err := cm.GetState(c.Source)
	if err != nil {
		return false, fmt.Errorf("failed to get consent state: %v", err)
	}

	return s, nil
}

// SetConsentState sets the state for Config.Source to consent.
// if Config.Source is "", the global source is effected.
// returns an error if the state could not be set.
func (c Config) SetConsentState(consentState bool) error {
	l := c.setup()

	cm := consent.New(l, c.ConsentDir)
	return cm.SetState(c.Source, consentState)
}

// newLogger sets the logging verbosity.
func newLogger(verbose bool) *slog.Logger {
	var hOpts slog.HandlerOptions
	if verbose {
		hOpts.Level = slog.LevelDebug
	} else {
		hOpts.Level = slog.LevelWarn
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &hOpts))
}

// setup sets defaults and creates a new logger, mutating the Config.
func (c *Config) setup() *slog.Logger {
	if c.ConsentDir == "" {
		c.ConsentDir = constants.DefaultConfigPath
	}
	if c.InsightsDir == "" {
		c.InsightsDir = constants.DefaultCachePath
	}

	return newLogger(c.Verbose)
}
