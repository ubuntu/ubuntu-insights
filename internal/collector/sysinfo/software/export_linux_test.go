package software

func WithOSInfo(cmd []string) Options {
	return func(o *options) {
		o.osCmd = cmd
	}
}
