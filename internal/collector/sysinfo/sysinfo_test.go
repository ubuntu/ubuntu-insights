package sysinfo_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo"
	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/hardware"
	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/software"
)

// fakeCollector implements Collector (for several interfaces).
type fakeCollector[T any] struct {
	fn func() (T, error)
}

func (f fakeCollector[T]) Collect() (T, error) {
	return f.fn()
}

func makeFakeCollector[T any](info T, err error) fakeCollector[T] {
	return fakeCollector[T]{
		fn: func() (T, error) {
			return info, err
		},
	}
}

func TestNewLinux(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
	}{
		"Instantiate a sys info manager": {},
	}
	for name := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			s := sysinfo.New(
				sysinfo.WithHardwareCollector(makeFakeCollector(hardware.Info{}, nil)),
				sysinfo.WithSoftwareCollector(makeFakeCollector(software.Info{}, nil)),
			)

			require.NotEmpty(t, s, "sysinfo manager has custom fields")
		})
	}
}
