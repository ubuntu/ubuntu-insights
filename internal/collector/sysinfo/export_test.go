package sysinfo

import (
	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/hardware"
	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/software"
)

func WithHardwareCollector(hw hardware.Collector) Options {
	return func(o *options) {
		o.hw = hw
	}
}

func WithSoftwareCollector(sw software.Collector) Options {
	return func(o *options) {
		o.sw = sw
	}
}
