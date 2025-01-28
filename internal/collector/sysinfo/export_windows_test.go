package sysinfo

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
