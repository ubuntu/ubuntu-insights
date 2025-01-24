// Package uploader implements the uploader component.
// The uploader component is responsible for uploading reports to the Ubuntu Insights server.
package uploader

import (
	"fmt"
	"log/slog"
	"math"
	"path/filepath"
	"time"

	"github.com/ubuntu/ubuntu-insights/internal/constants"
)

type timeProvider interface {
	NowUnix() int64
}

type realTimeProvider struct{}

func (realTimeProvider) NowUnix() int64 {
	return time.Now().Unix()
}

// Manager is an abstraction of the uploader component.
type Manager struct {
	source   string
	consentM consentManager
	minAge   int64
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
	GetConsentState(source string) (bool, error)
}

// New returns a new UploaderManager.
func New(cm consentManager, source string, minAge uint, dryRun bool, args ...Options) (Manager, error) {
	slog.Debug("Creating new uploader manager", "source", source, "minAge", minAge, "dryRun", dryRun)

	if source == "" {
		return Manager{}, fmt.Errorf("source cannot be an empty string")
	}

	if minAge > math.MaxInt64 {
		return Manager{}, fmt.Errorf("min age %d is too large, would overflow", minAge)
	}

	opts := options{
		baseServerURL: constants.DefaultServerURL,
		cachePath:     constants.GetDefaultCachePath(),
		timeProvider:  realTimeProvider{},
	}
	for _, opt := range args {
		opt(&opts)
	}

	return Manager{
		source:       source,
		consentM:     cm,
		minAge:       int64(minAge),
		dryRun:       dryRun,
		timeProvider: opts.timeProvider,

		baseServerURL: opts.baseServerURL,
		collectedDir:  filepath.Join(opts.cachePath, constants.LocalFolder),
		uploadedDir:   filepath.Join(opts.cachePath, constants.UploadedFolder),
	}, nil
}
