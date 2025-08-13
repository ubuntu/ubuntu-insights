// Package collector is the implementation of the collector component.
// The collector component is responsible for collecting data from sources, merging it into a report, and then writing the report to disk.
package collector

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/ubuntu/decorate"
	"github.com/ubuntu/ubuntu-insights/common/fileutils"
	"github.com/ubuntu/ubuntu-insights/insights/internal/collector/sysinfo"
	"github.com/ubuntu/ubuntu-insights/insights/internal/constants"
	"github.com/ubuntu/ubuntu-insights/insights/internal/report"
)

var (
	// ErrDuplicateReport is returned when a report already exists for the current period.
	ErrDuplicateReport = fmt.Errorf("report already exists for this period")
	// ErrSanitizeError is returned when the Config is not properly configured in an unrecoverable manner.
	ErrSanitizeError = fmt.Errorf("collect is not properly configured")
	// ErrSourceMetricsError is returned when the source metrics could not be loaded or parsed.
	ErrSourceMetricsError = fmt.Errorf("source metrics could not be loaded or parsed")
)

// Insights contains the insights report compiled by the collector.
type Insights struct {
	InsightsVersion string         `json:"insightsVersion"`
	CollectionTime  int64          `json:"collectionTime"`
	SysInfo         sysinfo.Info   `json:"systemInfo"`
	SourceMetrics   map[string]any `json:"sourceMetrics,omitempty"`
}

type timeProvider interface {
	Now() time.Time
}

type realTimeProvider struct{}

func (realTimeProvider) Now() time.Time {
	return time.Now()
}

// Consent is an interface for getting the consent state for a given source.
type Consent interface {
	HasConsent(source string) (bool, error)
}

// SysInfo is an interface for collecting system information.
type SysInfo interface {
	Collect() (sysinfo.Info, error)
}

// Collector is an interface for the collector component.
type Collector interface {
	// Compile checks if appropriate to make a new report, and if so, collects and compiles the data into a report.
	Compile() (Insights, error)

	// Write writes the insights report to disk, and cleans up old reports.
	//
	// If force is true, then Write will overwrite any existing reports for a given period.
	// If dryRun is true, then Write does nothing, other than checking consent.
	//
	// Note that duplicity checks and the timestamp in the file name is based on the current time,
	// not the collection time of the Insights report passed.
	Write(insights Insights, period uint32, force, dryRun bool) error
}

// collector is an abstraction of the collector component.
type collector struct {
	consent Consent
	source  string

	collectedDir      string
	uploadedDir       string
	sourceMetricsPath string
	sourceMetricsJSON []byte

	// Overrides for testing.
	maxReports uint32
	time       time.Time
	sysInfo    SysInfo

	log *slog.Logger
}

type options struct {
	// Private members exported for tests.
	maxReports   uint32
	timeProvider timeProvider
	sysInfo      func(*slog.Logger, ...sysinfo.Options) SysInfo
}

var defaultOptions = options{
	maxReports:   constants.MaxReports,
	timeProvider: realTimeProvider{},
	sysInfo: func(l *slog.Logger, opts ...sysinfo.Options) SysInfo {
		return sysinfo.New(l, opts...)
	},
}

// Options represents an optional function to override Collector default values.
type Options func(*options)

// Config represents the collector specific data needed to collect.
type Config struct {
	Source            string
	CachePath         string
	SourceMetricsPath string
	SourceMetricsJSON []byte
}

// Sanitize sets defaults and checks that the Config is properly configured.
func (c *Config) Sanitize(l *slog.Logger) error {
	// Handle global source and source metrics.
	if c.Source == "" { // Default source to platform
		c.Source = constants.DefaultCollectSource
		l.Info("No source provided, defaulting to platform", "source", c.Source)
	}

	if c.SourceMetricsPath != "" && c.SourceMetricsJSON != nil {
		return errors.New("only one of SourceMetricsPath or SourceMetricsJSON can be provided")
	}

	if c.SourceMetricsJSON != nil && !json.Valid(c.SourceMetricsJSON) {
		return errors.New("provided SourceMetricsJSON is not valid JSON")
	}

	if c.CachePath == "" {
		c.CachePath = constants.DefaultCachePath
		l.Info("No cache path provided, defaulting to", "cachePath", c.CachePath)
	}

	return nil
}

// New returns a new Collector.
//
// The internal time used for collecting and writing reports is the current time at the moment of creation of the Collector.
// Sanitize the config before use, but Sanitize may be called beforehand safely.
func New(l *slog.Logger, cm Consent, c Config, args ...Options) (Collector, error) {
	l.Debug("Creating new collector", "source", c.Source)

	if cm == nil {
		return collector{}, fmt.Errorf("consent manager cannot be nil")
	}

	if err := c.Sanitize(l); err != nil {
		return collector{}, errors.Join(ErrSanitizeError, err)
	}

	if err := os.MkdirAll(c.CachePath, 0750); err != nil {
		return collector{}, fmt.Errorf("failed to create cache directory: %v", err)
	}

	opts := defaultOptions
	for _, opt := range args {
		opt(&opts)
	}

	return collector{
		consent: cm,
		source:  c.Source,

		time:              opts.timeProvider.Now(),
		collectedDir:      filepath.Join(c.CachePath, c.Source, constants.LocalFolder),
		uploadedDir:       filepath.Join(c.CachePath, c.Source, constants.UploadedFolder),
		sourceMetricsPath: c.SourceMetricsPath,
		sourceMetricsJSON: c.SourceMetricsJSON,
		maxReports:        opts.maxReports,
		sysInfo:           opts.sysInfo(l),

		log: l,
	}, nil
}

