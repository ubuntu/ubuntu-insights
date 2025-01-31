// Package sysinfo allows collecting "common" system information for all insight reports.
package sysinfo

import (
	"fmt"
	"log/slog"

	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/hardware"
	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/software"
)

// Collector describes a type that collects "common" information.
type Collector interface {
	Collect() (Info, error)
}

// Options is the variadic options available to the manager.
type Options func(*options)

type options struct {
	hw  hardware.Collector
	sw  software.Collector
	log *slog.Logger
}

// Manager handles dependencies for collecting software & hardware information.
// Manager implements sysinfo.Collector.
type Manager struct {
	hw  hardware.Collector
	sw  software.Collector
	log *slog.Logger
}

// Info contains Software and Hardware information of the system.
type Info struct {
	Hardware hardware.Info
	Software software.Info
}

// New returns a new SysInfo.
func New(args ...Options) Manager {
	opts := &options{
		hw:  hardware.New(),
		sw:  software.New(),
		log: slog.Default(),
	}

	for _, opt := range args {
		opt(opts)
	}

	return Manager{
		hw:  opts.hw,
		sw:  opts.sw,
		log: opts.log,
	}
}

// Collect gathers system information and returns it.
// Will only return an error if both hardware and software collection fail.
func (s Manager) Collect() (Info, error) {
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
