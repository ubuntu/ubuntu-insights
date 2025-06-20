package software

// WithTimezoneProvider overrides the default time provider.
func WithTimezone(provider func() string) Options {
	return func(o *options) {
		o.timezone = provider
	}
}
