package hardware

type platformOptions struct {
}

func defaultPlatformOptions() platformOptions {
	return platformOptions{}
}

func (s Collector) collectProduct() (product, error) {
	return product{}, nil
}

func (s Collector) collectCPU() (cpu, error) {
	return cpu{}, nil
}

func (s Collector) collectGPUs() ([]gpu, error) {
	return []gpu{}, nil
}

func (s Collector) collectMemory() (memory, error) {
	return memory{}, nil
}

func (s Collector) collectDisks() ([]disk, error) {
	return []disk{}, nil
}

func (s Collector) collectScreens() ([]screen, error) {
	return []screen{}, nil
}
