//go:build integrationtests

package uploadertestutils

import (
	"time"
	_ "unsafe" // For go:linkname

	"github.com/ubuntu/ubuntu-insights/common/testsdetection"
)

func init() {
	// No import outside of testing environment.
	testsdetection.MustBeTesting()
}

type timeProvider interface {
	Now() time.Time
}

//go:linkname defaultOptions github.com/ubuntu/ubuntu-insights/insights/internal/uploader.defaultOptions
var defaultOptions struct {
	baseServerURL string
	maxReports    uint32
	timeProvider  timeProvider

	baseRetryPeriod time.Duration
	maxRetryPeriod  time.Duration
	maxAttempts     uint32

	responseTimeout time.Duration

	maxConcurrentUploads uint32
	maxConcurrentSources uint32
}

// SetServerURL overrides the server url the uploader is using.
func SetServerURL(url string) {
	defaultOptions.baseServerURL = url
}

// SetMaxReports overrides the max reports count the uploader is using.
func SetMaxReports(r uint32) {
	defaultOptions.maxReports = r
}

// SetTimeProvider overrides the time provider the uploader is using.
func SetTimeProvider(tp timeProvider) {
	defaultOptions.timeProvider = tp
}

// SetBaseRetryPeriod overrides the initial retry period the uploader is using.
func SetBaseRetryPeriod(d time.Duration) {
	defaultOptions.baseRetryPeriod = d
}

// SetMaxRetryPeriod overrides the report timeout the uploader is using.
func SetMaxRetryPeriod(d time.Duration) {
	defaultOptions.maxRetryPeriod = d
}

// SetMaxAttempts overrides the max attempts the uploader is using.
func SetMaxAttempts(a uint32) {
	defaultOptions.maxAttempts = a
}

// SetResponseTimeout overrides the response timeout the uploader is using.
func SetResponseTimeout(d time.Duration) {
	defaultOptions.responseTimeout = d
}
