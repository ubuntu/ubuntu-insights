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

// AllowSet returns the internal set of allowed names.
func (cm *Manager) AllowSet() map[string]struct{} {
	cm.lock.RLock()
	defer cm.lock.RUnlock()
	return cm.allowSet
}
