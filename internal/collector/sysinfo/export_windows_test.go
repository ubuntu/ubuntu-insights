package sysinfo

func WithProductInfo(cmd []string) Options {
	return func(o *options) {
		o.productCmd = cmd
	}
}

func WithGPUInfo(cmd []string) Options {
	return func(o *options) {
		o.gpuCmd = cmd
	}
}
