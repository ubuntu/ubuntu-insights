package platform_test

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf16"

	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/platform"
	"github.com/ubuntu/ubuntu-insights/internal/testutils"
)

func TestNewLinux(t *testing.T) {
	t.Parallel()

	s := platform.New(platform.WithRoot("/myspecialroot"))
	require.NotEmpty(t, s, "platform sysinfo Collector has custom fields")
}

func TestCollectLinux(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		roots         []string
		detectVirtCmd string
		wslVersionCmd string
		proStatusCmd  string

		missingFiles []string

		logs    map[slog.Level]uint
		wantErr bool
	}{
		// Non-WSL
		"Non-WSL Basic with Pro Attached": {
			detectVirtCmd: "regular",
			wslVersionCmd: "error",
			proStatusCmd:  "attached",
		},
		"Non-WSL Basic with Pro Detached": {
			detectVirtCmd: "regular",
			wslVersionCmd: "error",
			proStatusCmd:  "detached",
		},
		"Non-WSL Garbage Returns from Commands warns": {
			detectVirtCmd: "garbage",
			wslVersionCmd: "error",
			proStatusCmd:  "garbage",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},
		"Non-WSL Empty Returns from Commands warns": {
			detectVirtCmd: "",
			wslVersionCmd: "",
			proStatusCmd:  "",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},
		"Non-WSL Error Returns from Commands warns": {
			detectVirtCmd: "error",
			wslVersionCmd: "error",
			proStatusCmd:  "error",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 2,
			},
		},
		"Non-WSL Print to StdOut without exitcode warns": {
			detectVirtCmd: "error no exit",
			wslVersionCmd: "error no exit",
			proStatusCmd:  "error no exit",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
				slog.LevelInfo: 2,
			},
		},

		// WSL 2
		"WSL2 with interop and pro attached does not warn": {
			roots:         []string{"enabled", "version-wsl2"},
			detectVirtCmd: "wsl",
			wslVersionCmd: "regular",
			proStatusCmd:  "attached",
		},
		"WSL2 with interop and pro detached does not warn": {
			roots:         []string{"enabled", "version-wsl2"},
			detectVirtCmd: "wsl",
			wslVersionCmd: "regular",
			proStatusCmd:  "detached",
		},
		"WSL2 garbage version is WSL2": {
			roots:         []string{"enabled", "version-garbage"},
			detectVirtCmd: "wsl",
			wslVersionCmd: "regular",
			proStatusCmd:  "attached",
		},
		"WSL2 empty version is WSL2": {
			roots:         []string{"enabled", "version-empty"},
			detectVirtCmd: "wsl",
			wslVersionCmd: "regular",
			proStatusCmd:  "attached",
		},
		// WSL 2 interop testing
		"WSL2 without interop file does not warn": {
			roots:         []string{"enabled", "version-wsl2"},
			detectVirtCmd: "wsl",
			wslVersionCmd: "error",
			proStatusCmd:  "attached",

			missingFiles: []string{"proc/sys/fs/binfmt_misc/WSLInterop-late"},
		},
		"WSL2 with disabled interop does not warn": {
			roots:         []string{"disabled", "version-wsl2"},
			detectVirtCmd: "wsl",
			wslVersionCmd: "error",
			proStatusCmd:  "attached",
		},
		"WSL2 with late-disabled interop does not warn": {
			roots:         []string{"late-disabled", "version-wsl2"},
			detectVirtCmd: "wsl",
			wslVersionCmd: "error",
			proStatusCmd:  "attached",
		},
		"WSL2 with garbage interop file does not warn": {
			roots:         []string{"garbage", "version-wsl2"},
			detectVirtCmd: "wsl",
			wslVersionCmd: "error",
			proStatusCmd:  "attached",
		},
		"WSL2 with empty interop file does not warn": {
			roots:         []string{"empty", "version-wsl2"},
			detectVirtCmd: "wsl",
			wslVersionCmd: "error",
			proStatusCmd:  "attached",
		},
		// WSL 2 Warning cases
		"WSL2 duplicate parsed versions warns": {
			roots:         []string{"enabled", "version-wsl2"},
			detectVirtCmd: "wsl",
			wslVersionCmd: "duplicate versions",
			proStatusCmd:  "attached",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 2,
			},
		},
		"WSL2 empty version return warns": {
			roots:         []string{"enabled", "version-wsl2"},
			detectVirtCmd: "wsl",
			wslVersionCmd: "empty version",
			proStatusCmd:  "attached",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 2,
			},
		},
		"WSL2 all cmd empty return warns": {
			roots:         []string{"enabled", "version-wsl2"},
			detectVirtCmd: "wsl",
			wslVersionCmd: "",
			proStatusCmd:  "",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 3,
			},
		},
		"WSL2 garbage return from commands warns": {
			roots:         []string{"enabled", "version-wsl2"},
			detectVirtCmd: "wsl",
			wslVersionCmd: "garbage",
			proStatusCmd:  "garbage",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 3,
			},
		},
		"WSL2 cmd errors warns": {
			roots:         []string{"enabled", "version-wsl2"},
			detectVirtCmd: "wsl",
			wslVersionCmd: "error",
			proStatusCmd:  "error",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 2,
			},
		},
		"WSL2 cmd errors no exit warns": {
			roots:         []string{"enabled", "version-wsl2"},
			detectVirtCmd: "wsl",
			wslVersionCmd: "error no exit",
			proStatusCmd:  "error no exit",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 3,
				slog.LevelInfo: 2,
			},
		},
		"WSL2 missing WSL version is WSL2 but warns": {
			roots:         []string{"enabled"},
			detectVirtCmd: "wsl",
			wslVersionCmd: "regular",
			proStatusCmd:  "attached",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 2,
			},
		},

		// WSL 1
		"WSL1 with interop and pro attached does not warn": {
			roots:         []string{"enabled", "version-wsl1"},
			detectVirtCmd: "wsl",
			wslVersionCmd: "regular",
			proStatusCmd:  "attached",

			missingFiles: []string{"proc/sys/fs/binfmt_misc/WSLInterop-late"},
		},
		"WSL1 with interop and pro detached does not warn": {
			roots:         []string{"enabled", "version-wsl1"},
			detectVirtCmd: "wsl",
			wslVersionCmd: "regular",
			proStatusCmd:  "detached",
		},
		// WSL 1 interop testing
		"WSL1 without interop file does not warn": {
			roots:         []string{"enabled", "version-wsl2"},
			detectVirtCmd: "wsl",
			wslVersionCmd: "error",
			proStatusCmd:  "attached",

			missingFiles: []string{"proc/sys/fs/binfmt_misc/WSLInterop-late", "proc/sys/fs/binfmt_misc/WSLInterop"},
		},
		"WSL1 with disabled interop does not warn": {
			roots:         []string{"disabled", "version-wsl1"},
			detectVirtCmd: "wsl",
			wslVersionCmd: "error",
			proStatusCmd:  "attached",
		},
		"WSL1 with late-disabled interop does not warn": {
			roots:         []string{"late-disabled", "version-wsl1"},
			detectVirtCmd: "wsl",
			wslVersionCmd: "regular",
			proStatusCmd:  "attached",
		},
		"WSL1 with garbage interop file does not warn": {
			roots:         []string{"garbage", "version-wsl1"},
			detectVirtCmd: "wsl",
			wslVersionCmd: "error",
			proStatusCmd:  "attached",
		},
		"WSL1 with empty interop file does not warn": {
			roots:         []string{"empty", "version-wsl1"},
			detectVirtCmd: "wsl",
			wslVersionCmd: "error",
			proStatusCmd:  "attached",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tmp := t.TempDir()
			for _, r := range tc.roots {
				err := testutils.CopyDir(t, filepath.Join("testdata/linuxfs", r), tmp)
				require.NoError(t, err, "setup: failed to copy test data directory: ")
			}

			for _, f := range tc.missingFiles {
				err := os.Remove(filepath.Join(tmp, f))
				require.NoError(t, err, "setup: failed to remove file %s: ", f)
			}

			l := testutils.NewMockHandler(slog.LevelDebug)
			options := []platform.Options{
				platform.WithRoot(tmp),
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
	case "error no exit":
		fmt.Fprintf(os.Stderr, "Error requested in fake systemd-detect-virt")
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
	case "error no exit":
		file = os.Stderr
		str = "Error requested in fake wsl.exe --version"
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
	case "duplicate versions":
		str = `
WSL version: 2.4.11.0
Kernel version: 5.15.167.4-1
WSL version: 2.4.11.0
Kernel version: 5.15.167.4-1
WSLg version: 1.0.65
MSRDC version: 1.2.5716
Direct3D version: 1.611.1-81528511
DXCore version: 10.0.26100.1-240331-1435.ge-release
Windows version: 10.0.26100.3194`
	case "empty version":
		str = `
WSL version:
Kernel version:
WSLg version: 1.0.65
MSRDC version: 1.2.5716
Direct3D version: 1.611.1-81528511
DXCore version: 10.0.26100.1-240331-1435.ge-release
Windows version: 10.0.26100.3194`
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
	case "error no exit":
		fmt.Fprintf(os.Stderr, "Error requested in fake pro api")
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
