// Package sysinfo allows collecting "common" system information for all insight reports.
package sysinfo

import (
	"fmt"
	"log/slog"

	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/hardware"
	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/software"
)

// CollectorT describes a type that collects some information T.
type CollectorT[T any] interface {
	Collect() (T, error)
}

// Options is the variadic options available to the Collector.
type Options func(*options)

type options struct {
	hw  CollectorT[hardware.Info]
	sw  CollectorT[software.Info]
	log *slog.Logger
}

// Collector handles dependencies for collecting software & hardware information.
// Collector implements CollectorT[sysinfo.Info].
type Collector struct {
	hw  CollectorT[hardware.Info]
	sw  CollectorT[software.Info]
	log *slog.Logger
}

// Info contains Software and Hardware information of the system.
type Info struct {
	Hardware hardware.Info `json:"hardware"`
	Software software.Info `json:"software"`
}

// New returns a new SysInfo.
func New(source software.Source, tipe string, args ...Options) Collector {
	opts := &options{
		hw:  hardware.New(),
		sw:  software.New(source, tipe),
		log: slog.Default(),
	}

	for _, opt := range args {
		opt(opts)
	}

	return Collector{
		hw:  opts.hw,
		sw:  opts.sw,
		log: opts.log,
	}
}

// Collect gathers system information and returns it.
// Will only return an error if both hardware and software collection fail.
func (s Collector) Collect() (Info, error) {
	s.log.Debug("collecting sysinfo")

	hwInfo, hwErr := s.hw.Collect()
	swInfo, swErr := s.sw.Collect()

	if hwErr != nil {
		s.log.Warn("failed to collect hardware information", "error", hwErr)
	}
	if swErr != nil {
		s.log.Warn("failed to collect software information", "error", swErr)
	}
	if hwErr != nil && swErr != nil {
		return Info{}, fmt.Errorf("failed to collect system information")
	}

	return Info{
		Hardware: hwInfo,
		Software: swInfo,
	}, nil
}
