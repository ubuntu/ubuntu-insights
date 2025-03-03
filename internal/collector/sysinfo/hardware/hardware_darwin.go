package hardware

import "github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/platform"

type platformOptions struct {
}

func defaultPlatformOptions() platformOptions {
	return platformOptions{}
}

func (s Collector) collectProduct(_ platform.Info) (product, error) {
	return product{}, nil
}

func (s Collector) collectCPU() (cpu, error) {
	return cpu{}, nil
}

func (s Collector) collectGPUs(_ platform.Info) ([]gpu, error) {
	return []gpu{}, nil
}

func (s Collector) collectMemory() (memory, error) {
	return memory{}, nil
}

func (s Collector) collectDisks() ([]disk, error) {
	return []disk{}, nil
}

func (s Collector) collectScreens(_ platform.Info) ([]screen, error) {
	return []screen{}, nil
}
