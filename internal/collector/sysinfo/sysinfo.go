// Package sysinfo allows collecting "common" system information for all insight reports.
package sysinfo

// Options is the variadic options available to the manager.
type Options func(*options)

// Manager allows collecting Software and Hardware information of the system.
type Manager struct {
	opts options
}

// SysInfo contains Software and Hardware information of the system.
type SysInfo struct {
	Hardware hwInfo
	Software swInfo
}

// HwInfo is the hardware specific part.
type hwInfo struct {
	Product map[string]string

	CPU     cpuInfo
	GPUs    []gpuInfo
	Mem     memInfo
	Blks    []diskInfo
	Screens []screenInfo
}

// CpuInfo contains CPU information of a machine.
type cpuInfo struct {
	CPU map[string]string
}

// GpuInfo contains GPU information of a specific GPU.
type gpuInfo struct {
	GPU map[string]string
}

// MemInfo contains Memory information of a machine.
type memInfo struct {
	Mem map[string]int
}

// DiskInfo contains Disk information of a disk or partition.
type diskInfo struct {
	Name string
	Size string

	Partitions []diskInfo
}

// ScreenInfo contains Screen information for a screen.
type screenInfo struct {
	Name               string
	PhysicalResolution string
	Size               string
	Resolution         string
	RefreshRate        string
}

// SwInfo is the software specific part.
type swInfo struct {
}

// New returns a new SysInfo.
func New(args ...Options) Manager {
	// options defaults
	opts := defaultOptions()
	for _, opt := range args {
		opt(opts)
	}

	return Manager{
		opts: *opts,
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
