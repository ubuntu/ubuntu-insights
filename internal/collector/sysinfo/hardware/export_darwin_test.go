package hardware

import (
	"encoding/xml"
	"log/slog"
)

// WithCpuInfo overrides default CPU info.
func WithCPUInfo(cmd []string) Options {
	return func(o *options) {
		o.platform.cpuCmd = cmd
	}
}

// WithGPUInfo overrides default GPU info.
func WithGPUInfo(cmd []string) Options {
	return func(o *options) {
		o.platform.gpuCmd = cmd
	}
}

// WithMemoryInfo overrides default memory info.
func WithMemoryInfo(cmd []string) Options {
	return func(o *options) {
		o.platform.memCmd = cmd
	}
}

// WithDiskInfo overrides default disk info.
func WithDiskInfo(cmd []string) Options {
	return func(o *options) {
		o.platform.diskCmd = cmd
	}
}

// WithScreenInfo overrides default screen info.
func WithScreenInfo(cmd []string) Options {
	return func(o *options) {
		o.platform.screenCmd = cmd
	}
}

// ParsePListDict exports parsePListDict.
func ParsePListDict(start xml.StartElement, dec *xml.Decoder) (map[string]any, error) {
	return parsePListDict(start, dec)
}

// Disk appeases the linter.
type Disk = disk

// ParseDiskDict exports parseDiskDict.
func ParseDiskDict(data map[string]any, partition bool, log *slog.Logger) (Disk, error) {
	return parseDiskDict(data, partition, log)
}
