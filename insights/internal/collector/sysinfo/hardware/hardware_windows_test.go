package hardware_test

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/common/testutils"
	"github.com/ubuntu/ubuntu-insights/insights/internal/collector/sysinfo/hardware"
	"github.com/ubuntu/ubuntu-insights/insights/internal/collector/sysinfo/platform"
	"go.yaml.in/yaml/v3"
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
		productInfo       string
		cpuInfo           string
		gpuInfo           string
		memoryInfo        string
		diskInfo          string
		partitionInfo     string
		screenResInfo     string
		screenPhysResInfo string
		screenSizeInfo    string

		logs    map[slog.Level]uint
		wantErr bool
	}{
		"Regular hardware information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:     "regular",
			screenPhysResInfo: "regular",
			screenSizeInfo:    "regular",

			logs: map[slog.Level]uint{
				slog.LevelInfo: 3,
			},
		},

		"Missing product information": {
			productInfo:   "missing",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:     "regular",
			screenPhysResInfo: "regular",
			screenSizeInfo:    "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
				slog.LevelInfo: 3,
			},
		},

		"Error product information": {
			productInfo:   "error",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:     "regular",
			screenPhysResInfo: "regular",
			screenSizeInfo:    "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
				slog.LevelInfo: 3,
			},
		},

		"Missing CPU information": {
			productInfo:   "regular",
			cpuInfo:       "missing",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:     "regular",
			screenPhysResInfo: "regular",
			screenSizeInfo:    "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
				slog.LevelInfo: 3,
			},
		},

		"Negative valued CPU information": {
			productInfo:   "regular",
			cpuInfo:       "negative",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:     "regular",
			screenPhysResInfo: "regular",
			screenSizeInfo:    "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 2,
				slog.LevelInfo: 3,
			},
		},

		"Zero valued CPU information": {
			productInfo:   "regular",
			cpuInfo:       "zero",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:     "regular",
			screenPhysResInfo: "regular",
			screenSizeInfo:    "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
				slog.LevelInfo: 3,
			},
		},

		"Error CPU information": {
			productInfo:   "regular",
			cpuInfo:       "error",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:     "regular",
			screenPhysResInfo: "regular",
			screenSizeInfo:    "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
				slog.LevelInfo: 3,
			},
		},

		"Missing GPU information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "missing",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:     "regular",
			screenPhysResInfo: "regular",
			screenSizeInfo:    "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
				slog.LevelInfo: 3,
			},
		},

		"Error GPU information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "error",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:     "regular",
			screenPhysResInfo: "regular",
			screenSizeInfo:    "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
				slog.LevelInfo: 3,
			},
		},

		"Missing memory information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "missing",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:     "regular",
			screenPhysResInfo: "regular",
			screenSizeInfo:    "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
				slog.LevelInfo: 3,
			},
		},

		"Negative memory information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "negative",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:     "regular",
			screenPhysResInfo: "regular",
			screenSizeInfo:    "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
				slog.LevelInfo: 3,
			},
		},

		"Bad memory information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "bad",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:     "regular",
			screenPhysResInfo: "regular",
			screenSizeInfo:    "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
				slog.LevelInfo: 3,
			},
		},

		"Garbage memory information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "garbage",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:     "regular",
			screenPhysResInfo: "regular",
			screenSizeInfo:    "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
				slog.LevelInfo: 3,
			},
		},

		"Error memory information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "error",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:     "regular",
			screenPhysResInfo: "regular",
			screenSizeInfo:    "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
				slog.LevelInfo: 3,
			},
		},

		"Single disk and part": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "single",
			partitionInfo: "single",

			screenResInfo:     "regular",
			screenPhysResInfo: "regular",
			screenSizeInfo:    "regular",

			logs: map[slog.Level]uint{
				slog.LevelInfo: 1,
			},
		},

		"Ignores virtual disks": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "virtual-disk",
			partitionInfo: "single",

			screenResInfo:     "regular",
			screenPhysResInfo: "regular",
			screenSizeInfo:    "regular",

			logs: map[slog.Level]uint{
				slog.LevelInfo: 2,
			},
		},

		"Missing disk information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "missing",
			partitionInfo: "regular",

			screenResInfo:     "regular",
			screenPhysResInfo: "regular",
			screenSizeInfo:    "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 2,
				slog.LevelInfo: 1,
			},
		},

		"Error disk information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "error",
			partitionInfo: "regular",

			screenResInfo:     "regular",
			screenPhysResInfo: "regular",
			screenSizeInfo:    "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 2,
				slog.LevelInfo: 1,
			},
		},

		"Malicious disk information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "malicious",
			partitionInfo: "regular",

			screenResInfo:     "regular",
			screenPhysResInfo: "regular",
			screenSizeInfo:    "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 2,
				slog.LevelInfo: 1,
			},
		},

		"Missing partition information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "missing",

			screenResInfo:     "regular",
			screenPhysResInfo: "regular",
			screenSizeInfo:    "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 2,
				slog.LevelInfo: 2,
			},
		},

		"Error partition information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "error",

			screenResInfo:     "regular",
			screenPhysResInfo: "regular",
			screenSizeInfo:    "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 2,
				slog.LevelInfo: 2,
			},
		},

		"Malicious partition information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "malicious",

			screenResInfo:     "regular",
			screenPhysResInfo: "regular",
			screenSizeInfo:    "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 2,
				slog.LevelInfo: 2,
			},
		},

		"Improper disks index and partition information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "improper disks",
			partitionInfo: "regular",

			screenResInfo:     "regular",
			screenPhysResInfo: "regular",
			screenSizeInfo:    "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 10,
				slog.LevelInfo: 2,
			},
		},

		"Error no exit disks information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "error no exit",
			partitionInfo: "regular",

			screenResInfo:     "regular",
			screenPhysResInfo: "regular",
			screenSizeInfo:    "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 2,
				slog.LevelInfo: 2,
			},
		},

		"Error no exit partition information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "error no exit",

			screenResInfo:     "regular",
			screenPhysResInfo: "regular",
			screenSizeInfo:    "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 2,
				slog.LevelInfo: 3,
			},
		},

		"Missing screen resolution information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:     "missing",
			screenPhysResInfo: "regular",
			screenSizeInfo:    "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
				slog.LevelInfo: 4,
			},
		},

		// Screens
		"Single screen": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:     "single",
			screenPhysResInfo: "single",
			screenSizeInfo:    "single",

			logs: map[slog.Level]uint{
				slog.LevelInfo: 3,
			},
		},

		"Missing Screens": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:     "missing",
			screenPhysResInfo: "missing",
			screenSizeInfo:    "missing",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 4,
				slog.LevelInfo: 2,
			},
		},

		"Null Screens": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:     "null",
			screenPhysResInfo: "null",
			screenSizeInfo:    "null",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 4,
				slog.LevelInfo: 5,
			},
		},

		"Partial missing screen resolution information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:     "no resolution",
			screenPhysResInfo: "regular",
			screenSizeInfo:    "regular",

			logs: map[slog.Level]uint{
				slog.LevelInfo: 5,
				slog.LevelWarn: 1,
			},
		},

		"Partial missing screen resolution and size information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:     "no resolution",
			screenPhysResInfo: "regular",
			screenSizeInfo:    "null",

			logs: map[slog.Level]uint{
				slog.LevelInfo: 6,
				slog.LevelWarn: 2,
			},
		},

		"Error screen resolution information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:     "error",
			screenPhysResInfo: "regular",
			screenSizeInfo:    "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
				slog.LevelInfo: 4,
			},
		},

		"Missing screen size information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:     "regular",
			screenPhysResInfo: "regular",
			screenSizeInfo:    "missing",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
				slog.LevelInfo: 3,
			},
		},

		"Non-zero screen display count mismatch": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:     "single",
			screenPhysResInfo: "regular",
			screenSizeInfo:    "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 2,
				slog.LevelInfo: 3,
			},
		},

		"Error screen size information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:     "regular",
			screenPhysResInfo: "regular",
			screenSizeInfo:    "error",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
				slog.LevelInfo: 3,
			},
		},

		"Screen error no exit information": {
			productInfo:       "regular",
			cpuInfo:           "regular",
			gpuInfo:           "regular",
			memoryInfo:        "regular",
			diskInfo:          "regular",
			partitionInfo:     "regular",
			screenResInfo:     "error no exit",
			screenPhysResInfo: "error no exit",
			screenSizeInfo:    "error no exit",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 4,
				slog.LevelInfo: 5,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			options := []hardware.Options{
				hardware.WithArch("amd64"),
			}

			if tc.productInfo != "-" {
				cmdArgs := testutils.SetupFakeCmdArgs("TestFakeProductInfo", tc.productInfo)
				options = append(options, hardware.WithProductInfo(cmdArgs))
			}

			if tc.cpuInfo != "-" {
				cmdArgs := testutils.SetupFakeCmdArgs("TestFakeCPUInfo", tc.cpuInfo)
				options = append(options, hardware.WithCPUInfo(cmdArgs))
			}

			if tc.gpuInfo != "-" {
				cmdArgs := testutils.SetupFakeCmdArgs("TestFakeGPUInfo", tc.gpuInfo)
				options = append(options, hardware.WithGPUInfo(cmdArgs))
			}

			if tc.memoryInfo != "-" {
				cmdArgs := testutils.SetupFakeCmdArgs("TestFakeMemoryInfo", tc.memoryInfo)
				options = append(options, hardware.WithMemoryInfo(cmdArgs))
			}

			if tc.diskInfo != "-" {
				cmdArgs := testutils.SetupFakeCmdArgs("TestFakeDiskInfo", tc.diskInfo)
				options = append(options, hardware.WithDiskInfo(cmdArgs))
			}

			if tc.partitionInfo != "-" {
				cmdArgs := testutils.SetupFakeCmdArgs("TestFakePartitionInfo", tc.partitionInfo)
				options = append(options, hardware.WithPartitionInfo(cmdArgs))
			}

			if tc.screenResInfo != "-" {
				cmdArgs := testutils.SetupFakeCmdArgs("TestFakeScreenResInfo", tc.screenResInfo)
				options = append(options, hardware.WithScreenResInfo(cmdArgs))
			}

			if tc.screenPhysResInfo != "-" {
				cmdArgs := testutils.SetupFakeCmdArgs("TestFakeScreenPhysResInfo", tc.screenPhysResInfo)
				options = append(options, hardware.WithScreenPhysResInfo(cmdArgs))
			}

			if tc.screenSizeInfo != "-" {
				cmdArgs := testutils.SetupFakeCmdArgs("TestFakeDisplaySizeInfo", tc.screenSizeInfo)
				options = append(options, hardware.WithDisplaySizeInfo(cmdArgs))
			}

			l := testutils.NewMockHandler(slog.LevelDebug)
			s := hardware.New(slog.New(&l), options...)

			got, err := s.Collect(platform.Info{})
			if tc.wantErr {
				require.Error(t, err, "Collect should return an error and didn’t")
				return
			}
			require.NoError(t, err, "Collect should not return an error")

			sGot, err := yaml.Marshal(got)
			require.NoError(t, err, "Failed to marshal sysinfo to yaml")
			want := testutils.LoadWithUpdateFromGolden(t, string(sGot))
			assert.Equal(t, want, string(sGot), "Collect should return expected sys information")

			if !l.AssertLevels(t, tc.logs) {
				l.OutputLogs(t)
			}
		})
	}
}

