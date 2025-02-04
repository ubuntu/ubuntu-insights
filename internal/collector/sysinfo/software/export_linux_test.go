package software

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
