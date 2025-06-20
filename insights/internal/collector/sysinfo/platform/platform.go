// Package platform provides a way to collect information specific to a platform.
package platform

import (
	"log/slog"
)

// Collector handles dependencies for collecting platform information.
// Collector implements CollectorT[platform.Info].
type Collector struct {
	log      *slog.Logger
	platform platformOptions
}

// Options are the variadic options available to the Collector.
type Options func(*options)

type options struct {
	platform platformOptions
}

// New returns a new Collector.
func New(l *slog.Logger, args ...Options) Collector {
	opts := &options{}
	opts.platform = defaultPlatformOptions()
	for _, opt := range args {
		opt(opts)
	}

	return Collector{
		log:      l,
		platform: opts.platform,
	}
}

// Collect aggregates the data relevant to the platform.
func (s Collector) Collect() (info Info, err error) {
	s.log.Debug("collecting platform info")

	return s.collectPlatform()
}
