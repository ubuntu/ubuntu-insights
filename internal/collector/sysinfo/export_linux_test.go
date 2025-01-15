package sysinfo

// WithRoot overrides default root directory of the system.
func WithRoot(root string) Options {
	return func(o *options) {
		o.root = root
	}
}

// WithCpuInfo overrides default cpu info to return <info>
func WithCpuInfo(cmd []string) Options {
	return func(o *options) {
		o.cpuInfoCmd = cmd
	}
}
