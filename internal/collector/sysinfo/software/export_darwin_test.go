package software

// WithOsInfo overrides default os info.
func WithOSInfo(cmd []string) Options {
	return func(o *options) {
		o.platform.osCmd = cmd
	}
}
