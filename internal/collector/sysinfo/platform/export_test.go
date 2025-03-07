package platform

import "log/slog"

// WithLogger overrides the default logger.
func WithLogger(logger slog.Handler) Options {
	return func(o *options) {
		o.log = slog.New(logger)
	}
}
