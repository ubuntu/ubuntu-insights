package software_test

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/insights/internal/collector/sysinfo/platform"
	"github.com/ubuntu/ubuntu-insights/insights/internal/collector/sysinfo/software"
	"github.com/ubuntu/ubuntu-insights/shared/testutils"
)

func TestCollectLinux(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		root         string
		missingFiles []string
		pInfo        platform.Info

		osInfo   string
		timezone string

		language        string
		missingLanguage bool

		logs    map[slog.Level]uint
		wantErr bool
	}{
		"Regular software information": {
			root:     "regular",
			osInfo:   "regular",
			timezone: "EST",
			language: "en_US",
		},

		"Missing OS information": {
			root:     "regular",
			osInfo:   "",
			timezone: "CEN",
			language: "fr_FR",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Missing distributor OS information": {
			root:     "regular",
			osInfo:   "no distributor",
			timezone: "EST",
			language: "fr_FR",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Missing release OS information": {
			root:     "regular",
			osInfo:   "no release",
			timezone: "EST",
			language: "fr_FR",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Error OS information": {
			root:     "regular",
			osInfo:   "error",
			timezone: "PST",
			language: "en_ZA",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Garbage OS information": {
			root:     "regular",
			osInfo:   "garbage",
			timezone: "EDT",
			language: "en-CA",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Missing language information": {
			root:            "regular",
			osInfo:          "regular",
			timezone:        "EST",
			missingLanguage: true,

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Partial BIOS information": {
			root:     "regular",
			osInfo:   "regular",
			timezone: "EST",
			language: "en_US",

			missingFiles: []string{
				"sys/class/dmi/id/bios_vendor",
			},

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Missing BIOS information": {
			root:     "empty",
			osInfo:   "regular",
			timezone: "JST",
			language: "ja",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 2,
			},
		},

		"Garbage BIOS information": {
			root:     "garbage",
			osInfo:   "regular",
			timezone: "EDT",
			language: "en-CA",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 2,
			},
		},
		"WSL": {
			root: "empty",
			pInfo: platform.Info{
				WSL: platform.WSL{
					SubsystemVersion: 2},
			},
			osInfo:   "regular",
			timezone: "JST",
			language: "ja",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tmp := t.TempDir()
			err := testutils.CopyDir(t, "testdata/linuxfs", tmp)
			require.NoError(t, err, "setup: failed to copy test data directory: ")

			root := filepath.Join(tmp, tc.root)
			for _, f := range tc.missingFiles {
				err := os.Remove(filepath.Join(root, f))
				require.NoError(t, err, "setup: failed to remove file %s: ", f)
			}

			options := []software.Options{
				software.WithRoot(root),
				software.WithTimezone(func() string { return tc.timezone }),
				software.WithLang(func() (string, bool) { return tc.language, !tc.missingLanguage }),
			}

			if tc.osInfo != "-" {
				cmdArgs := testutils.SetupFakeCmdArgs("TestFakeOSInfo", tc.osInfo)
				options = append(options, software.WithOSInfo(cmdArgs))
			}
			l := testutils.NewMockHandler(slog.LevelDebug)
			s := software.New(slog.New(&l), options...)

			got, err := s.Collect(tc.pInfo)

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
	case "no distributor":
		fmt.Println(`
Release:	24.04`)
	case "no release":
		fmt.Println(`
Distributor ID:	Ubuntu`)
	case "garbage":
		fmt.Println(`
ID:	664z708,as
sdlfk oabgr3w90
bam398b-9a:c;;;
zbnznr89;'`)
	case "":
		fallthrough
	case "missing":
		os.Exit(0)
	}
}
