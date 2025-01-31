// Package sysinfo allows collecting "common" system information for all insight reports.
package sysinfo

import (
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
	hw hardware.Collector
	sw software.Collector
}

// Manager handles dependencies for collecting software & hardware information.
// Manager implements sysinfo.Collector.
type Manager struct {
	hw hardware.Collector
	sw software.Collector
}

// Info contains Software and Hardware information of the system.
type Info struct {
	Hardware hardware.Info
	Software software.Info
}

// New returns a new SysInfo.
func New(args ...Options) Manager {
	opts := &options{
		hw: hardware.New(),
		sw: software.New(),
	}

	for _, opt := range args {
		opt(opts)
	}

	return Manager{
		hw: opts.hw,
		sw: opts.sw,
	}
}

// Collect gather system information and return it.
func (s Manager) Collect() (Info, error) {
	hwInfo, err := s.hw.Collect()
	if err != nil {
		return Info{}, err
	}
	swInfo, err := s.sw.Collect()
	if err != nil {
		return Info{}, err
	}

	return Info{
		Hardware: hwInfo,
		Software: swInfo,
	}, nil
}
