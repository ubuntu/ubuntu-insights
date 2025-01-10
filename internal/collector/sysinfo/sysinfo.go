// package sysinfo allows collecting "common" system information for all insight reports.
package sysinfo

import "log/slog"

type options struct {
	root string
	log  *slog.Logger
}

// Options is the variadic options available to the manager.
type Options func(*options)

// Manager allows collecting Software and Hardware information of the system.
type Manager struct {
	root string
	log  *slog.Logger
}

// SysInfo contains Software and Hardware information of the system.
type SysInfo struct {
	Hardware HwInfo
	Software SwInfo
}

// HwInfo is the hardware specific part.
type HwInfo struct {
	Product map[string]string

	Gpus []GpuInfo
}

// GpuInfo contains GPU information of a specific GPU
type GpuInfo struct {
	Gpu map[string]string
}

// SwInfo is the software specific part.
type SwInfo struct {
}

// New returns a new SysInfo.
func New(args ...Options) Manager {
	// options defaults
	opts := &options{
		root: "/",
		log:  slog.Default(),
	}

	for _, opt := range args {
		opt(opts)
	}

	return Manager{
		root: opts.root,
		log:  opts.log,
	}
}

// Collect gather system information and return it.
func (s Manager) Collect() (SysInfo, error) {
	hwInfo, err := s.collectHardware()
	if err != nil {
		return SysInfo{}, err
	}
	swInfo, err := s.collectSoftware()
	if err != nil {
		return SysInfo{}, err
	}

	return SysInfo{
		Hardware: hwInfo,
		Software: swInfo,
	}, nil
}
