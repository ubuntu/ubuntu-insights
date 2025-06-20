//go:build integrationtests

package collectortestutils

import (
	"log/slog"
	"time"
	_ "unsafe" // For go:linkname

	"github.com/ubuntu/ubuntu-insights/insights/internal/collector"
	"github.com/ubuntu/ubuntu-insights/insights/internal/collector/sysinfo"
	"github.com/ubuntu/ubuntu-insights/shared/testsdetection"
)

func init() {
	// No import outside of testing environment.
	testsdetection.MustBeTesting()
}

type timeProvider interface {
	Now() time.Time
}

//go:linkname defaultOptions github.com/ubuntu/ubuntu-insights/insights/internal/collector.defaultOptions
var defaultOptions struct {
	maxReports   uint
	timeProvider timeProvider
	sysInfo      func(*slog.Logger, ...sysinfo.Options) collector.SysInfo
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
	defaultOptions.sysInfo = func(*slog.Logger, ...sysinfo.Options) collector.SysInfo {
		return si
	}
}
