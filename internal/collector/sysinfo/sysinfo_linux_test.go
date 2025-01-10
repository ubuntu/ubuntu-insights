package sysinfo_test

import (
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo"
	"github.com/ubuntu/ubuntu-insights/internal/testutils"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
	}{
		"Instantiate a sys info manager": {},
	}
	for name, _ := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			s := sysinfo.New(sysinfo.WithRoot("/myspecialroot"))

			require.NotEmpty(t, s, "sysinfo manager has custom fields")
		})
	}
}

func TestCollect(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		root string

		logs    []testutils.ExpectedRecord
		wantErr bool
	}{
		"Regular hardware information": {root: "regular"},

		"Missing hardware information is empty": {
			root: "withoutinfo",
			logs: []testutils.ExpectedRecord{
				{Level: slog.LevelWarn}, {Level: slog.LevelWarn}, {Level: slog.LevelWarn}, {Level: slog.LevelWarn},
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			l := testutils.NewMockHandler()
			s := sysinfo.New(
				sysinfo.WithRoot(filepath.Join("testdata", "linuxfs", tc.root)),
				sysinfo.WithLogger(&l),
			)

			got, err := s.Collect()
			if tc.wantErr {
				require.Error(t, err, "Collect should return an error and didn’t")
				return
			}
			require.NoError(t, err, "Collect should not return an error")

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			assert.Equal(t, want, got, "Collect should return expected sys information")

			assert.Equal(t, len(tc.logs), len(l.HandleCalls), "Collect should log expected amount")
			for i, expect := range tc.logs {
				expect.Compare(t, l.HandleCalls[i])
			}
		})
	}
}
