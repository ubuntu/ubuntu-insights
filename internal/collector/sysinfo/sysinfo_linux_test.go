package sysinfo_test

import (
	"fmt"
	"log/slog"
	"os"
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
		root    string
		cpuInfo string

		logs    []testutils.ExpectedRecord
		wantErr bool
	}{
		"Regular hardware information": {
			root: "regular",
			// temporary raw literal until it is replaced with proper subprocess mocking
			cpuInfo: "regular",
		},

		"Missing hardware information is empty": {
			root:    "withoutinfo",
			cpuInfo: "",
			logs: []testutils.ExpectedRecord{
				{Level: slog.LevelWarn}, {Level: slog.LevelWarn}, {Level: slog.LevelWarn}, {Level: slog.LevelWarn}, {Level: slog.LevelWarn}, {Level: slog.LevelWarn},
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			l := testutils.NewMockHandler()

			options := []sysinfo.Options{
				sysinfo.WithRoot(filepath.Join("testdata", "linuxfs", tc.root)),
				sysinfo.WithLogger(&l),
			}

			if tc.cpuInfo != "-" {
				cmdArgs := []string{"env", "GO_WANT_HELPER_PROCESS=1", os.Args[0], "-test.run=TestMockCPUList", "--"}
				cmdArgs = append(cmdArgs, tc.cpuInfo)
				options = append(options, sysinfo.WithCpuInfo(cmdArgs))
			}

			s := sysinfo.New(options...)

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

func TestMockCPUList(_ *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	args := os.Args
	for len(args) > 0 {
		if args[0] != "--" {
			args = args[1:]
			continue
		}
		args = args[1:]
		break
	}

	switch args[0] {
	case "exit 1":
		fmt.Fprint(os.Stderr, "Error requested in Mock cpulist")
		os.Exit(1)
	case "regular":
		fmt.Println(`{
	"lscpu": [
		{
			"field": "Architecture:",
			"data": "x86_64"
		}
	]
}`)

	}
}
