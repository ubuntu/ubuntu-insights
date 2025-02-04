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
	source   string
	consentM consentManager
	minAge   time.Duration
	dryRun   bool

	baseServerURL string
	collectedDir  string
	uploadedDir   string
	timeProvider  timeProvider
}

type options struct {
	// Private members exported for tests.
	baseServerURL string
	cachePath     string
	timeProvider  timeProvider
}

// Options represents an optional function to override Upload Manager default values.
type Options func(*options)

type consentManager interface {
	HasConsent(source string) (bool, error)
}

// New returns a new UploaderManager.
func New(cm consentManager, source string, minAge uint, dryRun bool, args ...Options) (Uploader, error) {
	slog.Debug("Creating new uploader manager", "source", source, "minAge", minAge, "dryRun", dryRun)

	if source == "" {
		return Uploader{}, fmt.Errorf("source cannot be an empty string")
	}

	if minAge > (1<<63-1)/uint(time.Second) {
		return Uploader{}, fmt.Errorf("min age %d is too large, would overflow", minAge)
	}

	opts := options{
		baseServerURL: constants.DefaultServerURL,
		cachePath:     constants.GetDefaultCachePath(),
		timeProvider:  realTimeProvider{},
	}
	for _, opt := range args {
		opt(&opts)
	}

	return Uploader{
		source:       source,
		consentM:     cm,
		minAge:       time.Duration(minAge) * time.Second,
		dryRun:       dryRun,
		timeProvider: opts.timeProvider,

		baseServerURL: opts.baseServerURL,
		collectedDir:  filepath.Join(opts.cachePath, source, constants.LocalFolder),
		uploadedDir:   filepath.Join(opts.cachePath, source, constants.UploadedFolder),
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