func TestFakeProductInfo(_ *testing.T) {
	args, err := testutils.GetFakeCmdArgs()
	if err != nil {
		return
	}
	defer os.Exit(0)

	switch args[0] {
	case "error":
		fmt.Fprint(os.Stderr, "Error requested in fake product info")
		os.Exit(1)
	case "regular":
		fmt.Println(`

AdminPasswordStatus         : 3
BootupState                 : Normal boot
ChassisBootupState          : 3
KeyboardPasswordStatus      : 3
PowerOnPasswordStatus       : 3
PowerSupplyState            : 3
PowerState                  : 0
FrontPanelResetStatus       : 3
ThermalState                : 3
Status                      : OK
Name                        : MSI
PowerManagementCapabilities :
PowerManagementSupported    :
Caption                     : MSI
Description                 : AT/AT COMPATIBLE
InstallDate                 :
CreationClassName           : Win32_ComputerSystem
NameFormat                  :
PrimaryOwnerContact         :
PrimaryOwnerName            : johndoe@internet.org
Roles                       : {LM_Workstation, LM_Server, NT}
InitialLoadInfo             :
LastLoadInfo                :
ResetCapability             : 1
AutomaticManagedPagefile    : True
AutomaticResetBootOption    : True
AutomaticResetCapability    : True
BootOptionOnLimit           :
BootOptionOnWatchDog        :
BootROMSupported            : True
BootStatus                  : {0, 0, 0, 0...}
ChassisSKUNumber            : Default string
CurrentTimeZone             : -300
DaylightInEffect            : False
DNSHostName                 : MSI
Domain                      : WORKGROUP
DomainRole                  : 0
EnableDaylightSavingsTime   : True
HypervisorPresent           : True
InfraredSupported           : False
Manufacturer                : Micro-Star International Co., Ltd.
Model                       : Star 11 CPP
NetworkServerModeEnabled    : True
NumberOfLogicalProcessors   : 16
NumberOfProcessors          : 1
OEMLogoBitmap               :
OEMStringArray              : { , $BIOSE1110000100000000200,  ,  ...}
PartOfDomain                : False
PauseAfterReset             : -1
PCSystemType                : 2
PCSystemTypeEx              : 2
ResetCount                  : -1
ResetLimit                  : -1
SupportContactDescription   :
SystemFamily                : GF
SystemSKUNumber             : 1582.3
SystemStartupDelay          :
SystemStartupOptions        :
SystemStartupSetting        :
SystemType                  : x64-based PC
TotalPhysicalMemory         : 68406489088
UserName                    : MSI\johndo
WakeUpType                  : 6
Workgroup                   : WORKGROUP`)
	case "":
		fallthrough
	case "missing":
		os.Exit(0)
	}
}

