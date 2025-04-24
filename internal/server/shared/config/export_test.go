package config

import "log/slog"

// WithLogger is an option to set the logger for the Manager.
func WithLogger(l *slog.Logger) Options {
	return func(o *options) {
		o.Logger = l
	}
}
