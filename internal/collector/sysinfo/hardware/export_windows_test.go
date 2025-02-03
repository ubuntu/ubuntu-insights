package hardware

func WithProductInfo(cmd []string) Options {
	return func(o *options) {
		o.productCmd = cmd
	}
}

func WithCPUInfo(cmd []string) Options {
	return func(o *options) {
		o.cpuCmd = cmd
	}
}

func WithGPUInfo(cmd []string) Options {
	return func(o *options) {
		o.gpuCmd = cmd
	}
}

func WithMemoryInfo(cmd []string) Options {
	return func(o *options) {
		o.memoryCmd = cmd
	}
}

func WithDiskInfo(cmd []string) Options {
	return func(o *options) {
		o.diskCmd = cmd
	}
}

func WithPartitionInfo(cmd []string) Options {
	return func(o *options) {
		o.partitionCmd = cmd
	}
}

func WithScreenInfo(cmd []string) Options {
	return func(o *options) {
		o.screenCmd = cmd
	}
}

func WithArch(arch string) Options {
	return func(o *options) {
		o.arch = arch
	}
}