func TestFakeCPUInfo(_ *testing.T) {
	args, err := testutils.GetFakeCmdArgs()
	if err != nil {
		return
	}
	defer os.Exit(0)

	switch args[0] {
	case "error":
		fmt.Fprint(os.Stderr, "Error requested in fake cpu info")
		os.Exit(1)
	case "regular":
		fmt.Println(`

Availability                            : 3
CpuStatus                               : 1
CurrentVoltage                          : 8
DeviceID                                : CPU0
ErrorCleared                            :
ErrorDescription                        :
LastErrorCode                           :
LoadPercentage                          : 4
Status                                  : OK
StatusInfo                              : 3
AddressWidth                            : 64
DataWidth                               : 64
ExtClock                                : 100
L2CacheSize                             : 10240
L2CacheSpeed                            :
MaxClockSpeed                           : 2304
PowerManagementSupported                : False
ProcessorType                           : 3
Revision                                :
SocketDesignation                       : U3E1
Version                                 :
VoltageCaps                             :
Caption                                 : Intel64 Family 6 Model 141 Stepping 1
Description                             : Intel64 Family 6 Model 141 Stepping 1
InstallDate                             :
Name                                    : 11th Gen Intel(R) Core(TM) i7-11800H @ 2.30GHz
ConfigManagerErrorCode                  :
ConfigManagerUserConfig                 :
CreationClassName                       : Win32_Processor
PNPDeviceID                             :
PowerManagementCapabilities             :
SystemCreationClassName                 : Win32_ComputerSystem
SystemName                              : MSI
CurrentClockSpeed                       : 2304
Family                                  : 198
OtherFamilyDescription                  :
Role                                    : CPU
Stepping                                :
UniqueId                                :
UpgradeMethod                           : 1
Architecture                            : 9
AssetTag                                : To Be Filled By O.E.M.
Characteristics                         : 252
L3CacheSize                             : 24576
L3CacheSpeed                            : 0
Level                                   : 6
Manufacturer                            : GenuineIntel
NumberOfCores                           : 8
NumberOfEnabledCore                     : 8
NumberOfLogicalProcessors               : 16
PartNumber                              : To Be Filled By O.E.M.
ProcessorId                             : BFEBFBFF000806D1
SecondLevelAddressTranslationExtensions : False
SerialNumber                            : To Be Filled By O.E.M.
ThreadCount                             : 16
VirtualizationFirmwareEnabled           : False
VMMonitorModeExtensions                 : False`)
	case "negative":
		fmt.Println(`

Name                                    : 11th Gen Intel(R) Core(TM) i7-11800H @ -2.30GHz
Manufacturer                            : GenuineIntel
NumberOfCores                           : -8
NumberOfLogicalProcessors               : -16`)
	case "zero":
		fmt.Println(`

Name                                    : 11th Gen Intel(R) Core(TM) i7-11800H @ 2.30GHz
Manufacturer                            : GenuineIntel
NumberOfCores                           : 0
NumberOfLogicalProcessors               : 0`)
	case "":
		fallthrough
	case "missing":
		os.Exit(0)
	}
}

