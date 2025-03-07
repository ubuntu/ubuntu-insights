package sysinfo

import (
	"log/slog"

	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/hardware"
	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/platform"
	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/software"
)

// WithHardwareCollector overrides the default hardware collector.
func WithHardwareCollector(hw PCollectorT[hardware.Info]) Options {
	return func(o *options) {
		o.hw = hw
	}
}

// WithSoftwareCollector overrides the default software collector.
func WithSoftwareCollector(sw PCollectorT[software.Info]) Options {
	return func(o *options) {
		o.sw = sw
	}
}

// WithPlatformCollector overrides the default platform collector.
func WithPlatformCollector(pl CollectorT[platform.Info]) Options {
	return func(o *options) {
		o.pl = pl
	}
}

// WithLogger overrides the default logger.
func WithLogger(logger slog.Handler) Options {
	return func(o *options) {
		o.log = slog.New(logger)
	}
}
