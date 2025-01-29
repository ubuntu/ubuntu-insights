package uploader

import "time"

type MockTimeProvider struct {
	CurrentTime int64
}

func (m MockTimeProvider) Now() time.Time {
	return time.Unix(m.CurrentTime, 0)
}

// WithCachePath sets the cache path for the uploader.
func WithCachePath(path string) Options {
	return func(o *options) {
		o.cachePath = path
	}
}

// WithBaseServerURL sets the base server URL for the uploader.
func WithBaseServerURL(url string) Options {
	return func(o *options) {
		o.baseServerURL = url
	}
}

// WithTimeProvider sets the time provider for the uploader.
func WithTimeProvider(tp timeProvider) Options {
	return func(o *options) {
		o.timeProvider = tp
	}
}
