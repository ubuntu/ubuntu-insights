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
	source  string
	consent Consent
	minAge  time.Duration
	dryRun  bool

	baseServerURL      string
	collectedDir       string
	uploadedDir        string
	maxReports         uint
	timeProvider       timeProvider
	initialRetryPeriod time.Duration // initialRetryPeriod is the initial wait period between retries.
	maxRetryPeriod     time.Duration // maxRetryPeriod is the maximum wait period between retries.
	responseTimeout    time.Duration // responseTimeout is the timeout for the HTTP request.
}

type options struct {
	// Private members exported for tests.
	baseServerURL      string
	maxReports         uint
	timeProvider       timeProvider
	initialRetryPeriod time.Duration
	maxRetryPeriod     time.Duration
	responseTimeout    time.Duration
}

var defaultOptions = options{
	baseServerURL:      "https://metrics.ubuntu.com",
	maxReports:         constants.MaxReports,
	timeProvider:       realTimeProvider{},
	initialRetryPeriod: 30 * time.Second,
	maxRetryPeriod:     30 * time.Minute,
	responseTimeout:    10 * time.Second,
}

// Options represents an optional function to override Upload Manager default values.
type Options func(*options)

// Consent is an interface for getting the consent state for a given source.
type Consent interface {
	HasConsent(source string) (bool, error)
}

// New returns a new UploaderManager.
func New(cm Consent, cachePath, source string, minAge uint, dryRun, expRetry bool, args ...Options) (Uploader, error) {
	slog.Debug("Creating new uploader manager", "source", source, "minAge", minAge, "dryRun", dryRun)

	if source == "" {
		return Uploader{}, fmt.Errorf("source cannot be an empty string")
	}

	if minAge > (1<<63-1)/uint(time.Second) {
		return Uploader{}, fmt.Errorf("min age %d is too large, would overflow", minAge)
	}

	opts := defaultOptions
	for _, opt := range args {
		opt(&opts)
	}

	return Uploader{
		source:       source,
		consent:      cm,
		minAge:       time.Duration(minAge) * time.Second,
		dryRun:       dryRun,
		timeProvider: opts.timeProvider,

		baseServerURL:      opts.baseServerURL,
		collectedDir:       filepath.Join(cachePath, source, constants.LocalFolder),
		uploadedDir:        filepath.Join(cachePath, source, constants.UploadedFolder),
		maxReports:         opts.maxReports,
		initialRetryPeriod: opts.initialRetryPeriod,
		maxRetryPeriod:     opts.maxRetryPeriod,
		responseTimeout:    opts.responseTimeout,
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
