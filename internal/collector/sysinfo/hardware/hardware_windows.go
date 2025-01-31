package hardware

import "log/slog"

type options struct {
	log *slog.Logger
}

func defaultOptions() *options {
	return &options{}
}

func (s Manager) collectProduct() (product, error) {
	return product{}, nil
}

func (s Manager) collectCPU() (cpu, error) {
	return cpu{}, nil
}

func (s Manager) collectGPUs() ([]gpu, error) {
	return []gpu{}, nil
}

func (s Manager) collectMemory() (memory, error) {
	return memory{}, nil
}

func (s Manager) collectDisks() ([]disk, error) {
	return []disk{}, nil
}

func (s Manager) collectScreens() ([]screen, error) {
	return []screen{}, nil
}
