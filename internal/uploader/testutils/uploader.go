//go:build integrationtests

package uploadertestutils

import (
	"time"
	_ "unsafe" // For go:linkname

	"github.com/ubuntu/ubuntu-insights/internal/testsdetection"
)

func init() {
	// No import outside of testing environment.
	testsdetection.MustBeTesting()
}

type timeProvider interface {
	Now() time.Time
}

//go:linkname defaultOptions github.com/ubuntu/ubuntu-insights/internal/uploader.defaultOptions
var defaultOptions struct {
	baseServerURL      string
	maxReports         uint
	timeProvider       timeProvider
	initialRetryPeriod time.Duration
	maxRetryPeriod     time.Duration
	responseTimeout    time.Duration
}

// SetServerURL overrides the server url the uploader is using.
func SetServerURL(url string) {
	defaultOptions.baseServerURL = url
}

// SetMaxReports overrides the max reports count the uploader is using.
func SetMaxReports(r uint) {
	defaultOptions.maxReports = r
}

// SetTimeProvider overrides the time provider the uploader is using.
func SetTimeProvider(tp timeProvider) {
	defaultOptions.timeProvider = tp
}

// SetInitialRetryPeriod overrides the initial retry period the uploader is using.
func SetInitialRetryPeriod(d time.Duration) {
	defaultOptions.initialRetryPeriod = d
}

// SetMaxRetryPeriod overrides the report timeout the uploader is using.
func SetMaxRetryPeriod(d time.Duration) {
	defaultOptions.maxRetryPeriod = d
}

// SetResponseTimeout overrides the response timeout the uploader is using.
func SetResponseTimeout(d time.Duration) {
	defaultOptions.responseTimeout = d
}
