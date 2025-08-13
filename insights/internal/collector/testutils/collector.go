//go:build integrationtests

package collectortestutils

import (
	"log/slog"
	_ "unsafe" // For go:linkname

	"github.com/ubuntu/ubuntu-insights/common/testsdetection"
	"github.com/ubuntu/ubuntu-insights/insights/internal/collector"
	"github.com/ubuntu/ubuntu-insights/insights/internal/collector/sysinfo"
)

func init() {
	// No import outside of testing environment.
	testsdetection.MustBeTesting()
}

type timeFunc func() int64

//go:linkname defaultOptions github.com/ubuntu/ubuntu-insights/insights/internal/collector.defaultOptions
var defaultOptions struct {
	maxReports uint32
	time       timeFunc
	sysInfo    func(*slog.Logger, ...sysinfo.Options) collector.SysInfo
}

// SetMaxReports overrides the max reports count the uploader is using.
func SetMaxReports(r uint32) {
	defaultOptions.maxReports = r
}

// SetTime overrides the collector time.
func SetTime(t int64) {
	defaultOptions.time = func() int64 { return t }
}

// SetSysInfo overrides the sysinfo the collector is using.
func SetSysInfo(si collector.SysInfo) {
	defaultOptions.sysInfo = func(*slog.Logger, ...sysinfo.Options) collector.SysInfo {
		return si
	}
}
