package platform_test

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf16"

	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/common/testutils"
	"github.com/ubuntu/ubuntu-insights/insights/internal/collector/sysinfo/platform"
)

func TestNewLinux(t *testing.T) {
	t.Parallel()

	s := platform.New(slog.Default(), platform.WithRoot("/myspecialroot"))
	require.NotEmpty(t, s, "platform sysinfo Collector has custom fields")
}

func TestCollectLinux(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		roots             []string
		detectVirtCmd     string
		systemdAnalyzeCmd string
		wslVersionCmd     string
		proStatusCmd      string
		env               map[string]string

		missingFiles []string

		logs    map[slog.Level]uint
		wantErr bool
	}{
		// Non-WSL
		"Non-WSL Basic with Pro Attached": {
			detectVirtCmd:     "none",
			systemdAnalyzeCmd: "regular",
			wslVersionCmd:     "error",
			proStatusCmd:      "attached",
			env: map[string]string{
				"XDG_CURRENT_DESKTOP": "ubuntu:GNOME",
				"XDG_SESSION_DESKTOP": "ubuntu",
				"XDG_SESSION_TYPE":    "wayland"},
		},
		"Non-WSL Basic with Pro Detached": {
			detectVirtCmd:     "none",
			systemdAnalyzeCmd: "regular",
			wslVersionCmd:     "error",
			proStatusCmd:      "detached",
			env: map[string]string{
				"XDG_CURRENT_DESKTOP": "ubuntu:GNOME",
				"XDG_SESSION_DESKTOP": "ubuntu",
				"XDG_SESSION_TYPE":    "wayland"},
		},
		"Non-WSL Garbage Returns from Commands warns": {
			detectVirtCmd:     "garbage",
			systemdAnalyzeCmd: "garbage",
			wslVersionCmd:     "garbage",
			proStatusCmd:      "garbage",
			env: map[string]string{
				"XDG_CURRENT_DESKTOP": "ubuntu:GNOME",
				"XDG_SESSION_DESKTOP": "ubuntu",
				"XDG_SESSION_TYPE":    "wayland"},

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},
		"Non-WSL Empty Returns from Commands warns": {
			detectVirtCmd:     "",
			systemdAnalyzeCmd: "",
			wslVersionCmd:     "",
			proStatusCmd:      "",
			env: map[string]string{
				"XDG_CURRENT_DESKTOP": "ubuntu:GNOME",
				"XDG_SESSION_DESKTOP": "ubuntu",
				"XDG_SESSION_TYPE":    "wayland"},

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},
		"Non-WSL Error Returns from Commands warns": {
			detectVirtCmd:     "error",
			systemdAnalyzeCmd: "error",
			wslVersionCmd:     "error",
			proStatusCmd:      "error",
			env: map[string]string{
				"XDG_CURRENT_DESKTOP": "ubuntu:GNOME",
				"XDG_SESSION_DESKTOP": "ubuntu",
				"XDG_SESSION_TYPE":    "wayland"},

			logs: map[slog.Level]uint{
				slog.LevelWarn: 2,
			},
		},
		"Non-WSL Print to StdOut without exitcode warns": {
			detectVirtCmd:     "error no exit",
			systemdAnalyzeCmd: "error no exit",
			wslVersionCmd:     "error no exit",
			proStatusCmd:      "error no exit",
			env: map[string]string{
				"XDG_CURRENT_DESKTOP": "ubuntu:GNOME",
				"XDG_SESSION_DESKTOP": "ubuntu",
				"XDG_SESSION_TYPE":    "wayland"},

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
				slog.LevelInfo: 2,
			},
		},
		"Non-WSl with empty deskstop env vars": {
			detectVirtCmd:     "none",
			systemdAnalyzeCmd: "regular",
			wslVersionCmd:     "error",
			proStatusCmd:      "attached",
		},
		"Non-WSL with garbage desktop env vars": {
			detectVirtCmd:     "none",
			systemdAnalyzeCmd: "regular",
			wslVersionCmd:     "error",
			proStatusCmd:      "attached",
			env: map[string]string{
				"XDG_CURRENT_DESKTOP": "❤️",
				"XDG_SESSION_DESKTOP": "(●'◡'●)",
				"XDG_SESSION_TYPE":    "(╯°□°）╯︵ ┻━┻"},
		},

		// Other virt types
		"Other virt type (uml) with Pro Attached": {
			detectVirtCmd:     "uml",
			systemdAnalyzeCmd: "regular",
			wslVersionCmd:     "error",
			proStatusCmd:      "attached",
			env: map[string]string{
				"XDG_CURRENT_DESKTOP": "ubuntu:GNOME",
				"XDG_SESSION_DESKTOP": "ubuntu",
				"XDG_SESSION_TYPE":    "wayland"},
		},

		// WSL 2
		"WSL2 with interop and pro attached does not warn": {
			roots:             []string{"enabled", "version-wsl2"},
			detectVirtCmd:     "wsl",
			systemdAnalyzeCmd: "regular",
			wslVersionCmd:     "regular-en",
			proStatusCmd:      "attached",
		},
		"WSL2 with interop and pro detached does not warn": {
			roots:             []string{"enabled", "version-wsl2"},
			detectVirtCmd:     "wsl",
			systemdAnalyzeCmd: "regular",
			wslVersionCmd:     "regular-en",
			proStatusCmd:      "detached",
		},
		"WSL2 with interop parses version in pr correctly": {
			roots:             []string{"enabled", "version-wsl2"},
			detectVirtCmd:     "wsl",
			systemdAnalyzeCmd: "regular",
			wslVersionCmd:     "regular-pr",
			proStatusCmd:      "attached",
		},
		"WSL2 with interop parses version in zh-cn correctly": {
			roots:             []string{"enabled", "version-wsl2"},
			detectVirtCmd:     "wsl",
			systemdAnalyzeCmd: "regular",
			wslVersionCmd:     "regular-zh-cn",
			proStatusCmd:      "attached",
		},
		"WSL2 with interop parses version in zh-tw correctly": {
			roots:             []string{"enabled", "version-wsl2"},
			detectVirtCmd:     "wsl",
			systemdAnalyzeCmd: "regular",
			wslVersionCmd:     "regular-zh-tw",
			proStatusCmd:      "attached",
		},
		"WSL 2 custom kernel version is WSL2": {
			roots:             []string{"enabled", "version-custom"},
			detectVirtCmd:     "wsl",
			systemdAnalyzeCmd: "regular",
			wslVersionCmd:     "regular-en",
			proStatusCmd:      "attached",
		},
		"WSL2 garbage version is WSL2": {
			roots:             []string{"enabled", "version-garbage"},
			detectVirtCmd:     "wsl",
			systemdAnalyzeCmd: "regular",
			wslVersionCmd:     "regular-en",
			proStatusCmd:      "attached",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},
		"WSL2 empty version is WSL2": {
			roots:             []string{"enabled", "version-empty"},
			detectVirtCmd:     "wsl",
			systemdAnalyzeCmd: "regular",
			wslVersionCmd:     "regular-en",
			proStatusCmd:      "attached",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},
		"WSL2 without systemd parses correctly": {
			roots:             []string{"enabled", "version-wsl2"},
			detectVirtCmd:     "wsl",
			systemdAnalyzeCmd: "not used",
			wslVersionCmd:     "regular-en",
			proStatusCmd:      "attached",
		},
		// WSL 2 interop testing
		"WSL2 without wsl.conf file does not warn": {
			roots:             []string{"enabled", "version-wsl2"},
			detectVirtCmd:     "wsl",
			systemdAnalyzeCmd: "regular",
			wslVersionCmd:     "regular-en",
			proStatusCmd:      "attached",

			missingFiles: []string{"etc/wsl.conf"},
		},
		"WSL2 with disabled interop does not warn": {
			roots:             []string{"disabled", "version-wsl2"},
			detectVirtCmd:     "wsl",
			systemdAnalyzeCmd: "regular",
			wslVersionCmd:     "error",
			proStatusCmd:      "attached",
		},
		"WSL2 with assumed-enabled interop does not warn": {
			roots:             []string{"assumed-enabled", "version-wsl2"},
			detectVirtCmd:     "wsl",
			systemdAnalyzeCmd: "regular",
			wslVersionCmd:     "regular-en",
			proStatusCmd:      "attached",
		},
		"WSL2 with garbage interop file does not warn": {
			roots:             []string{"garbage", "version-wsl2"},
			detectVirtCmd:     "wsl",
			systemdAnalyzeCmd: "regular",
			wslVersionCmd:     "regular-en",
			proStatusCmd:      "attached",
		},
		"WSL2 with empty interop file does not warn": {
			roots:             []string{"empty", "version-wsl2"},
			detectVirtCmd:     "wsl",
			systemdAnalyzeCmd: "regular",
			wslVersionCmd:     "regular-en",
			proStatusCmd:      "attached",
		},
		"WSL2 empty version return warns": {
			roots:             []string{"enabled", "version-wsl2"},
			detectVirtCmd:     "wsl",
			systemdAnalyzeCmd: "regular",
			wslVersionCmd:     "empty version",
			proStatusCmd:      "attached",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},
		"WSL2 all cmd empty return warns": {
			roots:             []string{"enabled", "version-wsl2"},
			detectVirtCmd:     "wsl",
			systemdAnalyzeCmd: "",
			wslVersionCmd:     "",
			proStatusCmd:      "",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 3,
			},
		},
		"WSL2 garbage return from commands warns": {
			roots:             []string{"enabled", "version-wsl2"},
			detectVirtCmd:     "wsl",
			systemdAnalyzeCmd: "garbage",
			wslVersionCmd:     "garbage",
			proStatusCmd:      "garbage",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 2,
			},
		},
		"WSL2 cmd errors warns": {
			roots:             []string{"enabled", "version-wsl2"},
			detectVirtCmd:     "wsl",
			systemdAnalyzeCmd: "error",
			wslVersionCmd:     "error",
			proStatusCmd:      "error",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 3,
			},
		},
		"WSL2 cmd errors no exit warns": {
			roots:             []string{"enabled", "version-wsl2"},
			detectVirtCmd:     "wsl",
			systemdAnalyzeCmd: "error no exit",
			wslVersionCmd:     "error no exit",
			proStatusCmd:      "error no exit",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 3,
				slog.LevelInfo: 3,
			},
		},
		"WSL2 missing WSL version is WSL2 but warns": {
			roots:             []string{"enabled"},
			detectVirtCmd:     "wsl",
			systemdAnalyzeCmd: "regular",
			wslVersionCmd:     "regular-en",
			proStatusCmd:      "attached",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 4,
			},
		},
		"WSL2 ignores desktop env vars": {
			roots:             []string{"enabled", "version-wsl2"},
			detectVirtCmd:     "wsl",
			systemdAnalyzeCmd: "regular",
			wslVersionCmd:     "regular-en",
			proStatusCmd:      "attached",
			env: map[string]string{
				"XDG_CURRENT_DESKTOP": "ubuntu:GNOME",
				"XDG_SESSION_DESKTOP": "ubuntu",
				"XDG_SESSION_TYPE":    "wayland"},
		},

		// WSL 1
		"WSL1 with interop and pro attached does not warn": {
			roots:             []string{"enabled", "version-wsl1"},
			detectVirtCmd:     "wsl",
			systemdAnalyzeCmd: "wsl1",
			wslVersionCmd:     "regular-en",
			proStatusCmd:      "attached",
		},
		"WSL1 with interop and pro detached does not warn": {
			roots:             []string{"enabled", "version-wsl1"},
			detectVirtCmd:     "wsl",
			systemdAnalyzeCmd: "wsl1",
			wslVersionCmd:     "regular-en",
			proStatusCmd:      "detached",
		},
		// WSL 1 interop testing
		"WSL1 without wsl.conf file does not warn": {
			roots:             []string{"enabled", "version-wsl2"},
			detectVirtCmd:     "wsl",
			systemdAnalyzeCmd: "wsl1",
			wslVersionCmd:     "regular-en",
			proStatusCmd:      "attached",

			missingFiles: []string{"etc/wsl.conf"},
		},
		"WSL1 with disabled interop does not warn": {
			roots:             []string{"disabled", "version-wsl1"},
			detectVirtCmd:     "wsl",
			systemdAnalyzeCmd: "wsl1",
			wslVersionCmd:     "error",
			proStatusCmd:      "attached",
		},
		"WSL1 with assumed-enabled interop does not warn": {
			roots:             []string{"assumed-enabled", "version-wsl1"},
			detectVirtCmd:     "wsl",
			systemdAnalyzeCmd: "wsl1",
			wslVersionCmd:     "regular-en",
			proStatusCmd:      "attached",
		},
		"WSL1 with garbage interop file does not warn": {
			roots:             []string{"garbage", "version-wsl1"},
			detectVirtCmd:     "wsl",
			systemdAnalyzeCmd: "wsl1",
			wslVersionCmd:     "regular-en",
			proStatusCmd:      "attached",
		},
		"WSL1 with empty interop file does not warn": {
			roots:             []string{"empty", "version-wsl1"},
			detectVirtCmd:     "wsl",
			systemdAnalyzeCmd: "wsl1",
			wslVersionCmd:     "regular-en",
			proStatusCmd:      "attached",
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
				platform.WithGetenv(tc.env),
			}

			if tc.detectVirtCmd != "-" {
				cmdArgs := testutils.SetupFakeCmdArgs("TestFakeVirtInfo", tc.detectVirtCmd)
				options = append(options, platform.WithDetectVirtCmd(cmdArgs))
			}

			if tc.systemdAnalyzeCmd != "-" {
				cmdArgs := testutils.SetupFakeCmdArgs("TestFakeSystemdAnalyze", tc.systemdAnalyzeCmd)
				options = append(options, platform.WithSystemdAnalyzeCmd(cmdArgs))
			}

			if tc.wslVersionCmd != "-" {
				cmdArgs := testutils.SetupFakeCmdArgs("TestWSLVersionInfo", tc.wslVersionCmd)
				options = append(options, platform.WithWSLVersionCmd(cmdArgs))
			}

			if tc.proStatusCmd != "-" {
				cmdArgs := testutils.SetupFakeCmdArgs("TestFakeProStatus", tc.proStatusCmd)
				options = append(options, platform.WithProStatusCmd(cmdArgs))
			}

			p := platform.New(slog.New(&l), options...)

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

