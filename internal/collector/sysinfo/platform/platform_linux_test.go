package platform_test

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"unicode/utf16"

	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/platform"
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

func TestNewLinux(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
	}{
		"Instantiate a platform sys info Collector": {},
	}
	for name := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			s := platform.New(platform.WithRoot("/myspecialroot"))

			require.NotEmpty(t, s, "platform sysinfo Collector has custom fields")
		})
	}
}

func TestCollectLinux(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		detectVirtCmd string
		wslVersionCmd string
		wslStatusCmd  string
		proStatusCmd  string

		logs    map[slog.Level]uint
		wantErr bool
	}{
		"Regular platform information - pro attached": {
			detectVirtCmd: "regular",
			wslVersionCmd: "error",
			proStatusCmd:  "attached",
		},
		"Regular platform information - pro detached": {
			detectVirtCmd: "regular",
			wslVersionCmd: "error",
			wslStatusCmd:  "error",
			proStatusCmd:  "detached",
		},
		"Garbage platform information": {
			detectVirtCmd: "garbage",
			wslVersionCmd: "error",
			wslStatusCmd:  "error",
			proStatusCmd:  "garbage",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},
		"Empty platform information": {
			detectVirtCmd: "",
			wslVersionCmd: "",
			wslStatusCmd:  "",
			proStatusCmd:  "",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},
		"Error platform information": {
			detectVirtCmd: "error",
			wslVersionCmd: "error",
			wslStatusCmd:  "error",
			proStatusCmd:  "error",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 2,
			},
		},
		"Regular platform information - WSL": {
			detectVirtCmd: "wsl",
			wslVersionCmd: "regular",
			wslStatusCmd:  "regular",
			proStatusCmd:  "attached",
		},
		"Regular platform information - WSL pro detached": {
			detectVirtCmd: "wsl",
			wslVersionCmd: "regular",
			wslStatusCmd:  "regular",
			proStatusCmd:  "detached",
		},
		"Regular platform information - WSL no interop": {
			detectVirtCmd: "wsl",
			wslVersionCmd: "regular",
			wslStatusCmd:  "error",
			proStatusCmd:  "attached",
		},
		"Empty platform information - WSL": {
			detectVirtCmd: "wsl",
			wslVersionCmd: "",
			wslStatusCmd:  "",
			proStatusCmd:  "",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 3,
			},
		},
		"Empty platform information - WSL no interop": {
			detectVirtCmd: "wsl",
			wslVersionCmd: "",
			wslStatusCmd:  "error",
			proStatusCmd:  "",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},
		"Garbage platform information - WSL": {
			detectVirtCmd: "garbage",
			wslVersionCmd: "garbage",
			wslStatusCmd:  "garbage",
			proStatusCmd:  "garbage",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},
		"Garbage platform information - WSL no interop": {
			detectVirtCmd: "garbage",
			wslVersionCmd: "garbage",
			wslStatusCmd:  "error",
			proStatusCmd:  "garbage",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			l := testutils.NewMockHandler(slog.LevelDebug)
			options := []platform.Options{
				platform.WithRoot(t.TempDir()),
				platform.WithLogger(&l),
			}

			if tc.detectVirtCmd != "-" {
				cmdArgs := testutils.SetupFakeCmdArgs("TestFakeVirtInfo", tc.detectVirtCmd)
				options = append(options, platform.WithDetectVirtCmd(cmdArgs))
			}

			if tc.wslVersionCmd != "-" {
				cmdArgs := testutils.SetupFakeCmdArgs("TestWSLVersionInfo", tc.wslVersionCmd)
				options = append(options, platform.WithWSLVersionCmd(cmdArgs))
			}

			if tc.wslStatusCmd != "-" {
				cmdArgs := testutils.SetupFakeCmdArgs("TestFakeWSLStatus", tc.wslStatusCmd)
				options = append(options, platform.WithWSLStatusCmd(cmdArgs))
			}

			if tc.proStatusCmd != "-" {
				cmdArgs := testutils.SetupFakeCmdArgs("TestFakeProStatus", tc.proStatusCmd)
				options = append(options, platform.WithProStatusCmd(cmdArgs))
			}

			p := platform.New(options...)

			got, err := p.Collect()

			if !l.AssertLevels(t, tc.logs) {
				l.OutputLogs(t)
			}

			if tc.wantErr {
				require.Error(t, err, "Collect should return an error and didn't")
				return
			}
			require.NoError(t, err, "Collect should not return an error")

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			require.Equal(t, want, got, "Collect should return expected platform information")
		})
	}
}