func TestFakeGPUInfo(_ *testing.T) {
	args, err := testutils.GetFakeCmdArgs()
	if err != nil {
		return
	}
	defer os.Exit(0)

	switch args[0] {
	case "error":
		fmt.Fprint(os.Stderr, "Error requested in fake gpu info")
		os.Exit(1)
	case "regular":
		fmt.Println(`

AcceleratorCapabilities      :
AdapterCompatibility         : NVIDIA
AdapterDACType               : Integrated RAMDAC
AdapterRAM                   : 4293918720
Availability                 : 8
CapabilityDescriptions       :
Caption                      : NVIDIA GeForce RTX 3050 Ti Laptop GPU
ColorTableEntries            :
ConfigManagerErrorCode       : 0
ConfigManagerUserConfig      : False
CreationClassName            : Win32_VideoController
CurrentBitsPerPixel          :
CurrentHorizontalResolution  :
CurrentNumberOfColors        :
CurrentNumberOfColumns       :
CurrentNumberOfRows          :
CurrentRefreshRate           :
CurrentScanMode              :
CurrentVerticalResolution    :
Description                  : NVIDIA GeForce RTX 3050 Ti Laptop GPU
DeviceID                     : VideoController1
DeviceSpecificPens           :
DitherType                   :
DriverDate                   : 20240926000000.000000-000
DriverVersion                : 32.0.15.6590
ErrorCleared                 :
ErrorDescription             :
ICMIntent                    :
ICMMethod                    :
InfFilename                  : oem316.inf
InfSection                   : Section181
InstallDate                  :
InstalledDisplayDrivers      : C:\Windows\System32\DriverStore\FileRepository\nvmii.inf_amd64_5b4deda8605cff46\nvldumdx.dll,C:\Windows\System32\DriverStore\FileRepository\nvmii.inf_amd64_5b4deda8605cff46\nvldumdx.dll,C
                               :\Windows\System32\DriverStore\FileRepository\nvmii.inf_amd64_5b4deda8605cff46\nvldumdx.dll,C:\Windows\System32\DriverStore\FileRepository\nvmii.inf_amd64_5b4deda8605cff46\nvldumdx.dll
LastErrorCode                :
MaxMemorySupported           :
MaxNumberControlled          :
MaxRefreshRate               :
MinRefreshRate               :
Monochrome                   : False
Name                         : NVIDIA GeForce RTX 3050 Ti Laptop GPU
NumberOfColorPlanes          :
NumberOfVideoPages           :
PNPDeviceID                  : PCI\VEN_10DE&DEV_25A0&SUBSYS_12EC1462&REV_A1\4&36DBEA65&0&0008
PowerManagementCapabilities  :
PowerManagementSupported     :
ProtocolSupported            :
ReservedSystemPaletteEntries :
SpecificationVersion         :
Status                       : OK
StatusInfo                   :
SystemCreationClassName      : Win32_ComputerSystem
SystemName                   : MSI
SystemPaletteEntries         :
TimeOfLastReset              :
VideoArchitecture            : 5
VideoMemoryType              : 2
VideoMode                    :
VideoModeDescription         :
VideoProcessor               : NVIDIA GeForce RTX 3050 Ti Laptop GPU

AcceleratorCapabilities      :
AdapterCompatibility         : Intel Corporation
AdapterDACType               : Internal
AdapterRAM                   : 1073741824
Availability                 : 3
CapabilityDescriptions       :
Caption                      : Intel(R) UHD Graphics
ColorTableEntries            :
ConfigManagerErrorCode       : 0
ConfigManagerUserConfig      : False
CreationClassName            : Win32_VideoController
CurrentBitsPerPixel          : 32
CurrentHorizontalResolution  : 1920
CurrentNumberOfColors        : 4294967296
CurrentNumberOfColumns       : 0
CurrentNumberOfRows          : 0
CurrentRefreshRate           : 144
CurrentScanMode              : 4
CurrentVerticalResolution    : 1080
Description                  : Intel(R) UHD Graphics
DeviceID                     : VideoController2
DeviceSpecificPens           :
DitherType                   : 0
DriverDate                   : 20220526000000.000000-000
DriverVersion                : 30.0.101.2079
ErrorCleared                 :
ErrorDescription             :
ICMIntent                    :
ICMMethod                    :
InfFilename                  : oem230.inf
InfSection                   : iTGLD_w10_DS
InstallDate                  :
InstalledDisplayDrivers      : C:\Windows\System32\DriverStore\FileRepository\iigd_dch.inf_amd64_357acc06f2c40efb\igdumdim0.dll,C:\Windows\System32\DriverStore\FileRepository\iigd_dch.inf_amd64_357acc06f2c40efb\igd10i
                               umd32.dll,C:\Windows\System32\DriverStore\FileRepository\iigd_dch.inf_amd64_357acc06f2c40efb\igd10iumd128.dll,C:\Windows\System32\DriverStore\FileRepository\iigd_dch.inf_amd64_357acc06f2c4
                               0efb\igd12umd64.dll
LastErrorCode                :
MaxMemorySupported           :
MaxNumberControlled          :
MaxRefreshRate               : 144
MinRefreshRate               : 60
Monochrome                   : False
Name                         : Intel(R) UHD Graphics
NumberOfColorPlanes          :
NumberOfVideoPages           :
PNPDeviceID                  : PCI\VEN_8086&DEV_9A60&SUBSYS_12EC1462&REV_01\3&11583659&2&10
PowerManagementCapabilities  :
PowerManagementSupported     :
ProtocolSupported            :
ReservedSystemPaletteEntries :
SpecificationVersion         :
Status                       : OK
StatusInfo                   :
SystemCreationClassName      : Win32_ComputerSystem
SystemName                   : MSI
SystemPaletteEntries         :
TimeOfLastReset              :
VideoArchitecture            : 5
VideoMemoryType              : 2
VideoMode                    :
VideoModeDescription         : 1920 x 1080 x 4294967296 colors
VideoProcessor               : Intel(R) UHD Graphics Family`)
	case "":
		fallthrough
	case "missing":
		os.Exit(0)
	}
}

