package collector

// WithMaxReports sets the maximum number of reports to keep.
func WithMaxReports(maxReports int) Options {
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

// WithSysInfo sets the system information collector.
func WithSysInfo(si SysInfo) Options {
	return func(o *options) {
		o.sysInfo = si
	}
}
