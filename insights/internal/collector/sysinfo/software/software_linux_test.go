package software_test

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/common/testutils"
	"github.com/ubuntu/ubuntu-insights/insights/internal/collector/sysinfo/platform"
	"github.com/ubuntu/ubuntu-insights/insights/internal/collector/sysinfo/software"
)

func TestCollectLinux(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		fixtures     []string
		missingFiles []string
		pInfo        platform.Info

		timezone string

		language        string
		missingLanguage bool

		logs map[slog.Level]uint
	}{
		"Regular software information": {
			fixtures: []string{"os/regular", "bios/regular"},
			timezone: "EST",
			language: "en_US",
		},

		"Vendor os-release used as fallback": {
			fixtures: []string{"os/vendor_only", "bios/partial"},
			timezone: "EST",
			language: "en_US",
		},

		"Snap hostfs os-release": {
			fixtures: []string{"os/snap", "bios/partial"},
			timezone: "EST",
			language: "en_US",
		},

		"Snap hostfs vendor only": {
			fixtures: []string{"os/snap_vendor_only", "bios/partial"},
			timezone: "EST",
			language: "en_US",
		},

		"Snap hostfs takes priority over local": {
			fixtures: []string{"os/regular", "os/snap", "bios/regular"},
			timezone: "EST",
			language: "en_US",
		},

		"No os-release file found": {
			fixtures: []string{"bios/regular"},
			timezone: "EST",
			language: "en_US",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Missing ID in os-release": {
			fixtures: []string{"os/no_distrib_id", "bios/partial"},
			timezone: "EST",
			language: "fr_FR",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Missing VERSION_ID in os-release": {
			fixtures: []string{"os/no_distrib_release", "bios/partial"},
			timezone: "EST",
			language: "fr_FR",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Garbage os-release file": {
			fixtures: []string{"os/garbage", "bios/regular"},
			timezone: "PST",
			language: "en_ZA",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"NAME present but no ID": {
			fixtures: []string{"os/name_no_id", "bios/regular"},
			timezone: "EST",
			language: "en_US",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"NAME differs from ID only by capitalization": {
			fixtures: []string{"os/name_capitalized", "bios/regular"},
			timezone: "EST",
			language: "en_US",
		},

		"NAME completely different from ID": {
			fixtures: []string{"os/name_different", "bios/regular"},
			timezone: "EST",
			language: "en_US",
		},

		"Single quoted os-release values": {
			fixtures: []string{"os/single_quotes", "bios/regular"},
			timezone: "EST",
			language: "en_US",
		},

		"Missing language information": {
			fixtures:        []string{"os/regular", "bios/regular"},
			timezone:        "EST",
			missingLanguage: true,

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Partial BIOS information": {
			fixtures: []string{"os/regular", "bios/regular"},
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
			fixtures: []string{"os/regular"},
			timezone: "JST",
			language: "ja",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 2,
			},
		},

		"Garbage BIOS information": {
			fixtures: []string{"os/regular", "bios/garbage"},
			timezone: "EDT",
			language: "en-CA",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 2,
			},
		},

		"WSL": {
			fixtures: []string{"os/regular"},
			pInfo: platform.Info{
				WSL: platform.WSL{
					SubsystemVersion: 2},
			},
			timezone: "JST",
			language: "ja",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			for _, fixture := range tc.fixtures {
				err := testutils.CopyDir(t, filepath.Join("testdata/linuxfs", fixture), root)
				require.NoError(t, err, "setup: failed to copy fixture %s", fixture)
			}

			for _, f := range tc.missingFiles {
				err := os.Remove(filepath.Join(root, f))
				require.NoError(t, err, "setup: failed to remove file %s: ", f)
			}

			options := []software.Options{
				software.WithRoot(root),
				software.WithTimezone(func() string { return tc.timezone }),
				software.WithLang(func() (string, bool) { return tc.language, !tc.missingLanguage }),
			}

			l := testutils.NewMockHandler(slog.LevelDebug)
			s := software.New(slog.New(&l), options...)

			got, err := s.Collect(tc.pInfo)

			if !l.AssertLevels(t, tc.logs) {
				l.OutputLogs(t)
			}

			require.NoError(t, err, "Collect should not return an error")

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			assert.Equal(t, want, got, "Collect should return expected software information")
		})
	}
}