func TestFakeMemoryInfo(_ *testing.T) {
	args, err := testutils.GetFakeCmdArgs()
	if err != nil {
		return
	}
	defer os.Exit(0)

	switch args[0] {
	case "error":
		fmt.Fprint(os.Stderr, "Error requested in fake memory info")
		os.Exit(1)
	case "regular":
		fmt.Println(`

TotalPhysicalMemory : 68406489088`)
	case "negative":
		fmt.Println(`

TotalPhysicalMemory : -68406489088`)
	case "bad":
		fmt.Println(`

TotalPhysicalMemory : ONE BILLION!!!`)
	case "garbage":
		fmt.Println(`
TLB:active
Memory:mapped
Pages:paged`)
	case "":
		fallthrough
	case "missing":
		os.Exit(0)
	}
}

func TestFakeDiskInfo(_ *testing.T) {
	args, err := testutils.GetFakeCmdArgs()
	if err != nil {
		return
	}
	defer os.Exit(0)

	switch args[0] {
	case "error":
		fmt.Fprint(os.Stderr, "Error requested in fake disk info")
		os.Exit(1)
	case "regular":
		fmt.Println(`
[
  {
    "MediaType": "Fixed hard disk media",
    "Index": 3,
    "Size": 1000202273280,
    "Partitions": 3
  },
  {
    "MediaType": "Fixed hard disk media",
    "Index": 4,
    "Size": 536864025600,
    "Partitions": 1
  },
  {
    "MediaType": "Fixed hard disk media",
    "Index": 0,
    "Size": 10000830067200,
    "Partitions": 1
  },
  {
    "MediaType": "Removable Media",
    "Index": 5,
    "Size": 123971420160,
    "Partitions": 1
  },
  {
    "MediaType": "Fixed hard disk media",
    "Index": 2,
    "Size": 1024203640320,
    "Partitions": 2
  },
  {
    "MediaType": "Fixed hard disk media",
    "Index": 1,
    "Size": 1000202273280,
    "Partitions": 1
  }
]`)
	case "single":
		fmt.Println(`
{
		  "MediaType": "Fixed hard disk media",
		  "Index": 0,
		  "Size": 1000202273280,
		  "Partitions": 1
}`)
	case "virtual-disk":
		fmt.Println(`
[
{
		  "MediaType": "Fixed hard disk media",
		  "Index": 0,
		  "Size": 1000202273280,
		  "Partitions": 1
},
{
		"Model": "Microsoft Virtual Disk",
		"MediaType": "Fixed hard disk media",
		"Index": 1,
		"Size": 1000202273280,
		"Partitions": 1
}
]`)
	case "malicious":
		fmt.Println(`
[
  {
    "MediaType": "Junk",
    "Index": 3,
    "Size": 1000202273280,
    "Partitions": 3
  },
  {
    "MediaType": "Fixed hard disk media",
    "Index": 4,
    "Size": -536864025600,
    "Partitions": 1
  },
  {
    "MediaType": "Fixed hard disk media",
    "Index": -1,
    "Size": 10000830067200,
    "Partitions": 1
  },
  {
    "MediaType": "Removable Media",
    "Index": 5,
    "Size": 123971420160,
    "Partitions": 100000000000000000
  },
  {
    "MediaType": "Fixed hard disk media",
    "Index": 2,
    "Size": 102420364032000000000,
    "Partitions": 2
  },
  {
    "MediaType": "Fixed hard disk media",
    "Size": 1000202273280,
    "Partitions": 1
	"UnknownField": "Unknown"
  }
]`)
	case "improper disks":
		fmt.Println(`
[
	{
			"MediaType": "Fixed hard disk media",
			"Index": 0,
			"Size": 1,
			"Partitions": 129
	},
	{
			"MediaType": "Fixed hard disk media",
			"Index": 0,
			"Size": 1,
			"Partitions": 126
	}
]`)
	case "error no exit":
		fmt.Fprint(os.Stderr, "Error requested in fake disk info")
	case "":
		fallthrough
	case "missing":
		os.Exit(0)
	}
}

