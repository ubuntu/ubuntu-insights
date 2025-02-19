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

func TestCollectWindows(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		src      software.Source
		tipe     string
		timezone string

		osInfo   string
		language string
		biosInfo string

		logs    map[slog.Level]uint
		wantErr bool
	}{
		"Regular software information": {
			timezone: "EST",

			osInfo:   "regular",
			language: "regular",
			biosInfo: "regular",
		},

		"Missing OS information": {
			timezone: "CEN",

			osInfo:   "",
			language: "regular",
			biosInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Error OS information": {
			timezone: "PST",

			osInfo:   "error",
			language: "regular",
			biosInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Missing language information": {
			timezone: "EST",

			osInfo:   "regular",
			language: "",
			biosInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Error language information": {
			timezone: "EST",

			osInfo:   "regular",
			language: "error",
			biosInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Missing BIOS information": {
			timezone: "JST",

			osInfo:   "regular",
			language: "regular",
			biosInfo: "",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Error BIOS information": {
			timezone: "JST",

			osInfo:   "regular",
			language: "regular",
			biosInfo: "error",

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

			if tc.biosInfo != "-" {
				cmdArgs := testutils.SetupFakeCmdArgs("TestFakeBIOSInfo", tc.biosInfo)
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

Status                                    : OK
Name                                      : Microsoft Windows 10 Home|C:\Windows|\Device\Harddisk0\Partition5
FreePhysicalMemory                        : 50701724
FreeSpaceInPagingFiles                    : 9961472
FreeVirtualMemory                         : 59692028
Caption                                   : Microsoft Windows 10 Home
Description                               :
InstallDate                               : 6/10/2021 10:59:28 AM
CreationClassName                         : Win32_OperatingSystem
CSCreationClassName                       : Win32_ComputerSystem
CSName                                    : MSI
CurrentTimeZone                           : -300
Distributed                               : False
LastBootUpTime                            : 2/13/2025 9:00:00 AM
LocalDateTime                             : 2/13/2025 6:00:00 PM
MaxNumberOfProcesses                      : 4294967295
MaxProcessMemorySize                      : 137438953344
NumberOfLicensedUsers                     :
NumberOfProcesses                         : 252
NumberOfUsers                             : 2
OSType                                    : 18
OtherTypeDescription                      :
SizeStoredInPagingFiles                   : 9961472
TotalSwapSpaceSize                        :
TotalVirtualMemorySize                    : 76764684
TotalVisibleMemorySize                    : 66803212
Version                                   : 10.0.19045
BootDevice                                : \Device\HarddiskVolume1
BuildNumber                               : 19045
BuildType                                 : Multiprocessor Free
CodeSet                                   : 1252
CountryCode                               : 1
CSDVersion                                :
DataExecutionPrevention_32BitApplications : True
DataExecutionPrevention_Available         : True
DataExecutionPrevention_Drivers           : True
DataExecutionPrevention_SupportPolicy     : 2
Debug                                     : False
EncryptionLevel                           : 1024
ForegroundApplicationBoost                : 2
LargeSystemCache                          :
Locale                                    : 0409
Manufacturer                              : Microsoft Corporation
MUILanguages                              : {en-US}
OperatingSystemSKU                        : 101
Organization                              :
OSArchitecture                            : 64-bit
OSLanguage                                : 1033
OSProductSuite                            : 768
PAEEnabled                                :
PlusProductID                             :
PlusVersionNumber                         :
PortableOperatingSystem                   : False
Primary                                   : True
ProductType                               : 1
RegisteredUser                            : user
SerialNumber                              : 12345-54321-98765-BEEFM
ServicePackMajorVersion                   : 0
ServicePackMinorVersion                   : 0
SuiteMask                                 : 784
SystemDevice                              : \Device\HarddiskVolume5
SystemDirectory                           : C:\Windows\system32
SystemDrive                               : C:
WindowsDirectory                          : C:\Windows`)
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
		fmt.Println(`

IetfLanguageTag : en-US`)
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
		fmt.Fprint(os.Stderr, "Error requested in fake bios info")
		os.Exit(1)
	case "regular":
		fmt.Println(`

SMBIOSBIOSVersion : E1582IMS.311
Manufacturer      : American Megatrends International, LLC.
Name              : E1582IMS.311
SerialNumber      : K9999N012345
Version           : MSI_NB - 1061003`)
	case "":
		fallthrough
	case "missing":
		os.Exit(0)
	}
}
