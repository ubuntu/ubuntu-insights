package hardware_test

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/hardware"
	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/platform"
	"github.com/ubuntu/ubuntu-insights/internal/testutils"
	"gopkg.in/yaml.v3"
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
		productInfo    string
		cpuInfo        string
		gpuInfo        string
		memoryInfo     string
		diskInfo       string
		partitionInfo  string
		screenResInfo  string
		screenSizeInfo string

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

			screenResInfo:  "regular",
			screenSizeInfo: "regular",
		},

		"Missing product information": {
			productInfo:   "missing",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:  "regular",
			screenSizeInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Error product information": {
			productInfo:   "error",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:  "regular",
			screenSizeInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Missing CPU information": {
			productInfo:   "regular",
			cpuInfo:       "missing",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:  "regular",
			screenSizeInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Negative valued CPU information": {
			productInfo:   "regular",
			cpuInfo:       "negative",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:  "regular",
			screenSizeInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 2,
			},
		},

		"Zero valued CPU information": {
			productInfo:   "regular",
			cpuInfo:       "zero",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:  "regular",
			screenSizeInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Error CPU information": {
			productInfo:   "regular",
			cpuInfo:       "error",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:  "regular",
			screenSizeInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Missing GPU information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "missing",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:  "regular",
			screenSizeInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Error GPU information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "error",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:  "regular",
			screenSizeInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Missing memory information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "missing",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:  "regular",
			screenSizeInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Negative memory information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "negative",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:  "regular",
			screenSizeInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Bad memory information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "bad",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:  "regular",
			screenSizeInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Garbage memory information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "garbage",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:  "regular",
			screenSizeInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Error memory information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "error",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:  "regular",
			screenSizeInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Missing disk information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "missing",
			partitionInfo: "regular",

			screenResInfo:  "regular",
			screenSizeInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Error disk information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "error",
			partitionInfo: "regular",

			screenResInfo:  "regular",
			screenSizeInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Malicious disk information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "malicious",
			partitionInfo: "regular",

			screenResInfo:  "regular",
			screenSizeInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 12,
			},
		},

		"Missing partition information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "missing",

			screenResInfo:  "regular",
			screenSizeInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Error partition information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "error",

			screenResInfo:  "regular",
			screenSizeInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Malicious partition information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "malicious",

			screenResInfo:  "regular",
			screenSizeInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 9,
			},
		},

		"Missing screen resolution information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:  "missing",
			screenSizeInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Error screen resolution information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:  "error",
			screenSizeInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Missing screen size information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:  "regular",
			screenSizeInfo: "missing",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Error screen size information": {
			productInfo:   "regular",
			cpuInfo:       "regular",
			gpuInfo:       "regular",
			memoryInfo:    "regular",
			diskInfo:      "regular",
			partitionInfo: "regular",

			screenResInfo:  "regular",
			screenSizeInfo: "error",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			l := testutils.NewMockHandler(slog.LevelDebug)

			options := []hardware.Options{
				hardware.WithLogger(&l),
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

			if tc.screenSizeInfo != "-" {
				cmdArgs := testutils.SetupFakeCmdArgs("TestFakeScreenSizeInfo", tc.screenSizeInfo)
				options = append(options, hardware.WithScreenSizeInfo(cmdArgs))
			}

			s := hardware.New(options...)

			got, err := s.Collect(platform.Info{})
			if tc.wantErr {
				require.Error(t, err, "Collect should return an error and didn’t")
				return
			}
			require.NoError(t, err, "Collect should not return an error")

			sGot, err := yaml.Marshal(got)
			require.NoError(t, err, "Failed to marshal sysinfo to yaml")
			want := testutils.LoadWithUpdateFromGolden(t, string(sGot))
			assert.Equal(t, strings.ReplaceAll(want, "\r\n", "\n"), string(sGot), "Collect should return expected sys information")

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

ConfigManagerErrorCode      : 0
LastErrorCode               : 
NeedsCleaning               : 
Status                      : OK
DeviceID                    : \\.\PHYSICALDRIVE3
StatusInfo                  : 
Partitions                  : 3
BytesPerSector              : 512
ConfigManagerUserConfig     : False
DefaultBlockSize            : 
Index                       : 3
InstallDate                 : 
InterfaceType               : SCSI
MaxBlockSize                : 
MaxMediaSize                : 
MinBlockSize                : 
NumberOfMediaSupported      : 
SectorsPerTrack             : 63
Size                        : 12345
TotalCylinders              : 12345
TotalHeads                  : 255
TotalSectors                : 1953520065
TotalTracks                 : 31008255
TracksPerCylinder           : 255
Caption                     : REDACTED SSD
Description                 : Disk drive
Name                        : \\.\PHYSICALDRIVE3
Availability                : 
CreationClassName           : Win32_DiskDrive
ErrorCleared                : 
ErrorDescription            : 
PNPDeviceID                 : REDACTED
PowerManagementCapabilities : 
PowerManagementSupported    : 
SystemCreationClassName     : Win32_ComputerSystem
SystemName                  : REDACTED SYSTEM NAME
Capabilities                : {3, 4}
CapabilityDescriptions      : {Random Access, Supports Writing}
CompressionMethod           : 
ErrorMethodology            : 
FirmwareRevision            : REDACTED
Manufacturer                : (Standard disk drives)
MediaLoaded                 : True
MediaType                   : Fixed hard disk media
Model                       : REDACTED SSD MODEL
SCSIBus                     : 0
SCSILogicalUnit             : 0
SCSIPort                    : 3
SCSITargetId                : 0
SerialNumber                : REDACTED SERIAL NUMBER.
Signature                   : 
PSComputerName              : 
CimClass                    : root/cimv2:Win32_DiskDrive
CimInstanceProperties       : {Caption, Description, InstallDate, Name…}
CimSystemProperties         : Microsoft.Management.Infrastructure.CimSystemProperties

ConfigManagerErrorCode      : 0
LastErrorCode               : 
NeedsCleaning               : 
Status                      : OK
DeviceID                    : \\.\PHYSICALDRIVE4
StatusInfo                  : 
Partitions                  : 1
BytesPerSector              : 512
ConfigManagerUserConfig     : False
DefaultBlockSize            : 
Index                       : 4
InstallDate                 : 
InterfaceType               : SCSI
MaxBlockSize                : 
MaxMediaSize                : 
MinBlockSize                : 
NumberOfMediaSupported      : 
SectorsPerTrack             : 63
Size                        : 536864025600
TotalCylinders              : 65270
TotalHeads                  : 255
TotalSectors                : 1048562550
TotalTracks                 : 16643850
TracksPerCylinder           : 255
Caption                     : Microsoft Virtual Disk
Description                 : Disk drive
Name                        : \\.\PHYSICALDRIVE4
Availability                : 
CreationClassName           : Win32_DiskDrive
ErrorCleared                : 
ErrorDescription            : 
PNPDeviceID                 : SCSI\DISK&VEN_MSFT&PROD_VIRTUAL_DISK\
PowerManagementCapabilities : 
PowerManagementSupported    : 
SystemCreationClassName     : Win32_ComputerSystem
SystemName                  : REDACTED SYSTEM NAME
Capabilities                : {3, 4}
CapabilityDescriptions      : {Random Access, Supports Writing}
CompressionMethod           : 
ErrorMethodology            : 
FirmwareRevision            : 1.0
Manufacturer                : (Standard disk drives)
MediaLoaded                 : True
MediaType                   : Fixed hard disk media
Model                       : Microsoft Virtual Disk
SCSIBus                     : 0
SCSILogicalUnit             : 1
SCSIPort                    : 4
SCSITargetId                : 0
SerialNumber                : 
Signature                   : 
PSComputerName              : 
CimClass                    : root/cimv2:Win32_DiskDrive
CimInstanceProperties       : {Caption, Description, InstallDate, Name…}
CimSystemProperties         : Microsoft.Management.Infrastructure.CimSystemProperties

ConfigManagerErrorCode      : 0
LastErrorCode               : 
NeedsCleaning               : 
Status                      : OK
DeviceID                    : \\.\PHYSICALDRIVE0
StatusInfo                  : 
Partitions                  : 1
BytesPerSector              : 512
ConfigManagerUserConfig     : False
DefaultBlockSize            : 
Index                       : 0
InstallDate                 : 
InterfaceType               : IDE
MaxBlockSize                : 
MaxMediaSize                : 
MinBlockSize                : 
NumberOfMediaSupported      : 
SectorsPerTrack             : 63
Size                        : 10000830067200
TotalCylinders              : 1215865
TotalHeads                  : 255
TotalSectors                : 19532871225
TotalTracks                 : 310045575
TracksPerCylinder           : 255
Caption                     : REDACTED
Description                 : Disk drive
Name                        : \\.\PHYSICALDRIVE0
Availability                : 
CreationClassName           : Win32_DiskDrive
ErrorCleared                : 
ErrorDescription            : 
PNPDeviceID                 : REDACTED
PowerManagementCapabilities : 
PowerManagementSupported    : 
SystemCreationClassName     : Win32_ComputerSystem
SystemName                  : REDACTED SYSTEM NAME
Capabilities                : {3, 4}
CapabilityDescriptions      : {Random Access, Supports Writing}
CompressionMethod           : 
ErrorMethodology            : 
FirmwareRevision            : DN01
Manufacturer                : (Standard disk drives)
MediaLoaded                 : True
MediaType                   : Fixed hard disk media
Model                       : REDACTED
SCSIBus                     : 0
SCSILogicalUnit             : 0
SCSIPort                    : 0
SCSITargetId                : 0
SerialNumber                :             REDACTED SERIAL NUMBER
Signature                   : 
PSComputerName              : 
CimClass                    : root/cimv2:Win32_DiskDrive
CimInstanceProperties       : {Caption, Description, InstallDate, Name…}
CimSystemProperties         : Microsoft.Management.Infrastructure.CimSystemProperties

ConfigManagerErrorCode      : 0
LastErrorCode               : 
NeedsCleaning               : 
Status                      : OK
DeviceID                    : \\.\PHYSICALDRIVE2
StatusInfo                  : 
Partitions                  : 2
BytesPerSector              : 512
ConfigManagerUserConfig     : False
DefaultBlockSize            : 
Index                       : 2
InstallDate                 : 
InterfaceType               : SCSI
MaxBlockSize                : 
MaxMediaSize                : 
MinBlockSize                : 
NumberOfMediaSupported      : 
SectorsPerTrack             : 63
Size                        : 1024203640320
TotalCylinders              : 124519
TotalHeads                  : 255
TotalSectors                : 2000397735
TotalTracks                 : 31752345
TracksPerCylinder           : 255
Caption                     : REDACTED
Description                 : Disk drive
Name                        : \\.\PHYSICALDRIVE2
Availability                : 
CreationClassName           : Win32_DiskDrive
ErrorCleared                : 
ErrorDescription            : 
PNPDeviceID                 : REDACTED
PowerManagementCapabilities : 
PowerManagementSupported    : 
SystemCreationClassName     : Win32_ComputerSystem
SystemName                  : REDACTED SYSTEM NAME
Capabilities                : {3, 4}
CapabilityDescriptions      : {Random Access, Supports Writing}
CompressionMethod           : 
ErrorMethodology            : 
FirmwareRevision            : 70626000
Manufacturer                : (Standard disk drives)
MediaLoaded                 : True
MediaType                   : Fixed hard disk media
Model                       : REDACTED MODEL
SCSIBus                     : 0
SCSILogicalUnit             : 0
SCSIPort                    : 2
SCSITargetId                : 0
SerialNumber                : REDACTED SERIAL NUMBER.
Signature                   : 
PSComputerName              : 
CimClass                    : root/cimv2:Win32_DiskDrive
CimInstanceProperties       : {Caption, Description, InstallDate, Name…}
CimSystemProperties         : Microsoft.Management.Infrastructure.CimSystemProperties

ConfigManagerErrorCode      : 0
LastErrorCode               : 
NeedsCleaning               : 
Status                      : OK
DeviceID                    : \\.\PHYSICALDRIVE1
StatusInfo                  : 
Partitions                  : 1
BytesPerSector              : 512
ConfigManagerUserConfig     : False
DefaultBlockSize            : 
Index                       : 1
InstallDate                 : 
InterfaceType               : IDE
MaxBlockSize                : 
MaxMediaSize                : 
MinBlockSize                : 
NumberOfMediaSupported      : 
SectorsPerTrack             : 63
Size                        : 1000202273280
TotalCylinders              : 121601
TotalHeads                  : 255
TotalSectors                : 1953520065
TotalTracks                 : 31008255
TracksPerCylinder           : 255
Caption                     : REDACTED
Description                 : Disk drive
Name                        : \\.\PHYSICALDRIVE1
Availability                : 
CreationClassName           : Win32_DiskDrive
ErrorCleared                : 
ErrorDescription            : 
PNPDeviceID                 : REDACTED ID
PowerManagementCapabilities : 
PowerManagementSupported    : 
SystemCreationClassName     : Win32_ComputerSystem
SystemName                  : REDACTED SYSTEM NAME
Capabilities                : {3, 4}
CapabilityDescriptions      : {Random Access, Supports Writing}
CompressionMethod           : 
ErrorMethodology            : 
FirmwareRevision            : REDACTED FIRMWARE
Manufacturer                : (Standard disk drives)
MediaLoaded                 : True
MediaType                   : Fixed hard disk media
Model                       : REDACTED MODEL
SCSIBus                     : 2
SCSILogicalUnit             : 0
SCSIPort                    : 0
SCSITargetId                : 0
SerialNumber                : REDACTED SERIAL NUMBER.
Signature                   : 
PSComputerName              : 
CimClass                    : root/cimv2:Win32_DiskDrive
CimInstanceProperties       : {Caption, Description, InstallDate, Name…}
CimSystemProperties         : Microsoft.Management.Infrastructure.CimSystemProperties`)
	case "malicious":
		fmt.Println(`

Partitions                  : 999999999999
BytesPerSector              : 512
Index                       : 0
SectorsPerTrack             : 63
Size                        : 2000396321280
TotalCylinders              : 243201
TotalHeads                  : 255
TotalSectors                : 3907024065
TotalTracks                 : 62016255
TracksPerCylinder           : 255
Caption                     : WD Green SN350 2TB
Description                 : Disk drive
Name                        : \\.\PHYSICALDRIVE0
Model                       : WD Green SN350 2TB

Partitions                  : -1
BytesPerSector              : 512
Index                       : 1
SectorsPerTrack             : 63
Size                        : 2000396321280
TotalCylinders              : 243201
TotalHeads                  : 255
TotalSectors                : 3907024065
TotalTracks                 : 62016255
TracksPerCylinder           : 255
Caption                     : WD Green SN350 2TB
Description                 : Disk drive
Name                        : \\.\PHYSICALDRIVE0
Model                       : WD Green SN350 2TB

Partitions                  : one gazillion
BytesPerSector              : 512
Index                       : 2
SectorsPerTrack             : 63
Size                        : 2000396321280
TotalCylinders              : 243201
TotalHeads                  : 255
TotalSectors                : 3907024065
TotalTracks                 : 62016255
TracksPerCylinder           : 255
Caption                     : WD Green SN350 2TB
Description                 : Disk drive
Name                        : \\.\PHYSICALDRIVE0
Model                       : WD Green SN350 2TB

Partitions                  : 1
BytesPerSector              : 512
Index                       : -2
SectorsPerTrack             : 63
Size                        : 2000396321280
TotalCylinders              : 243201
TotalHeads                  : 255
TotalSectors                : 3907024065
TotalTracks                 : 62016255
TracksPerCylinder           : 255
Caption                     : Generic Drive
Description                 : Disk drive
Name                        : \\.\PHYSICALDRIVE111
Model                       : Generic Drive

Partitions                  : 1
BytesPerSector              : 512
Index                       : 500
SectorsPerTrack             : 63
Size                        : 2000396321280
TotalCylinders              : 243201
TotalHeads                  : 255
TotalSectors                : 3907024065
TotalTracks                 : 62016255
TracksPerCylinder           : 255
Caption                     : Generic Drive
Description                 : Disk drive
Name                        : \\.\PHYSICALDRIVE111
Model                       : Generic Drive

Partitions                  : 1
BytesPerSector              : 512
Index                       : 3
SectorsPerTrack             : 63
Size                        : Not a Number
TotalCylinders              : 243201
TotalHeads                  : 255
TotalSectors                : 3907024065
TotalTracks                 : 62016255
TracksPerCylinder           : 255
Caption                     : Generic Drive
Description                 : Disk drive
Name                        : \\.\PHYSICALDRIVE111
Model                       : Generic Drive`)
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

Index                       : 0
Status                      : 
StatusInfo                  : 
Name                        : Disk #3, Partition #0
Caption                     : Disk #3, Partition #0
Description                 : GPT: System
InstallDate                 : 
Availability                : 
ConfigManagerErrorCode      : 
ConfigManagerUserConfig     : 
CreationClassName           : Win32_DiskPartition
DeviceID                    : Disk #3, Partition #0
ErrorCleared                : 
ErrorDescription            : 
LastErrorCode               : 
PNPDeviceID                 : 
PowerManagementCapabilities : 
PowerManagementSupported    : 
SystemCreationClassName     : Win32_ComputerSystem
SystemName                  : REDACTED SYSTEM NAME
Access                      : 
BlockSize                   : 512
ErrorMethodology            : 
NumberOfBlocks              : 204800
Purpose                     : 
Bootable                    : True
PrimaryPartition            : True
BootPartition               : True
DiskIndex                   : 3
HiddenSectors               : 
RewritePartition            : 
Size                        : 104857600
StartingOffset              : 1048576
Type                        : GPT: System
PSComputerName              : 
CimClass                    : root/cimv2:Win32_DiskPartition
CimInstanceProperties       : {Caption, Description, InstallDate, Name…}
CimSystemProperties         : Microsoft.Management.Infrastructure.CimSystemProperties

Index                       : 1
Status                      : 
StatusInfo                  : 
Name                        : Disk #3, Partition #1
Caption                     : Disk #3, Partition #1
Description                 : GPT: Basic Data
InstallDate                 : 
Availability                : 
ConfigManagerErrorCode      : 
ConfigManagerUserConfig     : 
CreationClassName           : Win32_DiskPartition
DeviceID                    : Disk #3, Partition #1
ErrorCleared                : 
ErrorDescription            : 
LastErrorCode               : 
PNPDeviceID                 : 
PowerManagementCapabilities : 
PowerManagementSupported    : 
SystemCreationClassName     : Win32_ComputerSystem
SystemName                  : REDACTED SYSTEM NAME
Access                      : 
BlockSize                   : 512
ErrorMethodology            : 
NumberOfBlocks              : 1951709184
Purpose                     : 
Bootable                    : False
PrimaryPartition            : True
BootPartition               : False
DiskIndex                   : 3
HiddenSectors               : 
RewritePartition            : 
Size                        : 999275102208
StartingOffset              : 122683392
Type                        : GPT: Basic Data
PSComputerName              : 
CimClass                    : root/cimv2:Win32_DiskPartition
CimInstanceProperties       : {Caption, Description, InstallDate, Name…}
CimSystemProperties         : Microsoft.Management.Infrastructure.CimSystemProperties

Index                       : 2
Status                      : 
StatusInfo                  : 
Name                        : Disk #3, Partition #2
Caption                     : Disk #3, Partition #2
Description                 : GPT: Unknown
InstallDate                 : 
Availability                : 
ConfigManagerErrorCode      : 
ConfigManagerUserConfig     : 
CreationClassName           : Win32_DiskPartition
DeviceID                    : Disk #3, Partition #2
ErrorCleared                : 
ErrorDescription            : 
LastErrorCode               : 
PNPDeviceID                 : 
PowerManagementCapabilities : 
PowerManagementSupported    : 
SystemCreationClassName     : Win32_ComputerSystem
SystemName                  : REDACTED SYSTEM NAME
Access                      : 
BlockSize                   : 512
ErrorMethodology            : 
NumberOfBlocks              : 1572864
Purpose                     : 
Bootable                    : False
PrimaryPartition            : False
BootPartition               : False
DiskIndex                   : 3
HiddenSectors               : 
RewritePartition            : 
Size                        : 805306368
StartingOffset              : 999397785600
Type                        : GPT: Unknown
PSComputerName              : 
CimClass                    : root/cimv2:Win32_DiskPartition
CimInstanceProperties       : {Caption, Description, InstallDate, Name…}
CimSystemProperties         : Microsoft.Management.Infrastructure.CimSystemProperties

Index                       : 0
Status                      : 
StatusInfo                  : 
Name                        : Disk #4, Partition #0
Caption                     : Disk #4, Partition #0
Description                 : GPT: Basic Data
InstallDate                 : 
Availability                : 
ConfigManagerErrorCode      : 
ConfigManagerUserConfig     : 
CreationClassName           : Win32_DiskPartition
DeviceID                    : Disk #4, Partition #0
ErrorCleared                : 
ErrorDescription            : 
LastErrorCode               : 
PNPDeviceID                 : 
PowerManagementCapabilities : 
PowerManagementSupported    : 
SystemCreationClassName     : Win32_ComputerSystem
SystemName                  : REDACTED SYSTEM NAME
Access                      : 
BlockSize                   : 512
ErrorMethodology            : 
NumberOfBlocks              : 1048541184
Purpose                     : 
Bootable                    : False
PrimaryPartition            : True
BootPartition               : False
DiskIndex                   : 4
HiddenSectors               : 
RewritePartition            : 
Size                        : 536853086208
StartingOffset              : 16777216
Type                        : GPT: Basic Data
PSComputerName              : 
CimClass                    : root/cimv2:Win32_DiskPartition
CimInstanceProperties       : {Caption, Description, InstallDate, Name…}
CimSystemProperties         : Microsoft.Management.Infrastructure.CimSystemProperties

Index                       : 0
Status                      : 
StatusInfo                  : 
Name                        : Disk #0, Partition #0
Caption                     : Disk #0, Partition #0
Description                 : GPT: Basic Data
InstallDate                 : 
Availability                : 
ConfigManagerErrorCode      : 
ConfigManagerUserConfig     : 
CreationClassName           : Win32_DiskPartition
DeviceID                    : Disk #0, Partition #0
ErrorCleared                : 
ErrorDescription            : 
LastErrorCode               : 
PNPDeviceID                 : 
PowerManagementCapabilities : 
PowerManagementSupported    : 
SystemCreationClassName     : Win32_ComputerSystem
SystemName                  : REDACTED SYSTEM NAME
Access                      : 
BlockSize                   : 512
ErrorMethodology            : 
NumberOfBlocks              : 19532869632
Purpose                     : 
Bootable                    : False
PrimaryPartition            : True
BootPartition               : False
DiskIndex                   : 0
HiddenSectors               : 
RewritePartition            : 
Size                        : 10000829251584
StartingOffset              : 1048576
Type                        : GPT: Basic Data
PSComputerName              : 
CimClass                    : root/cimv2:Win32_DiskPartition
CimInstanceProperties       : {Caption, Description, InstallDate, Name…}
CimSystemProperties         : Microsoft.Management.Infrastructure.CimSystemProperties

Index                       : 0
Status                      : 
StatusInfo                  : 
Name                        : Disk #2, Partition #0
Caption                     : Disk #2, Partition #0
Description                 : GPT: System
InstallDate                 : 
Availability                : 
ConfigManagerErrorCode      : 
ConfigManagerUserConfig     : 
CreationClassName           : Win32_DiskPartition
DeviceID                    : Disk #2, Partition #0
ErrorCleared                : 
ErrorDescription            : 
LastErrorCode               : 
PNPDeviceID                 : 
PowerManagementCapabilities : 
PowerManagementSupported    : 
SystemCreationClassName     : Win32_ComputerSystem
SystemName                  : REDACTED SYSTEM NAME
Access                      : 
BlockSize                   : 512
ErrorMethodology            : 
NumberOfBlocks              : 2201600
Purpose                     : 
Bootable                    : True
PrimaryPartition            : True
BootPartition               : True
DiskIndex                   : 2
HiddenSectors               : 
RewritePartition            : 
Size                        : 1127219200
StartingOffset              : 1048576
Type                        : GPT: System
PSComputerName              : 
CimClass                    : root/cimv2:Win32_DiskPartition
CimInstanceProperties       : {Caption, Description, InstallDate, Name…}
CimSystemProperties         : Microsoft.Management.Infrastructure.CimSystemProperties

Index                       : 1
Status                      : 
StatusInfo                  : 
Name                        : Disk #2, Partition #1
Caption                     : Disk #2, Partition #1
Description                 : GPT: Unknown
InstallDate                 : 
Availability                : 
ConfigManagerErrorCode      : 
ConfigManagerUserConfig     : 
CreationClassName           : Win32_DiskPartition
DeviceID                    : Disk #2, Partition #1
ErrorCleared                : 
ErrorDescription            : 
LastErrorCode               : 
PNPDeviceID                 : 
PowerManagementCapabilities : 
PowerManagementSupported    : 
SystemCreationClassName     : Win32_ComputerSystem
SystemName                  : REDACTED SYSTEM NAME
Access                      : 
BlockSize                   : 512
ErrorMethodology            : 
NumberOfBlocks              : 1998204928
Purpose                     : 
Bootable                    : False
PrimaryPartition            : False
BootPartition               : False
DiskIndex                   : 2
HiddenSectors               : 
RewritePartition            : 
Size                        : 1023080923136
StartingOffset              : 1128267776
Type                        : GPT: Unknown
PSComputerName              : 
CimClass                    : root/cimv2:Win32_DiskPartition
CimInstanceProperties       : {Caption, Description, InstallDate, Name…}
CimSystemProperties         : Microsoft.Management.Infrastructure.CimSystemProperties

Index                       : 0
Status                      : 
StatusInfo                  : 
Name                        : Disk #1, Partition #0
Caption                     : Disk #1, Partition #0
Description                 : GPT: Basic Data
InstallDate                 : 
Availability                : 
ConfigManagerErrorCode      : 
ConfigManagerUserConfig     : 
CreationClassName           : Win32_DiskPartition
DeviceID                    : Disk #1, Partition #0
ErrorCleared                : 
ErrorDescription            : 
LastErrorCode               : 
PNPDeviceID                 : 
PowerManagementCapabilities : 
PowerManagementSupported    : 
SystemCreationClassName     : Win32_ComputerSystem
SystemName                  : REDACTED SYSTEM NAME
Access                      : 
BlockSize                   : 512
ErrorMethodology            : 
NumberOfBlocks              : 1953488896
Purpose                     : 
Bootable                    : False
PrimaryPartition            : True
BootPartition               : False
DiskIndex                   : 1
HiddenSectors               : 
RewritePartition            : 
Size                        : 1000186314752
StartingOffset              : 16777216
Type                        : GPT: Basic Data
PSComputerName              : 
CimClass                    : root/cimv2:Win32_DiskPartition
CimInstanceProperties       : {Caption, Description, InstallDate, Name…}
CimSystemProperties         : Microsoft.Management.Infrastructure.CimSystemProperties`)
	case "malicious":
		fmt.Println(`

Index                       : -1
Name                        : Disk #0, Partition #-1
DiskIndex                   : 0
Size                        : 314572800

Index                       : alpha
Name                        : Disk #0, Partition alpha
DiskIndex                   : 0
Size                        : 943718400

Index                       : 4
Name                        : Disk #0, Partition #4
DiskIndex                   : 0
Size                        : 22153265152

Index                       : 0
Name                        : Disk #-1, Partition #0
DiskIndex                   : -1
Size                        : 22153265152

Index                       : 1
Name                        : Disk #1, Partition #1
DiskIndex                   : 1
Size                        : 22153265152

Index                       : 2
Name                        : Disk beta, Partition #2
DiskIndex                   : beta
Size                        : 1976850972672

Index                       : 500
Name                        : Disk 1, Partition #500
DiskIndex                   : 1
Size                        : 1976850972672

Index                       : 3
Name                        : Disk #0, Partition #3
DiskIndex                   : 0
Size                        : -20

Index                       : 3
Name                        : Disk #500, Partition #0
DiskIndex                   : 500
Size                        : -20`)
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