// Compile collects and compiles data into a report.
//
// Compile does not check consent or report duplicity, as this should be done at write time.
// Note that any source metrics must be a valid JSON object, not an array or primitive.
func (c collector) Compile() (insights Insights, err error) {
	c.log.Debug("Collecting data")
	defer decorate.OnError(&err, "insights compile failed")

	insights, err = c.compile()
	if err != nil {
		return Insights{}, fmt.Errorf("failed to compile insights: %w", err) // Need to expose these errors
	}
	c.log.Info("Insights report compiled", "report", insights)

	return insights, nil
}

// Write writes the insights report to disk, and cleans up old reports.
//
// If force is true, then Write will overwrite any existing reports for a given period.
// If dryRun is true, then Write does nothing, other than checking consent.
//
// Note that duplicity checks and the timestamp in the file name is based on the current time,
// not the collection time of the Insights report passed.
func (c collector) Write(insights Insights, period uint32, force, dryRun bool) (err error) {
	c.log.Debug("Writing data", "period", period, "force", force, "dryRun", dryRun)
	defer decorate.OnError(&err, "insights write failed")

	data, err := json.Marshal(insights)
	if err != nil {
		return fmt.Errorf("failed to marshal insights: %v", err)
	}

	consent, err := c.consent.HasConsent(c.source)
	if err != nil {
		return fmt.Errorf("failed to get consent state: %w", err)
	}
	if consent {
		c.log.Info("Consent granted, writing insights report")
	} else {
		c.log.Info("Consent not granted, writing optout instead of insights report")
		data = constants.OptOutPayload
	}

	if !force {
		duplicate, err := c.duplicateExists(period)
		if err != nil {
			return err
		}
		if duplicate {
			return ErrDuplicateReport
		}
	}

	if dryRun {
		c.log.Info("Dry run, not writing insights report")
		return nil
	}

	if err := c.makeDirs(); err != nil {
		return fmt.Errorf("failed to create directories: %v", err)
	}

	if err := c.write(data, period); err != nil {
		return fmt.Errorf("failed to write insights report: %v", err)
	}

	if err := report.Cleanup(c.log, c.collectedDir, c.maxReports); err != nil {
		return fmt.Errorf("failed to clean up old reports: %v", err)
	}

	return nil
}

// makeDirs creates the collected and uploaded directories if they do not already exist.
func (c collector) makeDirs() error {
	for _, dir := range []string{c.collectedDir, c.uploadedDir} {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}
	return nil
}

// duplicateExists returns true if a report for the current period already exists in the uploaded or collected directories.
// Directories are only checked if they exist.
func (c collector) duplicateExists(period uint32) (bool, error) {
	dirs := []string{}
	if _, err := os.Stat(c.collectedDir); !os.IsNotExist(err) {
		dirs = append(dirs, c.collectedDir)
	}
	if _, err := os.Stat(c.uploadedDir); !os.IsNotExist(err) {
		dirs = append(dirs, c.uploadedDir)
	}

	for _, dir := range dirs {
		cReport, err := report.GetForPeriod(c.log, dir, c.time, period)
		if err != nil {
			return false, fmt.Errorf("failed to check for duplicate report in %s for period: %v", dir, err)
		}
		if cReport.Name != "" {
			c.log.Info("Duplicate report already exists", "file", cReport.Path)
			return true, nil
		}
	}

	return false, nil
}

// compile collects data from sources, and returns an Insights object.
func (c collector) compile() (Insights, error) {
	insights := Insights{
		InsightsVersion: constants.Version,
		CollectionTime:  c.time.Unix(),
	}

	// Collect system information.
	info, err := c.sysInfo.Collect()
	if err != nil {
		return Insights{}, fmt.Errorf("failed to collect system information: %v", err)
	}
	insights.SysInfo = info

	// Load source specific metrics.
	metrics, err := c.getSourceMetrics()
	if err != nil {
		return Insights{}, errors.Join(ErrSourceMetricsError, err)
	}
	insights.SourceMetrics = metrics

	return insights, nil
}

// getSourceMetrics loads source specific metrics.
// If sourceMetricsJSON is set, it will attempt to use that.
// Otherwise, it will use sourceMetricsPath to load from a JSON file.
// If the sourceMetricsPath is empty, it returns nil.
//
// If sourceMetricsJSON is set but not valid JSON, it returns an error.
// If the file does not exist, or cannot be read, it returns an error.
// If the file is not valid JSON, it returns an error.
func (c collector) getSourceMetrics() (map[string]any, error) {
	c.log.Debug("Loading source metrics", "path", c.sourceMetricsPath)

	if c.sourceMetricsJSON != nil {
		var metrics map[string]any
		if err := json.Unmarshal(c.sourceMetricsJSON, &metrics); err != nil {
			return nil, fmt.Errorf("failed to unmarshal source metrics JSON, might be an invalid JSON object: %v", err)
		}
		return metrics, nil
	}

	if c.sourceMetricsPath == "" {
		c.log.Info("No source metrics file provided")
		return nil, nil
	}

	data, err := os.ReadFile(c.sourceMetricsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read source metrics file: %v", err)
	}

	var metrics map[string]any
	if err := json.Unmarshal(data, &metrics); err != nil {
		return nil, fmt.Errorf("failed to unmarshal source metrics, might be an invalid JSON object: %v", err)
	}

	return metrics, nil
}

// write writes the insights report to disk, with the appropriate name.
func (c collector) write(insights []byte, period uint32) error {
	time := report.GetPeriodStart(period, c.time)

	reportPath := filepath.Join(c.collectedDir, fmt.Sprintf("%v.json", time))
	if err := fileutils.AtomicWrite(reportPath, insights); err != nil {
		return fmt.Errorf("failed to write to disk: %v", err)
	}
	c.log.Info("Insights report written", "file", reportPath)

	return nil
}
