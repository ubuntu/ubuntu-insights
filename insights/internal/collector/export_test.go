package collector

import (
	"log/slog"

	"github.com/ubuntu/ubuntu-insights/insights/internal/collector/sysinfo"
)

// WithMaxReports sets the maximum number of reports to keep.
func WithMaxReports(maxReports uint32) Options {
	return func(o *options) {
		o.maxReports = maxReports
	}
}

func WithTime(time int64) Options {
	return func(o *options) {
		o.time = time
	}
}

// WithSysInfo sets the system information collector creation.
func WithSysInfo(si func(*slog.Logger, ...sysinfo.Options) SysInfo) Options {
	return func(o *options) {
		o.sysInfo = si
	}
}
