package sysinfo_test

import (
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo"
	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/hardware"
	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/software"
	"github.com/ubuntu/ubuntu-insights/internal/testutils"
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

func TestNew(t *testing.T) {
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

func TestCollect(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		hw    hardware.Info
		hwErr error

		sw    software.Info
		swErr error

		logs map[slog.Level]uint
	}{
		"Hardware and Software error is error": {
			hwErr: fmt.Errorf("fake hardware error"),
			swErr: fmt.Errorf("fake software error"),

			logs: map[slog.Level]uint{
				slog.LevelWarn: 2,
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			l := testutils.NewMockHandler(slog.LevelDebug)

			s := sysinfo.New(
				sysinfo.WithHardwareCollector(makeFakeCollector(tc.hw, tc.hwErr)),
				sysinfo.WithSoftwareCollector(makeFakeCollector(tc.sw, tc.swErr)),
				sysinfo.WithLogger(&l),
			)

			got, err := s.Collect()

			if tc.hwErr != nil && tc.swErr != nil {
				require.Error(t, err, "Collect should return an error and didn't")
				return
			}
			require.NoError(t, err, "Collect should not return an error and did")

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			assert.Equal(t, want, got, "Collect should return expected sys information")

			if !l.AssertLevels(t, tc.logs) {
				l.OutputLogs(t)
			}
		})
	}
}
