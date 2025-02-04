package software

// WithRoot overrides default root directory of the system.
func WithRoot(root string) Options {
	return func(o *options) {
		o.root = root
	}
}

// WithOsInfo overrides default os info.
func WithOSInfo(cmd []string) Options {
	return func(o *options) {
		o.osCmd = cmd
	}
}

// WithLang overrides default language provider.
func WithLang(provider func() (string, bool)) Options {
	return func(o *options) {
		o.langFunc = provider
	}
}
