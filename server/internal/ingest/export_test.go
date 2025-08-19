package ingest

import "time"

func WithMaxDegradedDuration(d time.Duration) Option {
	return func(o *options) {
		o.maxDegradedDuration = d
	}
}
