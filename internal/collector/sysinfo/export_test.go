package sysinfo

import (
	"log/slog"

	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/hardware"
	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/software"
)

// WithHardwareCollector overrides the default hardware collector.
func WithHardwareCollector(hw CollectorT[hardware.Info]) Options {
	return func(o *options) {
		o.hw = hw
	}
}

// WithSoftwareCollector overrides the default software collector.
func WithSoftwareCollector(sw CollectorT[software.Info]) Options {
	return func(o *options) {
		o.sw = sw
	}
}

// WithLogger overrides the default logger.
func WithLogger(logger slog.Handler) Options {
	return func(o *options) {
		o.log = slog.New(logger)
	}
}
