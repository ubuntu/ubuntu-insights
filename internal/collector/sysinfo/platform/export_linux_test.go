package platform

// WithRoot sets the root directory for the platform collector.
func WithRoot(root string) Options {
	return func(o *options) {
		o.platform.root = root
	}
}

// WithDetectVirtCmd sets the detect virtualization command for the platform collector.
func WithDetectVirtCmd(cmd []string) Options {
	return func(o *options) {
		o.platform.detectVirtCmd = cmd
	}
}

// WithWSLStatusCmd sets the WSL status command for the platform collector.
func WithWSLStatusCmd(cmd []string) Options {
	return func(o *options) {
		o.platform.wslStatusCmd = cmd
	}
}

// WithWSLVersionCmd sets the WSL version command for the platform collector.
func WithWSLVersionCmd(cmd []string) Options {
	return func(o *options) {
		o.platform.wslVersionCmd = cmd
	}
}

// WithProStatusCmd sets the pro status command for the platform collector.
func WithProStatusCmd(cmd []string) Options {
	return func(o *options) {
		o.platform.proStatusCmd = cmd
	}
}
