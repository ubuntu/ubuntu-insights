package software

import (
	"log/slog"
)

// WithTimezoneProvider overrides the default time provider.
func WithTimezone(provider func() string) Options {
	return func(o *options) {
		o.timezone = provider
	}
}

// WithLogger overrides the default logger.
func WithLogger(logger slog.Handler) Options {
	return func(o *options) {
		o.log = slog.New(logger)
	}
}
