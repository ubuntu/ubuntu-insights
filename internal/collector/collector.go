// Package collector is the implementation of the collector component.
// The collector component is responsible for collecting data from sources, merging it into a report, and then writing the report to disk.
package collector

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/ubuntu/decorate"
	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo"
	"github.com/ubuntu/ubuntu-insights/internal/constants"
	"github.com/ubuntu/ubuntu-insights/internal/fileutils"
	"github.com/ubuntu/ubuntu-insights/internal/report"
)

// ErrDuplicateReport is returned when a report already exists for the current period.
var ErrDuplicateReport = fmt.Errorf("report already exists for this period")

// Insights contains the insights report compiled by the collector.
type Insights struct {
	InsightsVersion string         `json:"insightsVersion"`
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

// Collector is an abstraction of the collector component.
type Collector struct {
	consent Consent
	period  int
	dryRun  bool
	source  string

	collectedDir      string
	uploadedDir       string
	sourceMetricsPath string
	maxReports        uint
	time              time.Time
	sysInfo           SysInfo
}

type options struct {
	sourceMetricsPath string
	// Private members exported for tests.
	maxReports   uint
	timeProvider timeProvider
	sysInfo      SysInfo
}

var defaultOptions = options{
	maxReports:   constants.MaxReports,
	timeProvider: realTimeProvider{},
	sysInfo:      sysinfo.New(),
}

// Options represents an optional function to override Collector default values.
type Options func(*options)

// WithSourceMetricsPath sets the path to an optional pre-made JSON file containing source specific metrics.
func WithSourceMetricsPath(path string) Options {
	return func(o *options) {
		o.sourceMetricsPath = path
	}
}

// New returns a new Collector.
//
// The internal time used for collecting and writing reports is the current time at the moment of creation of the Collector.
func New(cm Consent, cachePath, source string, period uint, dryRun bool, args ...Options) (Collector, error) {
	slog.Debug("Creating new collector", "source", source, "period", period, "dryRun", dryRun)

	if source == "" {
		return Collector{}, fmt.Errorf("source cannot be an empty string")
	}

	if period > math.MaxInt {
		return Collector{}, fmt.Errorf("period is too large")
	}

	if cm == nil {
		return Collector{}, fmt.Errorf("consent manager cannot be nil")
	}

	if cachePath == "" {
		return Collector{}, fmt.Errorf("cache path cannot be an empty string")
	}

	if err := os.MkdirAll(cachePath, 0750); err != nil {
		return Collector{}, fmt.Errorf("failed to create cache directory: %v", err)
	}

	opts := defaultOptions
	for _, opt := range args {
		opt(&opts)
	}

	return Collector{
		consent: cm,
		period:  int(period),
		dryRun:  dryRun,
		source:  source,

		time:              opts.timeProvider.Now(),
		collectedDir:      filepath.Join(cachePath, source, constants.LocalFolder),
		uploadedDir:       filepath.Join(cachePath, source, constants.UploadedFolder),
		sourceMetricsPath: opts.sourceMetricsPath,
		maxReports:        opts.maxReports,
		sysInfo:           opts.sysInfo,
	}, nil
}

// Compile checks if appropriate to make a new report, and if so, collects and compiles the data into a report.
func (c Collector) Compile(force bool) (insights []byte, err error) {
	slog.Debug("Collecting data", "force", force)
	defer decorate.OnError(&err, "insights compile failed")

	if err := c.makeDirs(); err != nil {
		return nil, err
	}

	if !force {
		duplicate, err := c.duplicateExists()
		if err != nil {
			return nil, err
		}
		if duplicate {
			return nil, ErrDuplicateReport
		}
	}

	consent, err := c.consent.HasConsent(c.source)
	if err != nil {
		return nil, fmt.Errorf("failed to get consent state: %v", err)
	}

	insights, err = json.Marshal(constants.OptOutJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal opt-out JSON: %v", err)
	}
	if consent {
		insights, err = c.compile()
		if err != nil {
			return nil, fmt.Errorf("failed to compile insights: %v", err)
		}
		slog.Info("Insights report compiled", "report", insights)
	}

	return insights, nil
}

// Write writes the insights report to disk, and cleans up old reports.
// Does not check for duplicates, as this should be done in Compile.
//
// If the dryRun is true, then Write does nothing.
func (c Collector) Write(insights []byte) (err error) {
	slog.Debug("Writing data", "dryRun", c.dryRun)
	defer decorate.OnError(&err, "insights write failed")

	if c.dryRun {
		slog.Info("Dry run, not writing insights report")
		return nil
	}

	if err := c.makeDirs(); err != nil {
		return fmt.Errorf("failed to create directories: %v", err)
	}

	if err := c.write(insights); err != nil {
		return fmt.Errorf("failed to write insights report: %v", err)
	}

	if err := report.Cleanup(c.collectedDir, c.maxReports); err != nil {
		return fmt.Errorf("failed to clean up old reports: %v", err)
	}

	return nil
}

// makeDirs creates the collected and uploaded directories if they do not already exist.
func (c Collector) makeDirs() error {
	for _, dir := range []string{c.collectedDir, c.uploadedDir} {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}
	return nil
}

// duplicateExists returns true if a report for the current period already exists in the uploaded or collected directories.
func (c Collector) duplicateExists() (bool, error) {
	for _, dir := range []string{c.collectedDir, c.uploadedDir} {
		cReport, err := report.GetForPeriod(dir, c.time, c.period)
		if err != nil {
			return false, fmt.Errorf("failed to check for duplicate report in %s for period: %v", dir, err)
		}
		if cReport.Name != "" {
			slog.Info("Duplicate report already exists", "file", cReport.Path)
			return true, nil
		}
	}

	return false, nil
}

// compile collects data from sources, and returns an Insights object.
func (c Collector) compile() ([]byte, error) {
	insights := Insights{
		InsightsVersion: constants.Version,
	}

	// Collect system information.
	info, err := c.sysInfo.Collect()
	if err != nil {
		return nil, fmt.Errorf("failed to collect system information: %v", err)
	}
	insights.SysInfo = info

	// Load source specific metrics.
	metrics, err := c.getSourceMetrics()
	if err != nil {
		return nil, fmt.Errorf("failed to load source metrics: %v", err)
	}
	insights.SourceMetrics = metrics

	return json.Marshal(insights)
}

// getSourceMetrics loads source specific metrics from a JSON file.
// If the sourceMetricsPath is empty, it returns nil.
//
// If the file does not exist, or cannot be read, it returns an error.
// If the file is not valid JSON, it returns an error.
func (c Collector) getSourceMetrics() (map[string]any, error) {
	slog.Debug("Loading source metrics", "path", c.sourceMetricsPath)

	if c.sourceMetricsPath == "" {
		slog.Info("No source metrics file provided")
		return nil, nil
	}

	data, err := os.ReadFile(c.sourceMetricsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read source metrics file: %v", err)
	}

	var metrics map[string]any
	if err := json.Unmarshal(data, &metrics); err != nil {
		return nil, fmt.Errorf("failed to unmarshal source metrics, might be invalid JSON: %v", err)
	}

	return metrics, nil
}

// write writes the insights report to disk, with the appropriate name.
func (c Collector) write(insights []byte) error {
	time, err := report.GetPeriodStart(c.period, c.time)
	if err != nil {
		return fmt.Errorf("failed to get report name: %v", err)
	}

	reportPath := filepath.Join(c.collectedDir, fmt.Sprintf("%v.json", time))
	if err := fileutils.AtomicWrite(reportPath, insights); err != nil {
		return fmt.Errorf("failed to write to disk: %v", err)
	}
	slog.Info("Insights report written", "file", reportPath)

	return nil
}
