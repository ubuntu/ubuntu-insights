// Package insights contains the Golang bindings: collect and upload system metrics.
package insights

import (
	"bytes"
	"encoding/json"
	"errors"
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

// CompileFlags represents optional parameters for Compile.
type CompileFlags struct {
	SourceMetricsPath string // Path to a JSON file a valid JSON object for source metrics.
	SourceMetricsJSON []byte // JSON object for source metrics.
}

// WriteFlags represents optional parameters for Write.
type WriteFlags struct {
	Period uint32
	Force  bool
	DryRun bool
}

// UploadFlags represents optional parameters for Upload.
type UploadFlags struct {
	MinAge uint32
	Force  bool
	DryRun bool
}

// Collect errors.
var (
	// ErrDuplicateReport is returned by Collect when a report for the specified period already exists.
	ErrDuplicateReport = collector.ErrDuplicateReport
	// ErrSanitizeError is returned by Collect when the Config is not properly configured in an unrecoverable manner.
	ErrSanitizeError = collector.ErrSanitizeError
	// ErrSourceMetricsError is returned by Collect when the source metrics could not be loaded or parsed.
	ErrSourceMetricsError = collector.ErrSourceMetricsError
)

// Consent errors.
var (
	// ErrConsentFileNotFound is returned by Consent when the consent file is not found.
	ErrConsentFileNotFound = consent.ErrConsentFileNotFound
)

// Upload errors.
var (
	// ErrSendFailure is returned by Upload when a report fails to be sent to the server, either due to a network error or a non-200 status code.
	ErrSendFailure = uploader.ErrSendFailure
)

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
		CachePath:         r.InsightsDir,
		SourceMetricsPath: flags.SourceMetricsPath,
		SourceMetricsJSON: flags.SourceMetricsJSON,
	}

	cm := consent.New(r.Logger, r.ConsentDir)
	col, err := collector.New(r.Logger, cm, cConf)
	if err != nil {
		return nil, err
	}

	insights, err := col.Compile()
	if err != nil { // Errors may need to be exposed for caller correction.
		return nil, err
	}

	if err := col.Write(insights, flags.Period, flags.Force, flags.DryRun); err != nil {
		if !(flags.DryRun && errors.Is(err, ErrConsentFileNotFound)) {
			return nil, err
		}
	}

	return json.MarshalIndent(insights, "", "  ")
}

// Compile compiles and returns a pretty printed insights report. Consent and duplicity are not checked.
//
// The SourceMetricsPath and SourceMetricsJSON fields in flags are mutually exclusive.
// If both are set, an error will be returned.
// If SourceMetricsPath in flags is set, it must be a valid path to a JSON file with a valid JSON object.
// SourceMetricsJSON in flags if set must be a valid JSON object, not an array or primitive.
func (c Config) Compile(flags CompileFlags) ([]byte, error) {
	r := c.Resolve()

	cConf := collector.Config{
		CachePath:         r.InsightsDir,
		SourceMetricsPath: flags.SourceMetricsPath,
		SourceMetricsJSON: flags.SourceMetricsJSON,
	}

	// TODO: remove consent manager dependency from Compile
	cm := consent.New(r.Logger, r.ConsentDir)
	col, err := collector.New(r.Logger, cm, cConf)
	if err != nil {
		return nil, err
	}

	insights, err := col.Compile()
	if err != nil { // Errors may need to be exposed for caller correction.
		return nil, err
	}

	return json.MarshalIndent(insights, "", "  ")
}

// Write writes a valid insights report to disk based on consent.
//
// If consent is false, an OptOut report will be written to disk.
// If a consent file could not be found, and error is returned.
func (c Config) Write(source string, report []byte, flags WriteFlags) error {
	r := c.Resolve()

	cm := consent.New(r.Logger, r.ConsentDir)
	col, err := collector.New(r.Logger, cm, collector.Config{
		Source:    source,
		CachePath: r.InsightsDir,
	})
	if err != nil {
		return err
	}

	var insightsObj collector.Insights
	dec := json.NewDecoder(bytes.NewReader(report))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&insightsObj); err != nil {
		return err
	}

	return col.Write(insightsObj, flags.Period, flags.Force, flags.DryRun)
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
// If source is "", the default consent state is retrieved.
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
// If source is "", the default consent state is affected.
//
// This method calls Resolve() on the config before proceeding.
func (c Config) SetConsentState(source string, consentState bool) error {
	r := c.Resolve()

	cm := consent.New(r.Logger, r.ConsentDir)
	return cm.SetState(source, consentState)
}
