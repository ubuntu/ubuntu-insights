// Package hardware handles collecting "common" hardware information for all insight reports.
package hardware

import (
	"log/slog"
	"runtime"

	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/platform"
)

// Info aggregates hardware info.
type Info struct {
	Product product  `json:"product,omitzero"`
	CPU     cpu      `json:"cpu,omitzero"`
	GPUs    []gpu    `json:"gpus,omitempty"`
	Mem     memory   `json:"memory,omitzero"`
	Blks    []disk   `json:"disks,omitempty"`
	Screens []screen `json:"screens,omitempty"`
}

// product contains information for a system's product.
type product struct {
	Family string `json:"family"`
	Name   string `json:"name"`
	Vendor string `json:"vendor"`
}

// cpu contains information for a system's cpus.
type cpu struct {
	Name    string `json:"name"`
	Vendor  string `json:"vendor"`
	Arch    string `json:"architecture"`
	Cpus    uint64 `json:"cpus"`
	Sockets uint64 `json:"sockets"`
	Cores   uint64 `json:"coresPerSocket"`
	Threads uint64 `json:"threadsPerCore"`
}

// gpu contains information for a gpu.
type gpu struct {
	Name   string `json:"name,omitempty"`
	Device string `json:"device,omitempty"`
	Vendor string `json:"vendor"`
	Driver string `json:"driver"`
}

// memory contains information for the system's memory.
type memory struct {
	Total int `json:"size"`
}

// disk contains information of a disk or partition.
type disk struct {
	Size uint64 `json:"size"`

	Partitions []disk `json:"partitions,omitempty"`
}

// screen contains information for a screen.
type screen struct {
	PhysicalResolution string `json:"physicalResolution,omitempty"`
	Size               string `json:"size,omitempty"`
	Resolution         string `json:"resolution,omitempty"`
	RefreshRate        string `json:"refreshRate,omitempty"`
}

// Collector handles dependencies for collecting hardware information.
// Collector implements CollectorT[hardware.Info].
type Collector struct {
	log  *slog.Logger
	arch string

	platform platformOptions
}

// Options are the variadic options available to the Collector.
type Options func(*options)

type options struct {
	log  *slog.Logger
	arch string

	platform platformOptions
}

// New returns a new Collector.
func New(args ...Options) Collector {
	opts := &options{
		log:  slog.Default(),
		arch: runtime.GOARCH,
	}
	opts.platform = defaultPlatformOptions()

	for _, opt := range args {
		opt(opts)
	}

	return Collector{
		log:  opts.log,
		arch: opts.arch,

		platform: opts.platform,
	}
}

// Collect aggregates the data from all the other hardware collect functions.
func (h Collector) Collect(pi platform.Info) (info Info, err error) {
	h.log.Debug("collecting hardware info")

	info.Product, err = h.collectProduct(pi)
	if err != nil {
		h.log.Warn("failed to collect Product info", "error", err)
		info.Product = product{}
	}

	info.CPU, err = h.collectCPU()
	if err != nil {
		h.log.Warn("failed to collect CPU info", "error", err)
		info.CPU = cpu{
			Arch: h.arch,
		}
	}

	info.GPUs, err = h.collectGPUs(pi)
	if err != nil {
		h.log.Warn("failed to collect GPU info", "error", err)
		info.GPUs = []gpu{}
	}

	info.Mem, err = h.collectMemory()
	if err != nil {
		h.log.Warn("failed to collect memory info", "error", err)
		info.Mem = memory{}
	}

	info.Blks, err = h.collectDisks()
	if err != nil {
		h.log.Warn("failed to collect disk info", "error", err)
		info.Blks = []disk{}
	}

	info.Screens, err = h.collectScreens(pi)
	if err != nil {
		h.log.Warn("failed to collect screen info", "error", err)
		info.Screens = []screen{}
	}

	return info, nil
}
