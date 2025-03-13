// Package insights Golang bindings: collect and upload system metrics.
package insights

import (
	"github.com/ubuntu/ubuntu-insights/internal/collector"
	"github.com/ubuntu/ubuntu-insights/internal/uploader"
)

// WithCollectorFactory overrides the default collection writer.
func WithCollectorWriter(writer func (collector.Collector, []byte) error) collectOption {
	return func(opts *collectOptions) {
		opts.writer = writer
	}
}

// WithCollectorFactory overrides the default collector factory.
func WithCollectorFactory(factory collector.Factory) collectOption {
	return func(opts *collectOptions) {
		opts.factory = factory
	}
}

// WithUploaderFactory overrides the default uploader factory.
func WithUploaderFactory(factory uploader.Factory) uploadOption {
	return func(opts *uploadOptions) {
		opts.factory = factory
	}
}