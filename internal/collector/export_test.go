package collector

import (
	"log/slog"

	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo"
)

// WithMaxReports sets the maximum number of reports to keep.
func WithMaxReports(maxReports uint) Options {
	return func(o *options) {
		o.maxReports = maxReports
	}
}

// WithTimeProvider sets the time provider for the collector.
func WithTimeProvider(tp timeProvider) Options {
	return func(o *options) {
		o.timeProvider = tp
	}
}

// WithSysInfo sets the system information collector creation.
func WithSysInfo(si func(*slog.Logger, ...sysinfo.Options) SysInfo) Options {
	return func(o *options) {
		o.sysInfo = si
	}
}
