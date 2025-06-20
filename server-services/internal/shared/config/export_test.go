package config

import "log/slog"

// WithLogger is an option to set the logger for the Manager.
func WithLogger(l *slog.Logger) Options {
	return func(o *options) {
		o.Logger = l
	}
}

// GetReservedNames returns a map of reserved names that the configuration
// manager will filter from the allow list.
func GetReservedNames() map[string]struct{} {
	return reservedNames
}
