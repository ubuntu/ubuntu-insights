package hardware

func WithProductInfo(cmd []string) Options {
	return func(o *options) {
		o.platform.productCmd = cmd
	}
}

func WithCPUInfo(cmd []string) Options {
	return func(o *options) {
		o.platform.cpuCmd = cmd
	}
}

func WithGPUInfo(cmd []string) Options {
	return func(o *options) {
		o.platform.gpuCmd = cmd
	}
}

func WithMemoryInfo(cmd []string) Options {
	return func(o *options) {
		o.platform.memoryCmd = cmd
	}
}

func WithDiskInfo(cmd []string) Options {
	return func(o *options) {
		o.platform.diskCmd = cmd
	}
}

func WithPartitionInfo(cmd []string) Options {
	return func(o *options) {
		o.platform.partitionCmd = cmd
	}
}

func WithScreenInfo(cmd []string) Options {
	return func(o *options) {
		o.platform.screenCmd = cmd
	}
}
