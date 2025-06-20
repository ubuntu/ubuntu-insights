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

// WithSystemdAnalyzeCmd sets the systemd-analyze command for the platform collector.
func WithSystemdAnalyzeCmd(cmd []string) Options {
	return func(o *options) {
		o.platform.systemdAnalyzeCmd = cmd
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

// WithGetenv sets the getenv function for the linux platform collector using a map.
func WithGetenv(env map[string]string) Options {
	return func(o *options) {
		o.platform.getenv = func(key string) string {
			return env[key]
		}
	}
}
