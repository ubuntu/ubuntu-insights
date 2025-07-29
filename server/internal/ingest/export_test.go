package ingest

import "time"

var (
	ErrServiceClosed = errServiceClosed
)

func WithMaxDegradedDuration(d time.Duration) Option {
	return func(o *options) {
		o.maxDegradedDuration = d
	}
}
