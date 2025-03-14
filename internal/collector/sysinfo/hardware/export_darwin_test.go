package hardware

import "encoding/xml"

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
