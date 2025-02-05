package hardware

import (
	"log/slog"
)

// WithLogger overrides the default logger.
func WithLogger(logger slog.Handler) Options {
	return func(o *options) {
		o.log = slog.New(logger)
	}
}

// WithArch overrides the default architecture.
func WithArch(arch string) Options {
	return func(o *options) {
		o.arch = arch
	}
}
