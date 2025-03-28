// Package insights Golang bindings: collect and upload system metrics.
package insights

import (
	"log/slog"

	"github.com/ubuntu/ubuntu-insights/internal/collector"
	"github.com/ubuntu/ubuntu-insights/internal/consent"
	"github.com/ubuntu/ubuntu-insights/internal/constants"
	"github.com/ubuntu/ubuntu-insights/internal/uploader"
)

// ConsentState is one of ConsentUnknown, ConsentFalse, ConsentTrue.
type ConsentState int

const (
	// ConsentUnknown is used when GetState returns an error.
	ConsentUnknown = iota - 1

	// ConsentFalse is used when GetState returns false.
	ConsentFalse

	// ConsentTrue is used when GetState returns an true.
	ConsentTrue
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
	Period uint
	Force  bool
	DryRun bool
}

// UploadFlags represents optional parameters for Upload.
type UploadFlags struct {
	MinAge uint
	Force  bool
	DryRun bool
}

// Collect creates a report for Config.Source.
// metricsPath is a filepath to a JSON file containing extra metrics.
// If Config.Source is "",  the source is the platform and metricsPath is ignored.
// returns an error if metricsPath is "" and not ignored.
// returns an error if collection fails.
func (c Config) Collect(metricsPath string, flags CollectFlags) error {
	setVerbosity(c.Verbose)

	cConf := collector.Config{
		Source:        c.Source,
		Period:        flags.Period,
		Force:         flags.Force,
		DryRun:        flags.DryRun,
		SourceMetrics: metricsPath,
	}

	return cConf.Run(c.ConsentDir, c.InsightsDir, func(c collector.Collector, b collector.Insights) error {
		return c.Write(b)
	}, collector.New)
}

// Upload uploads reports for Config.Source.
// if Config.Source is "", all reports are uploaded.
// returns an error if uploading fails.
func (c Config) Upload(flags UploadFlags) error {
	setVerbosity(c.Verbose)

	uConf := uploader.Config{
		Sources: []string{c.Source},
		MinAge:  flags.MinAge,
		Force:   flags.Force,
		DryRun:  flags.DryRun,
		Retry:   false,
	}

	return uConf.Run(c.ConsentDir, c.InsightsDir, uploader.New)
}

// GetConsentState gets the state for Config.Source.
// if Config.Source is "", the global source is retrieved.
// returns ConsentUnknown if it could not be retrieved.
// returns ConsentTrue or ConsentFalse otherwise.
func (c Config) GetConsentState() ConsentState {
	setVerbosity(c.Verbose)

	cm := consent.New(c.ConsentDir)
	s, err := cm.GetState(c.Source)
	if err != nil {
		return ConsentUnknown
	}

	if s {
		return ConsentTrue
	}
	return ConsentFalse
}

// SetConsentState sets the state for Config.Source to consent.
// if Config.Source is "", the global source is effected.
// returns an error if the state could not be set.
func (c Config) SetConsentState(consentState bool) error {
	setVerbosity(c.Verbose)

	cm := consent.New(c.ConsentDir)
	return cm.SetState(c.Source, consentState)
}

// setVerbosity sets the logging verbosity.
func setVerbosity(verbose bool) {
	if verbose {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	} else {
		slog.SetLogLoggerLevel(constants.DefaultLogLevel)
	}
}
