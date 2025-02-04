package software

import "log/slog"

type options struct {
	log *slog.Logger
}

func defaultOptions() *options {
	return &options{}
}

func (s Collector) collectOS() (info os, err error) {
	return info, nil
}
