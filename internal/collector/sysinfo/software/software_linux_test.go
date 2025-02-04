package software_test

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/software"
	"github.com/ubuntu/ubuntu-insights/internal/testutils"
)

func TestCollectLinux(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		root         string
		missingFiles []string

		src      software.Source
		tipe     string
		osInfo   string
		timezone string

		language        string
		missingLanguage bool

		logs    map[slog.Level]uint
		wantErr bool
	}{
		"Regular software information": {
			root: "regular",
			src: software.Source{
				Name:    "test",
				Version: "v1.2.3",
			},
			tipe:     software.TypeRegular,
			osInfo:   "regular",
			timezone: "EST",
			language: "en_US",
		},

		"Missing OS information": {
			root: "regular",
			src: software.Source{
				Name:    "test souce",
				Version: "v4.3.2",
			},
			tipe:     software.TypeManual,
			osInfo:   "",
			timezone: "CEN",
			language: "fr_FR",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Error OS information": {
			root: "regular",
			src: software.Source{
				Name:    "test",
				Version: "v1.1.1",
			},
			tipe:     software.TypeInstall,
			osInfo:   "error",
			timezone: "PST",
			language: "en_ZA",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Missing language information": {
			root: "regular",
			src: software.Source{
				Name:    "test",
				Version: "v1.7.10",
			},
			tipe:            software.TypeRegular,
			osInfo:          "regular",
			timezone:        "EST",
			missingLanguage: true,

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Missing BIOS information": {
			root: "empty",
			src: software.Source{
				Name:    "src",
				Version: "v1.6.4",
			},
			tipe:     software.TypeRegular,
			osInfo:   "regular",
			timezone: "JST",
			language: "ja",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 2,
			},
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

			l := testutils.NewMockHandler(slog.LevelDebug)

			options := []software.Options{
				software.WithRoot(root),
				software.WithLogger(&l),
				software.WithTimezone(func() string { return tc.timezone }),
				software.WithLang(func() (string, bool) { return tc.language, !tc.missingLanguage }),
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
