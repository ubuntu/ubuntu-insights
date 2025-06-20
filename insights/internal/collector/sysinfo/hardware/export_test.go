package hardware

// WithArch overrides the default architecture.
func WithArch(arch string) Options {
	return func(o *options) {
		o.arch = arch
	}
}
