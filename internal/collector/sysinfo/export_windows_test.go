package sysinfo

func WithGPUInfo(cmd []string) Options {
	return func(o *options) {
		o.gpuCmd = cmd
	}
}