func TestFakeVirtInfo(_ *testing.T) {
	args, err := testutils.GetFakeCmdArgs()
	if err != nil {
		return
	}
	defer os.Exit(0)

	switch args[0] {
	case "error":
		fmt.Fprintf(os.Stderr, "Error requested in fake systemd-detect-virt")
		os.Exit(1)
	case "regular":
		fmt.Println("uml")
	case "wsl":
		fmt.Println("wsl")
	case "garbage":
		fmt.Println("garbage 🗑️")
	case "":
		fallthrough
	case "missing":
		os.Exit(0)
	}
}

func TestWSLVersionInfo(_ *testing.T) {
	args, err := testutils.GetFakeCmdArgs()
	if err != nil {
		return
	}
	defer os.Exit(0)

	var str string
	file := os.Stdout
	switch args[0] {
	case "error":
		file = os.Stderr
		str = "Error requested in fake wsl.exe --version"
		defer os.Exit(1)
	case "regular":
		str = `
WSL version: 2.4.11.0
Kernel version: 5.15.167.4-1
WSLg version: 1.0.65
MSRDC version: 1.2.5716
Direct3D version: 1.611.1-81528511
DXCore version: 10.0.26100.1-240331-1435.ge-release
Windows version: 10.0.26100.3194`
	case "missing WSL version":
		str = `
Kernel version: 5.15.167.4-1
WSLg version: 1.0.65
MSRDC version: 1.2.5716
Direct3D version: 1.611.1-81528511
DXCore version: 10.0.26100.1-240331-1435.ge-release
Windows version: 10.0.26100.3194`
	case "missing kernel version":
		str = `
WSL version: 2.4.11.0
WSLg version: 1.0.65
MSRDC version: 1.2.5716
Direct3D version: 1.611.1-81528511
DXCore version: 10.0.26100.1-240331-1435.ge-release
Windows version: 10.0.26100.3194`
	case "garbage":
		str = `
WSL version 🗑️: 2.4.11.0
🗑️ version: 5.15.167.4-1
WSLg version: 1.0.65
MSRDC version: 1.2.5716🗑️`
	case "":
		fallthrough
	case "missing":
		os.Exit(0)
	}

	// Convert the string to UTF-16
	encoded := utf16.Encode([]rune(str))

	// Convert the UTF-16 encoded data to bytes
	var utf16Bytes []byte
	for _, r := range encoded {
		utf16Bytes = append(utf16Bytes, byte(r), byte(r>>8))
	}

	if _, err = file.Write(utf16Bytes); err != nil {
		panic(fmt.Errorf("failed to write to file: %w", err))
	}
}

func TestFakeWSLStatus(_ *testing.T) {
	args, err := testutils.GetFakeCmdArgs()
	if err != nil {
		return
	}
	defer os.Exit(0)

	file := os.Stdout
	var str string
	switch args[0] {
	case "error":
		file = os.Stderr
		str = "Error requested in fake wsl.exe --status"
		defer os.Exit(1)
	case "regular":
		str = `
Default Distribution: Ubuntu
Default Version: 2`
	case "garbage":
		str = `🗑️🗑️🗑️`
	case "":
		fallthrough
	case "missing":
		os.Exit(0)
	}

	// Convert the string to UTF-16
	encoded := utf16.Encode([]rune(str))

	// Convert the UTF-16 encoded data to bytes
	var utf16Bytes []byte
	for _, r := range encoded {
		utf16Bytes = append(utf16Bytes, byte(r), byte(r>>8))
	}

	if _, err = file.Write(utf16Bytes); err != nil {
		panic(fmt.Errorf("failed to write to file: %w", err))
	}
}

func TestFakeProStatus(_ *testing.T) {
	args, err := testutils.GetFakeCmdArgs()
	if err != nil {
		return
	}
	defer os.Exit(0)

	switch args[0] {
	case "error":
		fmt.Fprintf(os.Stderr, "Error requested in fake pro api")
		os.Exit(1)
	case "attached":
		fmt.Println(`
{"_schema_version": "v1", "data": {"attributes": {"contract_remaining_days": 2912745, "contract_status": "active", "is_attached": true, "is_attached_and_contract_valid": true}, "meta": {"environment_vars": []}, "type": "IsAttached"}, "errors": [], "result": "success", "version": "34~24.04", "warnings": []}`)
	case "detached":
		fmt.Println(`
{"_schema_version": "v1", "data": {"attributes": {"contract_remaining_days": 0, "contract_status": null, "is_attached": false, "is_attached_and_contract_valid": false}, "meta": {"environment_vars": []}, "type": "IsAttached"}, "errors": [], "result": "success", "version": "34~24.04", "warnings": []}`)
	case "garbage":
		fmt.Println(`
{"_schema_version": "v1", "data": {"attributes": {"contract_remaining_days": "idk", "contract_status": "yay", "is_attached": 12345, "is_attached_and_contract_valid": true}}`)
	case "empty":
		fmt.Println(`{}`)
	case "":
		fallthrough
	case "missing":
		os.Exit(0)
	}
}