func TestFakePartitionInfo(_ *testing.T) {
	args, err := testutils.GetFakeCmdArgs()
	if err != nil {
		return
	}
	defer os.Exit(0)

	switch args[0] {
	case "error":
		fmt.Fprint(os.Stderr, "Error requested in fake partition info")
		os.Exit(1)
	case "regular":
		fmt.Println(`
[
  {
    "DiskIndex": 3,
    "Size": 104857600,
    "Type": "GPT: System"
  },
  {
    "DiskIndex": 3,
    "Size": 999275102208,
    "Type": "GPT: Basic Data"
  },
  {
    "DiskIndex": 3,
    "Size": 805306368,
    "Type": "GPT: Unknown"
  },
  {
    "DiskIndex": 4,
    "Size": 536853086208,
    "Type": "GPT: Basic Data"
  },
  {
    "DiskIndex": 0,
    "Size": 10000829251584,
    "Type": "GPT: Basic Data"
  },
  {
    "DiskIndex": 5,
    "Size": 123977334784,
    "Type": "Installable File System"
  },
  {
    "DiskIndex": 2,
    "Size": 1127219200,
    "Type": "GPT: System"
  },
  {
    "DiskIndex": 2,
    "Size": 1023080923136,
    "Type": "GPT: Unknown"
  },
  {
    "DiskIndex": 1,
    "Size": 1000186314752,
    "Type": "GPT: Basic Data"
  }
]`)
	case "single":
		fmt.Println(`
{
		  "DiskIndex": 0,
		  "Size": 104857600,
		  "Type": "GPT: System"
}`)
	case "malicious":
		fmt.Println(`
[
  {
    "DiskIndex": -3,
    "Size": 104857600,
    "Type": "GPT: System"
  },
  {
    "DiskIndex": 10000,
    "Size": 999275102208,
    "Type": "GPT: Basic Data"
  },
  {
    "DiskIndex": 0,
    "Size": 805306368,
    "Type": "GPT: Unknown"
  },
  {
    "DiskIndex": 0,
    "Size": 536853086208,
    "Type": "GPT: Basic Data"
  },
  {
    "DiskIndex": 2,
    "Size": 10000829251584,
    "Type": "GPT: Basic Data"
  },
  {
    "DiskIndex": 8,
    "Size": 123977334784,
    "Type": "Installable File System"
  },
  {
    "DiskIndex": -2,
    "Size": 1127219200,
    "Type": "GPT: System"
  },
  {
    "DiskIndex": 2,
    "Size": 1023080923136,
    "Type": "GPT: Unknown"
  },
  {
    "Size": 1000186314752,
    "Type": "GPT: Basic Data"
  }
]`)
	case "error no exit":
		fmt.Fprint(os.Stderr, "Error requested in fake partition info")
	case "":
		fallthrough
	case "missing":
		os.Exit(0)
	}
}

