package software

// WithRoot overrides default root directory of the system.
func WithRoot(root string) Options {
	return func(o *options) {
		o.platform.root = root
	}
}

// WithLang overrides default language provider.
func WithLang(provider func() (string, bool)) Options {
	return func(o *options) {
		o.platform.langFunc = provider
	}
}

// WithSnapEnv overrides the SNAP directory lookup.
func WithSnapEnv(dir string) Options {
	return func(o *options) {
		o.platform.snapEnvFunc = func() string { return dir }
	}
}
