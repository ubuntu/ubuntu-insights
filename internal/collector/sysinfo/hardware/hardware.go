// Package hardware handles collecting "common" hardware information for all insight reports.
package hardware

// Info aggregates hardware info.
type Info struct {
	Product product

	CPU     cpu
	GPUs    []gpu
	Mem     memory
	Blks    []disk
	Screens []screen
}

type product map[string]string
type cpu map[string]string
type gpu map[string]string
type memory map[string]int

// DiskInfo contains information of a disk or partition.
type disk struct {
	Name string `json:"name"`
	Size string `json:"size"`

	Partitions []disk `json:"partitions,omitempty"`
}

// Screen contains information for a screen.
type screen struct {
	Name               string `json:"name"`
	PhysicalResolution string `json:"physicalResolution"`
	Size               string `json:"size"`
	Resolution         string `json:"resolution"`
	RefreshRate        string `json:"refreshRate"`
}

// Collector handles dependencies for collecting hardware information.
// Collector implements CollectorT[hardware.Info].
type Collector struct {
	opts options
}

// Options are the variadic options available to the Collector.
type Options func(*options)

// New returns a new Collector.
func New(args ...Options) Collector {
	// options defaults are platform dependent.
	opts := defaultOptions()
	for _, opt := range args {
		opt(opts)
	}

	return Collector{
		opts: *opts,
	}
}

// Collect aggregates the data from all the other hardware collect functions.
func (s Collector) Collect() (info Info, err error) {
	s.opts.log.Debug("collecting hardware info")

	info.Product, err = s.collectProduct()
	if err != nil {
		s.opts.log.Warn("failed to collect Product info", "error", err)
		info.Product = product{}
	}

	info.CPU, err = s.collectCPU()
	if err != nil {
		s.opts.log.Warn("failed to collect CPU info", "error", err)
		info.CPU = cpu{}
	}

	info.GPUs, err = s.collectGPUs()
	if err != nil {
		s.opts.log.Warn("failed to collect GPU info", "error", err)
		info.GPUs = []gpu{}
	}

	info.Mem, err = s.collectMemory()
	if err != nil {
		s.opts.log.Warn("failed to collect memory info", "error", err)
		info.Mem = memory{}
	}

	info.Blks, err = s.collectDisks()
	if err != nil {
		s.opts.log.Warn("failed to collect disk info", "error", err)
		info.Blks = []disk{}
	}

	info.Screens, err = s.collectScreens()
	if err != nil {
		s.opts.log.Warn("failed to collect screen info", "error", err)
		info.Screens = []screen{}
	}

	return info, nil
}