func TestFakeScreenResInfo(_ *testing.T) {
	args, err := testutils.GetFakeCmdArgs()
	if err != nil {
		return
	}
	defer os.Exit(0)

	switch args[0] {
	case "error":
		fmt.Fprint(os.Stderr, "Error requested in fake screen resolution info")
		os.Exit(1)
	case "regular":
		fmt.Println(`
[
  {
    "Bounds": {
      "Location": {
        "IsEmpty": false,
        "X": 2560,
        "Y": 0
      },
      "Size": {
        "IsEmpty": false,
        "Width": 1280,
        "Height": 800
      },
      "X": 2560,
      "Y": 0,
      "Width": 1280,
      "Height": 800,
      "Left": 2560,
      "Top": 0,
      "Right": 3840,
      "Bottom": 800,
      "IsEmpty": false
    }
  },
  {
    "Bounds": {
      "Location": {
        "IsEmpty": true,
        "X": 0,
        "Y": 0
      },
      "Size": {
        "IsEmpty": false,
        "Width": 2560,
        "Height": 1440
      },
      "X": 0,
      "Y": 0,
      "Width": 2560,
      "Height": 1440,
      "Left": 0,
      "Top": 0,
      "Right": 2560,
      "Bottom": 1440,
      "IsEmpty": false
    }
  }
]`)
	case "single":
		fmt.Println(`
{
  "Bounds": {
    "Location": {
      "IsEmpty": true,
      "X": 0,
      "Y": 0
    },
    "Size": {
      "IsEmpty": false,
      "Width": 2560,
      "Height": 1440
    },
    "X": 0,
    "Y": 0,
    "Width": 2560,
    "Height": 1440,
    "Left": 0,
    "Top": 0,
    "Right": 2560,
    "Bottom": 1440,
    "IsEmpty": false
  }
}`)

	case "no resolution":
		fmt.Println(`
{
  "Bounds": {
    "Location": {
      "IsEmpty": true,
      "X": 0,
      "Y": 0
    },
    "Size": {
      "IsEmpty": false,
      "Width": 2560,
      "Height": 1440
    },
    "X": 0,
    "Y": 0,
    "Left": 0,
    "Top": 0,
    "Right": 2560,
    "Bottom": 1440,
    "IsEmpty": false
  }
}`)
	case "null":
		fmt.Println(`
{
	"Bounds": null
}`)
	case "error no exit":
		fmt.Fprint(os.Stderr, "Error requested in fake screen resolution info")
	case "":
		fallthrough
	case "missing":
		os.Exit(0)
	}
}

