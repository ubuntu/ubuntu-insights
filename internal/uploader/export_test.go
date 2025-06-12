package uploader

import "time"

type MockTimeProvider struct {
	CurrentTime int64
}

func (m MockTimeProvider) Now() time.Time {
	return time.Unix(m.CurrentTime, 0)
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

// WithMaxRetryPeriod sets the report timeout for the uploader, for exponential backoff retries.
func WithMaxRetryPeriod(d time.Duration) Options {
	return func(o *options) {
		o.maxRetryPeriod = d
	}
}

// WithInitialRetryPeriod sets the initial retry period for the uploader, for exponential backoff retries.
func WithInitialRetryPeriod(d time.Duration) Options {
	return func(o *options) {
		o.initialRetryPeriod = d
	}
}

// WithMaxAttempts sets the maximum number of attempts for the uploader for exponential backoff retries.
func WithMaxAttempts(n int) Options {
	return func(o *options) {
		o.maxAttempts = n
	}
}

// WithResponseTimeout sets the response timeout for the uploader when waiting for a response from the server.
func WithResponseTimeout(d time.Duration) Options {
	return func(o *options) {
		o.responseTimeout = d
	}
}
