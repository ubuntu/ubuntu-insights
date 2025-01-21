// Package uploader implements the uploader component.
// The uploader component is responsible for uploading reports to the Ubuntu Insights server.
package uploader

import (
	"errors"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/ubuntu/ubuntu-insights/internal/constants"
)

var (
	// ErrReportNotMature is returned when a report is not mature enough to be uploaded based on min age.
	ErrReportNotMature = errors.New("report is not mature enough to be uploaded")
	// ErrDuplicateReport is returned when a report has already been uploaded for this period.
	ErrDuplicateReport = errors.New("report has already been uploaded for this period")
	// ErrEmptySource is returned when the passed source is incorrectly an empty string.
	ErrEmptySource = errors.New("source cannot be an empty string")
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
	source         string
	consentManager consentManager
	minAge         uint
	dryRun         bool

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
		return Manager{}, ErrEmptySource
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
		source:         source,
		consentManager: cm,
		minAge:         minAge,
		dryRun:         dryRun,
		timeProvider:   opts.timeProvider,

		baseServerURL: opts.baseServerURL,
		collectedDir:  filepath.Join(opts.cachePath, constants.LocalFolder),
		uploadedDir:   filepath.Join(opts.cachePath, constants.UploadedFolder),
	}, nil
}
