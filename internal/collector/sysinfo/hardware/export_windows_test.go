package hardware

// WithProductInfo overrides default product info.
func WithProductInfo(cmd []string) Options {
	return func(o *options) {
		o.platform.productCmd = cmd
	}
}

// WithCpuInfo overrides default CPU info.
func WithCPUInfo(cmd []string) Options {
	return func(o *options) {
		o.platform.cpuCmd = cmd
	}
}

// WithGpuInfo overrides default GPU info.
func WithGPUInfo(cmd []string) Options {
	return func(o *options) {
		o.platform.gpuCmd = cmd
	}
}

// WithMemoryInfo overrides default memory info.
func WithMemoryInfo(cmd []string) Options {
	return func(o *options) {
		o.platform.memoryCmd = cmd
	}
}

// WithDiskInfo overrides default disk info.
func WithDiskInfo(cmd []string) Options {
	return func(o *options) {
		o.platform.diskCmd = cmd
	}
}

// WithPartitionInfo overrides default partition info.
func WithPartitionInfo(cmd []string) Options {
	return func(o *options) {
		o.platform.partitionCmd = cmd
	}
}

// WithScreenResInfo overrides default screen resolution info.
func WithScreenResInfo(cmd []string) Options {
	return func(o *options) {
		o.platform.screenResCmd = cmd
	}
}

// WithScreenResInfo overrides default screen size info.
func WithScreenSizeInfo(cmd []string) Options {
	return func(o *options) {
		o.platform.displaySizeCmd = cmd
	}
}
