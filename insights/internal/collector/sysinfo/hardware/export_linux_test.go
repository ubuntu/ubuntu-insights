package hardware

type Screen = screen
type CWaylandDisplay = cWaylandDisplay

// WithRoot overrides default root directory of the system.
func WithRoot(root string) Options {
	return func(o *options) {
		o.platform.root = root
	}
}

// WithCpuInfo overrides default cpu info.
func WithCPUInfo(cmd []string) Options {
	return func(o *options) {
		o.platform.cpuInfoCmd = cmd
	}
}

// WithBlkInfo overrides default blk info.
func WithBlkInfo(cmd []string) Options {
	return func(o *options) {
		o.platform.lsblkCmd = cmd
	}
}

// WithScreenInfo overrides default screen info.
func WithScreenInfo(cmd []string) Options {
	return func(o *options) {
		o.platform.screenCmd = cmd
	}
}

// WithWaylandProvider overrides default wayland provider.
func WithWaylandProvider(wp waylandProvider) Options {
	return func(o *options) {
		o.platform.wayland = wp
	}
}
