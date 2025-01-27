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
		gpuInfo string
		productInfo string

		logs    map[slog.Level]uint
		wantErr bool
	}{
		"Regular hardware information": {
			gpuInfo: "regular",
			productInfo: "regular",
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
			if tc.gpuInfo != "-" {
				cmdArgs := []string{os.Args[0], "-test.run=TestMockGPUInfo", "--"}
				cmdArgs = append(cmdArgs, tc.gpuInfo)
				options = append(options, sysinfo.WithGPUInfo(cmdArgs))
			}

			if tc.productInfo != "-" {
				cmdArgs := []string{os.Args[0], "-test.run=TestMockProductInfo", "--"}
				cmdArgs = append(cmdArgs, tc.productInfo)
				options = append(options, sysinfo.WithProductInfo(cmdArgs))
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
					fmt.Printf("Logged %v: %s\n", call.Level, call.Message)
				}
			}
		})
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

__GENUS                      : 2
__CLASS                      : Win32_VideoController
__SUPERCLASS                 : CIM_PCVideoController
__DYNASTY                    : CIM_ManagedSystemElement
__RELPATH                    : Win32_VideoController.DeviceID="VideoController1"
__PROPERTY_COUNT             : 59
__DERIVATION                 : {CIM_PCVideoController, CIM_VideoController, CIM_Controller, CIM_LogicalDevice...}
__NAMESPACE                  : root\cimv2
__PATH                       : root\cimv2:Win32_VideoController.DeviceID="VideoController1"
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

__GENUS                      : 2
__CLASS                      : Win32_VideoController
__SUPERCLASS                 : CIM_PCVideoController
__DYNASTY                    : CIM_ManagedSystemElement
__RELPATH                    : Win32_VideoController.DeviceID="VideoController2"
__PROPERTY_COUNT             : 59
__DERIVATION                 : {CIM_PCVideoController, CIM_VideoController, CIM_Controller, CIM_LogicalDevice...}
__NAMESPACE                  : root\cimv2
__PATH                       : root\cimv2:Win32_VideoController.DeviceID="VideoController2"
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

Manufacturer : Micro-Star International Co., Ltd.
Model        :
Name         : Base Board
SerialNumber : BSS-0123456789
SKU          :
Product      : MS-1582`)
	case "":
		fallthrough
	case "missing":
		os.Exit(0)
	}
}
