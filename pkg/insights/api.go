// Package insights Golang bindings: collect and upload system metrics.
package insights

import (
	"github.com/ubuntu/ubuntu-insights/internal/collector"
	"github.com/ubuntu/ubuntu-insights/internal/consent"
	"github.com/ubuntu/ubuntu-insights/internal/constants"
	"github.com/ubuntu/ubuntu-insights/internal/uploader"
)

// ConsentState is one of ConsentUnknown, ConsentFalse, ConsentTrue.
type ConsentState int

const (
	ConsentUnknown = iota - 1
	ConsentFalse
	ConsentTrue
)

type Config struct {
	Source      string
	ConsentDir  string
	InsightsDir string
	Verbose     bool
}

type CollectFlags struct {
	Period uint
	Force  bool
	DryRun bool
}

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
	cConf := collector.Config{
		Source:        c.Source,
		Period:        flags.Period,
		Force:         flags.Force,
		DryRun:        flags.DryRun,
		SourceMetrics: metricsPath,
	}

	return cConf.Run(c.ConsentDir, c.InsightsDir, func(c collector.Collector, b []byte) error {
		return c.Write(b)
	})
}

// Upload uploads reports for Config.Source.
// if Config.Source is "", all reports are uploaded.
// returns an error if uploading fails.
func (c Config) Upload(flags UploadFlags) error {
	c.setDefaults()
	flags.setDefaults()
	cm := c.newConsentManager()

	u, err := uploader.New(cm, c.InsightsDir, c.Source, flags.MinAge, flags.DryRun)
	if err != nil {
		return err
	}

	return u.Upload(flags.Force)
}

// GetConsentState gets the state for Config.Source.
// if Config.Source is "", the global source is retrieved.
// returns ConsentUnknown if it could not be retrieved.
// returns ConsentTrue or ConsentFalse otherwise.
func (c Config) GetConsentState() ConsentState {
	c.setDefaults()
	cm := c.newConsentManager()

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
func (c Config) SetConsentState(consent bool) error {
	c.setDefaults()
	cm := c.newConsentManager()
	return cm.SetState(c.Source, consent)
}

func (c Config) newConsentManager() *consent.Manager {
	return consent.New(c.ConsentDir)
}

func (c *Config) setDefaults() {
	if c.ConsentDir == "" {
		c.ConsentDir = constants.DefaultConfigPath
	}
	if c.InsightsDir == "" {
		c.InsightsDir = constants.DefaultCachePath
	}
}

func (f *CollectFlags) setDefaults() {
	if f.Period == 0 {
		f.Period = constants.DefaultPeriod
	}
}

func (f *UploadFlags) setDefaults() {
	if f.MinAge == 0 {
		f.MinAge = constants.DefaultMinAge
	}
}