func TestFakeScreenPhysResInfo(_ *testing.T) {
	args, err := testutils.GetFakeCmdArgs()
	if err != nil {
		return
	}
	defer os.Exit(0)

	switch args[0] {
	case "error":
		fmt.Fprint(os.Stderr, "Error requested in fake screen physical size info")
		os.Exit(1)
	case "regular":
		fmt.Println(`
[
  {
    "ScreenWidth": null,
    "ScreenHeight": null
  },
  {
    "ScreenWidth": 2560,
    "ScreenHeight": 1600
  },
  {
    "ScreenWidth": 2560,
    "ScreenHeight": 1440
  }
]`)
	case "single":
		fmt.Println(`
[
{
    "ScreenWidth": null,
    "ScreenHeight": null
  },
  {
	"ScreenWidth": 2560,
	"ScreenHeight": 1440
}
]`)
	case "true single":
		fmt.Println(`
{
	"ScreenWidth": 2560,
	"ScreenHeight": 1440
}`)
	case "null":
		fmt.Println(`
{
	"ScreenWidth": null,
	"ScreenHeight": null
}`)
	case "error no exit":
		fmt.Fprint(os.Stderr, "Error requested in fake screen physical size info")
	case "":
		fallthrough
	case "missing":
		os.Exit(0)
	}
}

func TestFakeDisplaySizeInfo(_ *testing.T) {
	args, err := testutils.GetFakeCmdArgs()
	if err != nil {
		return
	}
	defer os.Exit(0)

	switch args[0] {
	case "error":
		fmt.Fprint(os.Stderr, "Error requested in fake screen size info")
		os.Exit(1)
	case "regular":
		fmt.Println(`
[
  {
    "MaxHorizontalImageSize": 30,
    "MaxVerticalImageSize": 19
  },
  {
    "MaxHorizontalImageSize": 60,
    "MaxVerticalImageSize": 34
  }
]`)
	case "single":
		fmt.Println(`
{
	"MaxHorizontalImageSize": 30,
	"MaxVerticalImageSize": 19
}`)
	case "null":
		fmt.Println(`
{
	"MaxHorizontalImageSize": null,
	"MaxVerticalImageSize": null
}`)
	case "error no exit":
		fmt.Fprint(os.Stderr, "Error requested in fake screen size info")
	case "":
		fallthrough
	case "missing":
		os.Exit(0)
	}
}
