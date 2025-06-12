// Package uploader implements the uploader component.
// The uploader component is responsible for uploading reports to the Ubuntu Insights server.
package uploader

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/ubuntu/ubuntu-insights/internal/constants"
)

type timeProvider interface {
	Now() time.Time
}

type realTimeProvider struct{}

func (realTimeProvider) Now() time.Time {
	return time.Now()
}

// Uploader is an abstraction of the uploader component.
type Uploader struct {
	consent  Consent
	minAge   time.Duration
	dryRun   bool
	cacheDir string

	baseServerURL string
	maxReports    uint
	timeProvider  timeProvider

	baseRetryPeriod time.Duration // initialRetryPeriod is the initial wait period between retries.
	maxRetryPeriod  time.Duration // maxRetryPeriod is the maximum wait period between retries.
	maxAttempts     int           // maxRetries is the maximum number of retry attempts before giving up.

	responseTimeout time.Duration // responseTimeout is the timeout for the HTTP request.

	log *slog.Logger
}

type options struct {
	// Private members exported for tests.
	baseServerURL string
	maxReports    uint
	timeProvider  timeProvider

	baseRetryPeriod time.Duration
	maxRetryPeriod  time.Duration
	maxAttempts     int

	responseTimeout time.Duration
}

var defaultOptions = options{
	baseServerURL: "https://metrics.ubuntu.com",
	maxReports:    constants.MaxReports,
	timeProvider:  realTimeProvider{},

	// Approximately 5 hours before giving up.
	baseRetryPeriod: 30 * time.Second,
	maxRetryPeriod:  30 * time.Minute,
	maxAttempts:     15,

	responseTimeout: 10 * time.Second,
}

// Config represents the uploader specific data needed to upload.
type Config struct {
	Sources []string
	MinAge  uint `mapstructure:"minAge"`
	Force   bool
	DryRun  bool `mapstructure:"dryRun"`
	Retry   bool `mapstructure:"retry"`
}

// Sanitize sets defaults and checks that the Config is properly configured.
func (c *Config) Sanitize(l *slog.Logger, cacheDir string) error {
	if len(c.Sources) == 0 {
		l.Info("No sources provided, uploading all sources")
		var err error
		c.Sources, err = GetAllSources(cacheDir)
		if err != nil {
			return fmt.Errorf("failed to get all sources: %v", err)
		}
	}

	return nil
}

// Options represents an optional function to override Upload Manager default values.
type Options func(*options)

// Consent is an interface for getting the consent state for a given source.
type Consent interface {
	HasConsent(source string) (bool, error)
}

// New returns a new UploaderManager.
func New(l *slog.Logger, cm Consent, cachePath string, minAge uint, dryRun bool, args ...Options) (Uploader, error) {
	l.Debug("Creating new uploader manager", "minAge", minAge, "dryRun", dryRun)

	if minAge > (1<<63-1)/uint(time.Second) {
		return Uploader{}, fmt.Errorf("min age %d is too large, would overflow", minAge)
	}

	opts := defaultOptions
	for _, opt := range args {
		opt(&opts)
	}

	return Uploader{
		consent:      cm,
		minAge:       time.Duration(minAge) * time.Second,
		dryRun:       dryRun,
		timeProvider: opts.timeProvider,
		cacheDir:     cachePath,

		baseServerURL: opts.baseServerURL,
		maxReports:    opts.maxReports,

		baseRetryPeriod: opts.baseRetryPeriod,
		maxRetryPeriod:  opts.maxRetryPeriod,
		maxAttempts:     opts.maxAttempts,

		responseTimeout: opts.responseTimeout,

		log: l,
	}, nil
}

// GetAllSources returns a list of the source directories in the cache directory.
func GetAllSources(dir string) ([]string, error) {
	sources := make([]string, 0)
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip files.
		if !d.IsDir() {
			return nil
		}

		// Skip the root directory.
		if path == dir {
			return nil
		}

		// Skip subdirectories.
		if d.IsDir() && filepath.Dir(path) != dir {
			return filepath.SkipDir
		}

		sources = append(sources, filepath.Base(path))
		return nil
	})
	return sources, err
}
