package sysinfo_test

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo"
	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/hardware"
	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/platform"
	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/software"
	"github.com/ubuntu/ubuntu-insights/internal/testutils"
)

// fakePCollector implements PCollector (for several interfaces).
type fakePCollector[T any] struct {
	fn func() (T, error)
}

func (f fakePCollector[T]) Collect(platform.Info) (T, error) {
	return f.fn()
}

func makeFakePCollector[T any](info T, err error) fakePCollector[T] {
	return fakePCollector[T]{
		fn: func() (T, error) {
			return info, err
		},
	}
}

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
				sysinfo.WithHardwareCollector(makeFakePCollector(hardware.Info{}, nil)),
				sysinfo.WithSoftwareCollector(makeFakePCollector(software.Info{}, nil)),
				sysinfo.WithPlatformCollector(makeFakeCollector(platform.Info{}, nil)),
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

		p    platform.Info
		pErr error

		logs map[slog.Level]uint
	}{
		"Hardware and Software error is error": {
			hwErr: fmt.Errorf("fake hardware error"),
			swErr: fmt.Errorf("fake software error"),

			logs: map[slog.Level]uint{
				slog.LevelWarn: 2,
			},
		},
		"Platform, Hardware and Software error is error": {
			hwErr: fmt.Errorf("fake hardware error"),
			swErr: fmt.Errorf("fake software error"),
			pErr:  fmt.Errorf("fake platform error"),
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			l := testutils.NewMockHandler(slog.LevelDebug)

			s := sysinfo.New(
				sysinfo.WithHardwareCollector(makeFakePCollector(tc.hw, tc.hwErr)),
				sysinfo.WithSoftwareCollector(makeFakePCollector(tc.sw, tc.swErr)),
				sysinfo.WithPlatformCollector(makeFakeCollector(tc.p, tc.pErr)),
				sysinfo.WithLogger(&l),
			)

			got, err := s.Collect()

			if tc.hwErr != nil && tc.swErr != nil && tc.pErr != nil {
				require.Error(t, err, "Collect should return an error and didn't")
				return
			}
			require.NoError(t, err, "Collect should not return an error and did")
			sGot, err := json.Marshal(got)
			require.NoError(t, err, "Collect should marshal sys information")

			want := testutils.LoadWithUpdateFromGolden(t, string(sGot))
			assert.Equal(t, want, string(sGot), "Collect should return expected sys information")

			if !l.AssertLevels(t, tc.logs) {
				l.OutputLogs(t)
			}
		})
	}
}
