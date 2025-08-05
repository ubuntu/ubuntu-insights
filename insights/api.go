// Package insights contains the Golang bindings: collect and upload system metrics.
package insights

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ubuntu/ubuntu-insights/insights/internal/collector"
	"github.com/ubuntu/ubuntu-insights/insights/internal/consent"
	"github.com/ubuntu/ubuntu-insights/insights/internal/constants"
	"github.com/ubuntu/ubuntu-insights/insights/internal/uploader"
)

// Config represents optional parameters shared by all calls.
type Config struct {
	ConsentDir  string
	InsightsDir string

	Logger *slog.Logger // Optional logger, if not set, a new one will be created.
}

// CollectFlags represents optional parameters for Collect.
type CollectFlags struct {
	SourceMetricsPath string // Path to a JSON file a valid JSON object for source metrics.
	SourceMetricsJSON []byte // JSON object for source metrics.
	Period            uint32
	Force             bool
	DryRun            bool
}

// UploadFlags represents optional parameters for Upload.
type UploadFlags struct {
	MinAge uint32
	Force  bool
	DryRun bool
}

// Resolve returns a copy of the config with default values filled in where necessary.
//
// If ConsentDir is not set, a default path will be used.
// If InsightsDir is not set, a default path will be used.
// If Logger is not set, the global default slog Logger will be used.
func (c Config) Resolve() Config {
	if c.ConsentDir == "" {
		c.ConsentDir = constants.DefaultConfigPath
	}
	if c.InsightsDir == "" {
		c.InsightsDir = constants.DefaultCachePath
	}

	if c.Logger == nil {
		c.Logger = slog.Default()
	}
	return c
}

// Collect creates a report for the specified source and writes it to Config.InsightsDir.
//
// The SourceMetricsPath and SourceMetricsJSON fields in flags are mutually exclusive.
// If both are set, an error will be returned.
// If SourceMetricsPath in flags is set, it must be a valid path to a JSON file with a valid JSON object.
// SourceMetricsJSON in flags if set must be a valid JSON object, not an array or primitive.
//
// Collect returns the compiled insights as a pretty printed JSON byte slice.
// Note that this return may not match with what is written to disk depending on provided flags and the consent state.
// If the consent state is determined to be false, an OptOut report will be written to disk, but the full compiled
// report will still be returned.
//
// This method calls Resolve() on the config before proceeding.
func (c Config) Collect(source string, flags CollectFlags) ([]byte, error) {
	r := c.Resolve()

	cConf := collector.Config{
		Source:            source,
		Period:            flags.Period,
		CachePath:         r.InsightsDir,
		SourceMetricsPath: flags.SourceMetricsPath,
		SourceMetricsJSON: flags.SourceMetricsJSON,
	}

	cm := consent.New(r.Logger, r.ConsentDir)
	col, err := collector.New(r.Logger, cm, cConf)
	if err != nil {
		return nil, err
	}

	insights, err := col.Compile(flags.Force)
	if err != nil {
		return nil, err
	}

	if err := col.Write(insights, flags.DryRun); err != nil {
		return nil, err
	}

	return json.MarshalIndent(insights, "", "  ")
}

// Upload uploads reports for the specified sources.
// If sources is empty, all reports found are handled.
//
// This method calls Resolve() on the config before proceeding.
func (c Config) Upload(sources []string, flags UploadFlags) error {
	r := c.Resolve()

	uConf := uploader.Config{
		Sources: sources,
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

// GetConsentState gets the state for the specified source.
// If source is "", the global source is retrieved.
//
// This method calls Resolve() on the config before proceeding.
func (c Config) GetConsentState(source string) (bool, error) {
	r := c.Resolve()

	cm := consent.New(r.Logger, r.ConsentDir)
	s, err := cm.GetState(source)
	if err != nil {
		return false, fmt.Errorf("failed to get consent state: %v", err)
	}

	return s, nil
}

// SetConsentState sets the consent state for the specified source.
// If source is "", the global source is affected.
//
// This method calls Resolve() on the config before proceeding.
func (c Config) SetConsentState(source string, consentState bool) error {
	r := c.Resolve()

	cm := consent.New(r.Logger, r.ConsentDir)
	return cm.SetState(source, consentState)
}
