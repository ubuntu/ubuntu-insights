// Package sysinfo allows collecting "common" system information for all insight reports.
package sysinfo

import (
	"os"
	"strings"
)

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

	CPU     map[string]string
	GPUs    []map[string]string
	Mem     map[string]int
	Blks    []diskInfo
	Screens []screenInfo
}

// DiskInfo contains Disk information of a disk or partition.
type diskInfo struct {
	Name string `json:"name"`
	Size string `json:"size"`

	Partitions []diskInfo `json:"partitions,omitempty"`
}

// ScreenInfo contains Screen information for a screen.
type screenInfo struct {
	Name               string `json:"name"`
	PhysicalResolution string `json:"physicalResolution"`
	Size               string `json:"size"`
	Resolution         string `json:"resolution"`
	RefreshRate        string `json:"refreshRate"`
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

// readFile returns the data in the file path, trimming whitespace, or "" on error.
func (s Manager) readFileDiscardError(path string) string {
	f, err := os.ReadFile(path)
	if err != nil {
		s.opts.log.Warn("failed to read file", "file", path, "error", err)
		return ""
	}

	return strings.TrimSpace(string(f))
}

// convertUnitToBytes takes a string bytes unit and converts value to bytes.
func (s Manager) convertUnitToBytes(unit string, value int) int {
	switch strings.ToLower(unit) {
	case "":
		fallthrough
	case "b":
		return value
	case "k":
		fallthrough
	case "kb":
		fallthrough
	case "kib":
		return value * 1024
	case "m":
		fallthrough
	case "mb":
		fallthrough
	case "mib":
		return value * 1024 * 1024
	case "g":
		fallthrough
	case "gb":
		fallthrough
	case "gib":
		return value * 1024 * 1024 * 1024
	case "t":
		fallthrough
	case "tb":
		fallthrough
	case "tib":
		return value * 1024 * 1024 * 1024 * 1024
	default:
		s.opts.log.Warn("unrecognized bytes unit", "unit", unit)
		return value
	}
}