func TestFakeSystemdAnalyze(*testing.T) {
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
		fmt.Println(`
Startup finished in 296ms (userspace)
multi-user.target reached after 275ms in userspace`)
	case "wsl1":
		fallthrough
	case "not used":
		fmt.Fprintf(os.Stderr, `
System has not been booted with systemd as init system (PID 1). Can't operate.
Failed to connect to bus: Host is down`)
		os.Exit(1)
	case "garbage":
		fmt.Println("garbage 🗑️")
	case "":
		fallthrough
	case "missing":
		os.Exit(0)
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
	case "none":
		fmt.Println("none")
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
	case "error no exit":
		file = os.Stderr
		str = "Error requested in fake wsl.exe --version"
	case "regular-en":
		str = `
WSL version: 2.4.11.0
Kernel version: 5.15.167.4-1
WSLg version: 1.0.65
MSRDC version: 1.2.5716
Direct3D version: 1.611.1-81528511
DXCore version: 10.0.26100.1-240331-1435.ge-release
Windows version: 10.0.26100.3194`
	case "regular-pr":
		str = `
Versão do WSL: 2.1.0.0
Versão do kernel: 5.15.137.3-1
Versão do WSLg: 1.0.59
Versão do MSRDC: 1.2.4677
Versão do Direct3D: 1.611.1-81528511
Versão do DXCore: 10.0.25131.1002-220531-1700.rs-onecore-base2-hyp
Versão do Windows: 10.0.26120.3360`
	case "regular-zh-cn":
		str = `
WSL 版本： 0.67.6.0
内核版本： 5.15.62.1
WSLg 版本： 1.0.44
MSRDC 版本： 1.2.3401
Direct3D 版本： 1.606.4
DXCore 版本： 10.0.25131.1002-220531-1700.rs-onecore-base2-hyp
Windows版本： 10.0.25206.1000`
	case "regular-zh-tw":
		str = `
WSL 版本： 2.4.11.0
核心版本： 5.15.167.4-1
WSLg 版本： 1.0.65
MSRDC 版本： 1.2.5716
Direct3D 版本： 1.611.1-81528511
DXCore 版本： 10.0.26100.1-240331-1435.ge-release
Windows 版本： 10.0.26100.3323`
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
🗑️🗑️🗑️🗑️🗑️
ㄟ( ▔, ▔ )ㄏ`
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
WSLg version:
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
