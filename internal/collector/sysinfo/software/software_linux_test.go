package software_test

import (
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/software"
	"github.com/ubuntu/ubuntu-insights/internal/testutils"
)

func TestCollectLinux(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		src      software.Source
		tipe     string
		osInfo   string
		timezone string

		logs    map[slog.Level]uint
		wantErr bool
	}{
		"Regular software information": {
			src: software.Source{
				Name:    "test",
				Version: "v1.2.3",
			},
			tipe:     software.TypeRegular,
			osInfo:   "regular",
			timezone: "EST",
		},

		"Missing OS information": {
			src: software.Source{
				Name:    "test",
				Version: "v1.2.3",
			},
			tipe:     software.TypeManual,
			osInfo:   "",
			timezone: "CEN",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Error OS information": {
			src: software.Source{
				Name:    "test",
				Version: "v1.2.3",
			},
			tipe:     software.TypeInstall,
			osInfo:   "error",
			timezone: "PST",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			l := testutils.NewMockHandler(slog.LevelDebug)

			options := []software.Options{
				software.WithLogger(&l),
				software.WithTimezone(func() string { return tc.timezone }),
			}

			if tc.osInfo != "-" {
				cmdArgs := testutils.SetupFakeCmdArgs("TestFakeOSInfo", tc.osInfo)
				options = append(options, software.WithOSInfo(cmdArgs))
			}

			s := software.New(tc.src, tc.tipe, options...)

			got, err := s.Collect()

			if !l.AssertLevels(t, tc.logs) {
				l.OutputLogs(t)
			}

			if tc.wantErr {
				require.Error(t, err, "Collect should return an error and didn't")
				return
			}
			require.NoError(t, err, "Collect should not return an error")

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			assert.Equal(t, want, got, "Collect should return expected software information")
		})
	}
}

func TestFakeOSInfo(_ *testing.T) {
	args, err := testutils.GetFakeCmdArgs()
	if err != nil {
		return
	}
	defer os.Exit(0)

	switch args[0] {
	case "error":
		fmt.Fprint(os.Stderr, "Error requested in fake lsb_release")
		os.Exit(1)
	case "regular":
		fmt.Println(`No LSB modules are available.
Distributor ID:	Ubuntu
Description:	Ubuntu 24.04.1 LTS
Release:	24.04
Codename:	noble`)
	case "":
		fallthrough
	case "missing":
		os.Exit(0)
	}
}
