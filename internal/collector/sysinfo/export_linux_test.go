package sysinfo

import "github.com/ubuntu/ubuntu-insights/internal/testutils"

// WithRoot overrides default root directory of the system.
func WithRoot(root string) Options {
	return func(o *options) {
		o.root = root
	}
}

// WithCpuInfo overrides default cpu info to return <info>
func WithCpuInfo(info string) Options {
	return func(o *options) {
		if info != "-" {
			o.cpuInfoCmd = testutils.MakeTestCmd(info)
		}
	}
}
