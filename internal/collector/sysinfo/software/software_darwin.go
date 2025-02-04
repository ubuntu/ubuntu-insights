package software

import (
	"log/slog"
)

type options struct {
	timezone func() string
	log      *slog.Logger
}

func defaultOptions() *options {
	return &options{}
}

func (s Collector) collectOS() (info osInfo, err error) {
	return info, nil
}

func (s Collector) collectLang() (string, error) {
	return "", nil
}
