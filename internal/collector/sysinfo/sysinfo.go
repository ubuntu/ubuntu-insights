// Package sysinfo allows collecting "common" system information for all insight reports.
package sysinfo

import (
	"fmt"
	"log/slog"

	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/hardware"
	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/platform"
	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/software"
)

// PCollectorT describes a type that collects some information T while using platform.Info.
type PCollectorT[T any] interface {
	Collect(platform.Info) (T, error)
}

// CollectorT describes a type that collects some information T.
type CollectorT[T any] interface {
	Collect() (T, error)
}

// Options is the variadic options available to the Collector.
type Options func(*options)

type options struct {
	hw PCollectorT[hardware.Info]
	sw PCollectorT[software.Info]
	pl CollectorT[platform.Info]

	log *slog.Logger
}

// Collector handles dependencies for collecting software & hardware information.
// Collector implements CollectorT[sysinfo.Info].
type Collector struct {
	hw PCollectorT[hardware.Info]
	sw PCollectorT[software.Info]
	pl CollectorT[platform.Info]

	log *slog.Logger
}

// Info contains Software and Hardware information of the system.
type Info struct {
	Hardware hardware.Info `json:"hardware"`
	Software software.Info `json:"software"`
	Platform platform.Info `json:"platform,omitzero"`
}

// New returns a new Collector.
func New(args ...Options) Collector {
	opts := &options{
		hw:  hardware.New(),
		sw:  software.New(),
		pl:  platform.New(),
		log: slog.Default(),
	}

	for _, opt := range args {
		opt(opts)
	}

	return Collector{
		hw:  opts.hw,
		sw:  opts.sw,
		pl:  opts.pl,
		log: opts.log,
	}
}

// Collect gathers system information and returns it.
// Will only return an error if platform, hardware, and software collection fail.
func (s Collector) Collect() (Info, error) {
	s.log.Debug("collecting sysinfo")

	plInfo, plErr := s.pl.Collect()
	if plErr != nil {
		s.log.Warn("failed to collect platform information", "error", plErr)
		plInfo = platform.Info{}
	}

	hwInfo, hwErr := s.hw.Collect(plInfo)
	swInfo, swErr := s.sw.Collect(plInfo)

	if plErr != nil {
		s.log.Warn("failed to collect platform information", "error", plErr)
	}

	if hwErr != nil {
		s.log.Warn("failed to collect hardware information", "error", hwErr)
	}
	if swErr != nil {
		s.log.Warn("failed to collect software information", "error", swErr)
	}
	if hwErr != nil && swErr != nil && plErr != nil {
		return Info{}, fmt.Errorf("failed to collect system information")
	}

	return Info{
		Platform: plInfo,
		Hardware: hwInfo,
		Software: swInfo,
	}, nil
}
