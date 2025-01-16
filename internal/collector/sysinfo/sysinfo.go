// package sysinfo allows collecting "common" system information for all insight reports.
package sysinfo

// Options is the variadic options available to the manager.
type Options func(*options)

// Manager allows collecting Software and Hardware information of the system.
type Manager struct {
	opts options
}

// SysInfo contains Software and Hardware information of the system.
type SysInfo struct {
	Hardware HwInfo
	Software SwInfo
}

// HwInfo is the hardware specific part.
type HwInfo struct {
	Product map[string]string

	Cpu     CpuInfo
	Gpus    []GpuInfo
	Mem     MemInfo
	Blks    []DiskInfo
	Screens []ScreenInfo
}

// CpuInfo contains CPU information of a machine.
type CpuInfo struct {
	Cpu map[string]string
}

// GpuInfo contains GPU information of a specific GPU.
type GpuInfo struct {
	Gpu map[string]string
}

// MemInfo contains Memory information of a machine.
type MemInfo struct {
	Mem map[string]int
}

type DiskInfo struct {
	Name string
	Size string

	Partitions []DiskInfo
}

type ScreenInfo struct {
	Name               string
	PhysicalResolution string
	Size               string
	Resolution         string
	RefreshRate        string
}

// SwInfo is the software specific part.
type SwInfo struct {
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
