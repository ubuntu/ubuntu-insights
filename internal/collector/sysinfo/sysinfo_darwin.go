package sysinfo

import "log/slog"

type options struct {
	log *slog.Logger
}

func defaultOptions() *options {
	return &options{}
}

func (s Manager) collectHardware() (hwInfo, error) {
	return hwInfo{}, nil
}

func (s Manager) collectSoftware() (swInfo, error) {
	return swInfo{}, nil
}
