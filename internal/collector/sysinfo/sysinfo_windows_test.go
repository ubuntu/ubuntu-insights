package sysinfo_test

import (
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo"
	"github.com/ubuntu/ubuntu-insights/internal/testutils"
)

func TestCollectWindows(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		productInfo string
		cpuInfo string
		gpuInfo string
		

		logs    map[slog.Level]uint
		wantErr bool
	}{
		"Regular hardware information": {
			productInfo: "regular",
			cpuInfo: "regular",
			gpuInfo: "regular",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tmp := t.TempDir()
			err := testutils.CopyDir("testdata/windowsfs", tmp)
			require.NoError(t, err, "setup: failed to copy testdata directory")

			l := testutils.NewMockHandler()

			options := []sysinfo.Options{
				sysinfo.WithLogger(&l),
			}

			os.Setenv("GO_WANT_HELPER_PROCESS", "1")
			if tc.productInfo != "-" {
				cmdArgs := []string{os.Args[0], "-test.run=TestMockProductInfo", "--"}
				cmdArgs = append(cmdArgs, tc.productInfo)
				options = append(options, sysinfo.WithProductInfo(cmdArgs))
			}

			if tc.cpuInfo != "-" {
				cmdArgs := []string{os.Args[0], "-test.run=TestMockCPUInfo", "--"}
				cmdArgs = append(cmdArgs, tc.cpuInfo)
				options = append(options, sysinfo.WithCPUInfo(cmdArgs))
			}

			if tc.gpuInfo != "-" {
				cmdArgs := []string{os.Args[0], "-test.run=TestMockGPUInfo", "--"}
				cmdArgs = append(cmdArgs, tc.gpuInfo)
				options = append(options, sysinfo.WithGPUInfo(cmdArgs))
			}

			s := sysinfo.New(options...)

			got, err := s.Collect()
			if tc.wantErr {
				require.Error(t, err, "Collect should return an error and didn’t")
				return
			}
			require.NoError(t, err, "Collect should not return an error")

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			assert.Equal(t, want, got, "Collect should return expected sys information")

			if !l.AssertLevels(t, tc.logs) {
				for _, call := range l.HandleCalls {
					t.Logf("Logged %v: %s\n", call.Level, call.Message)
				}
			}
		})
	}
}


func TestMockProductInfo(_ *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	args := os.Args
	for len(args) > 0 {
		if args[0] != "--" {
			args = args[1:]
			continue
		}
		args = args[1:]
		break
	}

	switch args[0] {
	case "error":
		fmt.Fprint(os.Stderr, "Error requested in Mock product info")
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

func TestMockCPUInfo(_ *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	args := os.Args
	for len(args) > 0 {
		if args[0] != "--" {
			args = args[1:]
			continue
		}
		args = args[1:]
		break
	}

	switch args[0] {
	case "error":
		fmt.Fprint(os.Stderr, "Error requested in Mock cpu info")
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
	case "":
		fallthrough
	case "missing":
		os.Exit(0)
	}
}

func TestMockGPUInfo(_ *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	args := os.Args
	for len(args) > 0 {
		if args[0] != "--" {
			args = args[1:]
			continue
		}
		args = args[1:]
		break
	}

	switch args[0] {
	case "error":
		fmt.Fprint(os.Stderr, "Error requested in Mock gpu info")
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