DeviceID                    : DesktopMonitor1
Name                        : Default Monitor
PixelsPerXLogicalInch       : 120
PixelsPerYLogicalInch       : 120
ScreenHeight                : 1080
ScreenWidth                 : 1920
IsLocked                    :
LastErrorCode               :
Status                      : OK
StatusInfo                  :
Caption                     : Default Monitor
Description                 : Default Monitor
InstallDate                 :
Availability                : 3
ConfigManagerErrorCode      :
ConfigManagerUserConfig     :
CreationClassName           : Win32_DesktopMonitor
ErrorCleared                :
ErrorDescription            :
PNPDeviceID                 :
PowerManagementCapabilities :
PowerManagementSupported    :
SystemCreationClassName     : Win32_ComputerSystem
SystemName                  : MSI
Bandwidth                   :
DisplayType                 :
MonitorManufacturer         :
MonitorType                 : Default Monitor

DeviceID                    : DesktopMonitor2
Name                        : Generic PnP Monitor
PixelsPerXLogicalInch       : 120
PixelsPerYLogicalInch       : 120
ScreenHeight                : 1080
ScreenWidth                 : 1920
IsLocked                    :
LastErrorCode               :
Status                      : OK
StatusInfo                  :
Caption                     : Generic PnP Monitor
Description                 : Generic PnP Monitor
InstallDate                 :
Availability                : 3
ConfigManagerErrorCode      : 0
ConfigManagerUserConfig     : False
CreationClassName           : Win32_DesktopMonitor
ErrorCleared                :
ErrorDescription            :
PNPDeviceID                 : DISPLAY\AUOAF90\4&28FE40F5&0&UID8388688
PowerManagementCapabilities :
PowerManagementSupported    :
SystemCreationClassName     : Win32_ComputerSystem
SystemName                  : MSI
Bandwidth                   :
DisplayType                 :
MonitorManufacturer         : (Standard monitor types)
MonitorType                 : Generic PnP Monitor`)
	case "":
		fallthrough
	case "missing":
		os.Exit(0)
	}
}

func TestFakeScreenSizeInfo(_ *testing.T) {
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

Active                        : True
DisplayTransferCharacteristic : 120
InstanceName                  : DISPLAY\AUOAF90\4&28fe40f5&0&UID1234
MaxHorizontalImageSize        : 34
MaxVerticalImageSize          : 19
SupportedDisplayFeatures      : WmiMonitorSupportedDisplayFeatures
VideoInputType                : 1

Active                        : True
DisplayTransferCharacteristic : 120
InstanceName                  : DISPLAY\ACR09D8\4&28fe40f5&0&UID4321
MaxHorizontalImageSize        : 60
MaxVerticalImageSize          : 34
SupportedDisplayFeatures      : WmiMonitorSupportedDisplayFeatures
VideoInputType                : 1`)
	case "":
		fallthrough
	case "missing":
		os.Exit(0)
	}
}
