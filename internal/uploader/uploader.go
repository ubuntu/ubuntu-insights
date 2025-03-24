// Package uploader implements the uploader component.
// The uploader component is responsible for uploading reports to the Ubuntu Insights server.
package uploader

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ubuntu/ubuntu-insights/internal/consent"
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

// Config represents the uploader specific data needed to upload.
type Config struct {
	Sources []string
	MinAge  uint `mapstructure:"minAge"`
	Force   bool
	DryRun  bool `mapstructure:"dryRun"`
	Retry   bool `mapstructure:"retry"`
}

// Factory represents a function that creates a new Uploader.
type Factory = func(cm Consent, cachePath, source string, minAge uint, dryRun bool, args ...Options) (Uploader, error)

// Run creates an uploader then uploads using it based off the given config and arguments.
func (c Config) Run(consentDir, cacheDir string, factory Factory) error {
	if cacheDir == "" {
		cacheDir = constants.DefaultCachePath
	}

	if len(c.Sources) == 0 {
		slog.Info("No sources provided, uploading all sources")
		var err error
		c.Sources, err = GetAllSources(cacheDir)
		if err != nil {
			return fmt.Errorf("failed to get all sources: %v", err)
		}
	}

	cm := consent.New(consentDir)

	uploaders := make(map[string]Uploader)
	for _, source := range c.Sources {
		u, err := factory(cm, cacheDir, source, c.MinAge, c.DryRun)
		if err != nil {
			return fmt.Errorf("failed to create uploader for source %s: %v", source, err)
		}
		uploaders[source] = u
	}

	var uploadError error
	mu := &sync.Mutex{}
	var wg sync.WaitGroup
	for s, u := range uploaders {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var err error
			if c.Retry {
				err = u.BackoffUpload(c.Force)
			} else {
				err = u.Upload(c.Force)
			}
			if errors.Is(err, consent.ErrConsentFileNotFound) {
				slog.Warn("Consent file not found, skipping upload", "source", s)
				return
			}

			if err != nil {
				errMsg := fmt.Errorf("failed to upload reports for source %s: %v", s, err)
				mu.Lock()
				defer mu.Unlock()
				uploadError = errors.Join(uploadError, errMsg)
			}
		}()
	}
	wg.Wait()
	return uploadError
}

// Options represents an optional function to override Upload Manager default values.
type Options func(*options)

// Consent is an interface for getting the consent state for a given source.
type Consent interface {
	HasConsent(source string) (bool, error)
}

// New returns a new UploaderManager.
func New(cm Consent, cachePath, source string, minAge uint, dryRun bool, args ...Options) (Uploader, error) {
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
