package uploader

func WithCachePath(path string) Options {
	return func(o *options) {
		o.cachePath = path
	}
}

func WithBaseServerURL(url string) Options {
	return func(o *options) {
		o.baseServerURL = url
	}
}

func WithTimeProvider(tp timeProvider) Options {
	return func(o *options) {
		o.timeProvider = tp
	}
}
