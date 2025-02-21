package software_test

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/software"
	"github.com/ubuntu/ubuntu-insights/internal/testutils"
)

func TestMain(m *testing.M) {
	flag.Parse()
	dir, ok := testutils.SetupHelperCoverdir()

	r := m.Run()
	if ok {
		os.Remove(dir)
	}
	os.Exit(r)
}

func TestCollectMacos(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		timezone string

		osInfo   string
		language string
		bios     string

		logs    map[slog.Level]uint
		wantErr bool
	}{
		"Regular software information": {
			timezone: "EST",

			osInfo:   "regular",
			language: "regular",
			bios:     "regular",
		},

		"Missing OS information": {
			timezone: "CEN",

			osInfo:   "",
			language: "regular",
			bios:     "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Error OS information": {
			timezone: "PST",

			osInfo:   "error",
			language: "regular",
			bios:     "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Missing language information": {
			timezone: "EST",

			osInfo:   "regular",
			language: "",
			bios:     "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Error language information": {
			timezone: "EST",

			osInfo:   "regular",
			language: "error",
			bios:     "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Missing BIOS information": {
			timezone: "EST",

			osInfo:   "regular",
			language: "regular",
			bios:     "",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Error BIOS information": {
			timezone: "EST",

			osInfo:   "regular",
			language: "regular",
			bios:     "error",

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

			if tc.language != "-" {
				cmdArgs := testutils.SetupFakeCmdArgs("TestFakeLangInfo", tc.language)
				options = append(options, software.WithLang(cmdArgs))
			}

			if tc.language != "-" {
				cmdArgs := testutils.SetupFakeCmdArgs("TestFakeBIOSInfo", tc.bios)
				options = append(options, software.WithBIOS(cmdArgs))
			}

			s := software.New(options...)

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
		fmt.Fprint(os.Stderr, "Error requested in fake os info")
		os.Exit(1)
	case "regular":
		fmt.Println(`
ProductName:	Mac OS X
ProductVersion: 10.11.1
BuildVersion:   15B42`)
	case "":
		fallthrough
	case "missing":
		os.Exit(0)
	}
}

func TestFakeLangInfo(_ *testing.T) {
	args, err := testutils.GetFakeCmdArgs()
	if err != nil {
		return
	}
	defer os.Exit(0)

	switch args[0] {
	case "error":
		fmt.Fprint(os.Stderr, "Error requested in fake language")
		os.Exit(1)
	case "regular":
		fmt.Println(`en_US`)
	case "":
		fallthrough
	case "missing":
		os.Exit(0)
	}
}

func TestFakeBIOSInfo(_ *testing.T) {
	args, err := testutils.GetFakeCmdArgs()
	if err != nil {
		return
	}
	defer os.Exit(0)

	switch args[0] {
	case "error":
		fmt.Fprint(os.Stderr, "Error requested in BIOS info")
		os.Exit(1)
	case "regular":
		fmt.Println(`Hardware:

    Hardware Overview:

      Model Name: Mac mini
      Model Identifier: Macmini6,2
      Processor Name: Quad-Core Intel Core i7
      Processor Speed: 2.3 GHz
      Number of Processors: 1
      Total Number of Cores: 4
      L2 Cache (per Core): 256 KB
      L3 Cache: 6 MB
      Hyper-Threading Technology: Enabled
      Memory: 16 GB
      Boot ROM Version: 429.0.0.0.0
      SMC Version (system): 2.8f1`)
	case "":
		fallthrough
	case "missing":
		os.Exit(0)
	}
}
