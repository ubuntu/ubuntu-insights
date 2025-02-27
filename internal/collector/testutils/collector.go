//go:build integrationtests

package collectortestutils

import (
	"time"
	_ "unsafe" // For go:linkname

	"github.com/ubuntu/ubuntu-insights/internal/collector"
	"github.com/ubuntu/ubuntu-insights/internal/testsdetection"
)

func init() {
	// No import outside of testing environment.
	testsdetection.MustBeTesting()
}

type timeProvider interface {
	Now() time.Time
}

//go:linkname defaultOptions github.com/ubuntu/ubuntu-insights/internal/collector.defaultOptions
var defaultOptions struct {
	sourceMetricsPath string
	maxReports        uint
	timeProvider      timeProvider
	sysInfo           collector.SysInfo
}

// SetMaxReports overrides the max reports count the uploader is using.
func SetMaxReports(r uint) {
	defaultOptions.maxReports = r
}

// SetTimeProvider overrides the time provider the uploader is using.
func SetTimeProvider(tp timeProvider) {
	defaultOptions.timeProvider = tp
}

// SetSysInfo overrides the sysinfo the collector is using.
func SetSysInfo(si collector.SysInfo) {
	defaultOptions.sysInfo = si
}
