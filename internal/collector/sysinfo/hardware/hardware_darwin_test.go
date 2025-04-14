package hardware_test

import (
	"encoding/xml"
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

func TestCollectDarwin(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		cpuInfo    string
		gpuInfo    string
		memInfo    string
		diskInfo   string
		screenInfo string

		logs    map[slog.Level]uint
		wantErr bool
	}{
		"Regular hardware information": {
			cpuInfo:    "regular",
			gpuInfo:    "regular",
			memInfo:    "regular",
			diskInfo:   "regular",
			screenInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelInfo: 1,
			},
		},

		"Missing information": {
			cpuInfo:    "",
			gpuInfo:    "",
			memInfo:    "",
			diskInfo:   "",
			screenInfo: "",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 5,
			},
		},

		"Missing CPU is missing": {
			cpuInfo:    "",
			gpuInfo:    "regular",
			memInfo:    "regular",
			diskInfo:   "regular",
			screenInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelInfo: 1,
				slog.LevelWarn: 1,
			},
		},

		"Negative CPU values warns": {
			cpuInfo:    "negative",
			gpuInfo:    "regular",
			memInfo:    "regular",
			diskInfo:   "regular",
			screenInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelInfo: 1,
				slog.LevelWarn: 3,
			},
		},

		"Zero CPU values warns": {
			cpuInfo:    "zero",
			gpuInfo:    "regular",
			memInfo:    "regular",
			diskInfo:   "regular",
			screenInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelInfo: 1,
				slog.LevelWarn: 1,
			},
		},

		"Bad CPU warns": {
			cpuInfo:    "bad",
			gpuInfo:    "regular",
			memInfo:    "regular",
			diskInfo:   "regular",
			screenInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelInfo: 1,
				slog.LevelWarn: 1,
			},
		},

		"Missing GPU is missing": {
			cpuInfo:    "regular",
			gpuInfo:    "",
			memInfo:    "regular",
			diskInfo:   "regular",
			screenInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelInfo: 1,
				slog.LevelWarn: 1,
			},
		},

		"Bad GPU warns": {
			cpuInfo:    "regular",
			gpuInfo:    "bad",
			memInfo:    "regular",
			diskInfo:   "regular",
			screenInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelInfo: 1,
				slog.LevelWarn: 1,
			},
		},

		"Error GPU warns": {
			cpuInfo:    "regular",
			gpuInfo:    "error",
			memInfo:    "regular",
			diskInfo:   "regular",
			screenInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelInfo: 1,
				slog.LevelWarn: 1,
			},
		},

		"Missing Memory is missing": {
			cpuInfo:    "regular",
			gpuInfo:    "regular",
			memInfo:    "",
			diskInfo:   "regular",
			screenInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelInfo: 1,
				slog.LevelWarn: 1,
			},
		},

		"Negative Memory values warns": {
			cpuInfo:    "regular",
			gpuInfo:    "regular",
			memInfo:    "negative",
			diskInfo:   "regular",
			screenInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelInfo: 1,
				slog.LevelWarn: 1,
			},
		},

		"Bad Memory warns": {
			cpuInfo:    "regular",
			gpuInfo:    "regular",
			memInfo:    "bad",
			diskInfo:   "regular",
			screenInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelInfo: 1,
				slog.LevelWarn: 1,
			},
		},

		"Missing Disk is missing": {
			cpuInfo:    "regular",
			gpuInfo:    "regular",
			memInfo:    "regular",
			diskInfo:   "",
			screenInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Disk info missing xml version warns": {
			cpuInfo:    "regular",
			gpuInfo:    "regular",
			memInfo:    "regular",
			diskInfo:   "missing xml",
			screenInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Disk info missing doctype warns": {
			cpuInfo:    "regular",
			gpuInfo:    "regular",
			memInfo:    "regular",
			diskInfo:   "missing doctype",
			screenInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Disk info missing plist warns": {
			cpuInfo:    "regular",
			gpuInfo:    "regular",
			memInfo:    "regular",
			diskInfo:   "missing plist",
			screenInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Disk info missing dict keys warns": {
			cpuInfo:    "regular",
			gpuInfo:    "regular",
			memInfo:    "regular",
			diskInfo:   "no dict key",
			screenInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Disk info missing AllDisksAndPartitions warns": {
			cpuInfo:    "regular",
			gpuInfo:    "regular",
			memInfo:    "regular",
			diskInfo:   "missing AllDisksAndPartitions",
			screenInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Disk info with bad sizes warns": {
			cpuInfo:    "regular",
			gpuInfo:    "regular",
			memInfo:    "regular",
			diskInfo:   "bad sizes",
			screenInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Disk info with bad value type warns": {
			cpuInfo:    "regular",
			gpuInfo:    "regular",
			memInfo:    "regular",
			diskInfo:   "bad value type",
			screenInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Disk info with negative values warns": {
			cpuInfo:    "regular",
			gpuInfo:    "regular",
			memInfo:    "regular",
			diskInfo:   "negative",
			screenInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 2,
			},
		},

		"Bad Disk warns": {
			cpuInfo:    "regular",
			gpuInfo:    "regular",
			memInfo:    "regular",
			diskInfo:   "bad",
			screenInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Error Disk warns": {
			cpuInfo:    "regular",
			gpuInfo:    "regular",
			memInfo:    "regular",
			diskInfo:   "error",
			screenInfo: "regular",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Missing Screen is missing": {
			cpuInfo:    "regular",
			gpuInfo:    "regular",
			memInfo:    "regular",
			diskInfo:   "regular",
			screenInfo: "",

			logs: map[slog.Level]uint{
				slog.LevelInfo: 1,
				slog.LevelWarn: 1,
			},
		},

		"Error Screen warns": {
			cpuInfo:    "regular",
			gpuInfo:    "regular",
			memInfo:    "regular",
			diskInfo:   "regular",
			screenInfo: "error",

			logs: map[slog.Level]uint{
				slog.LevelInfo: 1,
				slog.LevelWarn: 1,
			},
		},

		"Bad Screen info warns": {
			cpuInfo:    "regular",
			gpuInfo:    "regular",
			memInfo:    "regular",
			diskInfo:   "regular",
			screenInfo: "bad",

			logs: map[slog.Level]uint{
				slog.LevelInfo: 1,
				slog.LevelWarn: 1,
			},
		},

		"Bad Screen values warns": {
			cpuInfo:    "regular",
			gpuInfo:    "regular",
			memInfo:    "regular",
			diskInfo:   "regular",
			screenInfo: "bad values",

			logs: map[slog.Level]uint{
				slog.LevelInfo: 1,
				slog.LevelWarn: 2,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			options := []hardware.Options{
				hardware.WithArch("arm64"),
			}

			if tc.cpuInfo != "-" {
				cmdArgs := testutils.SetupFakeCmdArgs("TestFakeCPUInfo", tc.cpuInfo)
				options = append(options, hardware.WithCPUInfo(cmdArgs))
			}

			if tc.gpuInfo != "-" {
				cmdArgs := testutils.SetupFakeCmdArgs("TestFakeGpuScreenInfo", tc.gpuInfo)
				options = append(options, hardware.WithGPUInfo(cmdArgs))
			}

			if tc.memInfo != "-" {
				cmdArgs := testutils.SetupFakeCmdArgs("TestFakeMemoryInfo", tc.memInfo)
				options = append(options, hardware.WithMemoryInfo(cmdArgs))
			}

			if tc.diskInfo != "-" {
				cmdArgs := testutils.SetupFakeCmdArgs("TestFakeDiskInfo", tc.diskInfo)
				options = append(options, hardware.WithDiskInfo(cmdArgs))
			}

			if tc.screenInfo != "-" {
				cmdArgs := testutils.SetupFakeCmdArgs("TestFakeGpuScreenInfo", tc.screenInfo)
				options = append(options, hardware.WithScreenInfo(cmdArgs))
			}

			l := testutils.NewMockHandler(slog.LevelDebug)
			s := hardware.New(slog.New(&l), options...)

			got, err := s.Collect(platform.Info{})
			if tc.wantErr {
				require.Error(t, err, "Collect should return an error and didn’t")
				return
			}
			require.NoError(t, err, "Collect should not return an error")

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			assert.Equal(t, want, got, "Collect should return expected sys information")

			if !l.AssertLevels(t, tc.logs) {
				l.OutputLogs(t)
			}
		})
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
hw.packages: 1
machdep.cpu.max_basic: 13
machdep.cpu.max_ext: 2147483656
machdep.cpu.vendor: GenuineIntel
machdep.cpu.brand_string: Intel(R) Core(TM) i7-3615QM CPU @ 2.30GHz
machdep.cpu.family: 6
machdep.cpu.model: 58
machdep.cpu.extmodel: 3
machdep.cpu.extfamily: 0
machdep.cpu.stepping: 9
machdep.cpu.brand: 0
machdep.cpu.features: FPU VME DE PSE TSC MSR PAE MCE CX8 APIC SEP MTRR PGE MCA CMOV PAT PSE36 CLFSH DS ACPI MMX FXSR SSE SSE2 SS HTT TM PBE SSE3 PCLMULQDQ DTES64 MON DSCPL VMX EST TM2 SSSE3 CX16 TPR PDCM SSE4.1 SSE4.2 x2APIC POPCNT AES PCID XSAVE OSXSAVE TSCTMR AVX1.0 RDRAND F16C
machdep.cpu.extfeatures: SYSCALL XD EM64T LAHF RDTSCP TSCI
machdep.cpu.logical_per_package: 16
machdep.cpu.cores_per_package: 8
machdep.cpu.microcode_version: 33
machdep.cpu.processor_flag: 4
machdep.cpu.thermal.sensor: 1
machdep.cpu.thermal.dynamic_acceleration: 1
machdep.cpu.thermal.invariant_APIC_timer: 1
machdep.cpu.thermal.thresholds: 2
machdep.cpu.thermal.ACNT_MCNT: 1
machdep.cpu.thermal.core_power_limits: 1
machdep.cpu.thermal.fine_grain_clock_mod: 1
machdep.cpu.thermal.package_thermal_intr: 1
machdep.cpu.thermal.hardware_feedback: 0
machdep.cpu.thermal.energy_policy: 0
machdep.cpu.arch_perf.version: 3
machdep.cpu.arch_perf.number: 4
machdep.cpu.arch_perf.width: 48
machdep.cpu.arch_perf.events_number: 7
machdep.cpu.arch_perf.events: 0
machdep.cpu.arch_perf.fixed_number: 3
machdep.cpu.arch_perf.fixed_width: 48
machdep.cpu.cache.linesize: 64
machdep.cpu.cache.L2_associativity: 8
machdep.cpu.cache.size: 256
machdep.cpu.address_bits.physical: 36
machdep.cpu.address_bits.virtual: 48
machdep.cpu.core_count: 4
machdep.cpu.thread_count: 8
machdep.cpu.tsc_ccc.numerator: 0
machdep.cpu.tsc_ccc.denominator: 0`)
	case "negative":
		fmt.Println(`
hw.packages: -1
machdep.cpu.vendor: GenuineIntel
machdep.cpu.brand_string: Intel(R) Core(TM) i7-3615QM CPU @ 2.30GHz
machdep.cpu.logical_per_package: -16
machdep.cpu.cores_per_package: -8`)
	case "zero":
		fmt.Println(`
hw.packages: 0
machdep.cpu.vendor: GenuineIntel
machdep.cpu.brand_string: Intel(R) Core(TM) i7-3615QM CPU @ 2.30GHz
machdep.cpu.logical_per_package: 0
machdep.cpu.cores_per_package: 0`)
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
hw.memsize: 17179869184`)
	case "negative":
		fmt.Println(`
hw.memsize: -17179869184`)
	case "bad":
		fmt.Println(`
hw.memsize: ONE BILLION!!!`)
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
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>AllDisks</key>
	<array>
		<string>disk0</string>
		<string>disk0s1</string>
		<string>disk0s2</string>
		<string>disk1</string>
		<string>disk1s1</string>
		<string>disk1s2</string>
		<string>disk1s3</string>
		<string>disk1s4</string>
		<string>disk1s5</string>
	</array>
	<key>AllDisksAndPartitions</key>
	<array>
		<dict>
			<key>Content</key>
			<string>GUID_partition_scheme</string>
			<key>DeviceIdentifier</key>
			<string>disk0</string>
			<key>Partitions</key>
			<array>
				<dict>
					<key>Content</key>
					<string>EFI</string>
					<key>DeviceIdentifier</key>
					<string>disk0s1</string>
					<key>Size</key>
					<integer>209715200</integer>
					<key>VolumeName</key>
					<string>EFI</string>
				</dict>
				<dict>
					<key>Content</key>
					<string>Apple_APFS</string>
					<key>DeviceIdentifier</key>
					<string>disk0s2</string>
					<key>Size</key>
					<integer>499763888128</integer>
				</dict>
			</array>
			<key>Size</key>
			<integer>500107862016</integer>
		</dict>
		<dict>
			<key>APFSPhysicalStores</key>
			<array>
				<dict>
					<key>DeviceIdentifier</key>
					<string>disk0s2</string>
				</dict>
			</array>
			<key>APFSVolumes</key>
			<array>
				<dict>
					<key>DeviceIdentifier</key>
					<string>disk1s1</string>
					<key>MountPoint</key>
					<string>/System/Volumes/Data</string>
					<key>Size</key>
					<integer>499763888128</integer>
					<key>VolumeName</key>
					<string>MacintoshHD - Data</string>
				</dict>
				<dict>
					<key>DeviceIdentifier</key>
					<string>disk1s2</string>
					<key>Size</key>
					<integer>499763888128</integer>
					<key>VolumeName</key>
					<string>Preboot</string>
				</dict>
				<dict>
					<key>DeviceIdentifier</key>
					<string>disk1s3</string>
					<key>MountPoint</key>
					<string>/Volumes/Recovery</string>
					<key>Size</key>
					<integer>499763888128</integer>
					<key>VolumeName</key>
					<string>Recovery</string>
				</dict>
				<dict>
					<key>DeviceIdentifier</key>
					<string>disk1s4</string>
					<key>MountPoint</key>
					<string>/private/var/vm</string>
					<key>Size</key>
					<integer>499763888128</integer>
					<key>VolumeName</key>
					<string>VM</string>
				</dict>
				<dict>
					<key>DeviceIdentifier</key>
					<string>disk1s5</string>
					<key>MountPoint</key>
					<string>/</string>
					<key>Size</key>
					<integer>499763888128</integer>
					<key>VolumeName</key>
					<string>MacintoshHD</string>
				</dict>
			</array>
			<key>DeviceIdentifier</key>
			<string>disk1</string>
			<key>Partitions</key>
			<array/>
			<key>Size</key>
			<integer>499763888128</integer>
		</dict>
	</array>
	<key>VolumesFromDisks</key>
	<array>
		<string>MacintoshHD - Data</string>
		<string>Recovery</string>
		<string>VM</string>
		<string>MacintoshHD</string>
	</array>
	<key>WholeDisks</key>
	<array>
		<string>disk0</string>
		<string>disk1</string>
	</array>
</dict>
</plist>`)
	case "missing xml":
		fmt.Println(`
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>AllDisksAndPartitions</key>
	<array>
		<dict>
			<key>DeviceIdentifier</key>
			<string>disk0</string>
			<key>Size</key>
			<integer>500107862016</integer>
		</dict>
	</array>
</dict>
</plist>`)
	case "missing doctype":
		fmt.Println(`
<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
	<key>AllDisksAndPartitions</key>
	<array>
		<dict>
			<key>DeviceIdentifier</key>
			<string>disk0</string>
			<key>Size</key>
			<integer>500107862016</integer>
		</dict>
	</array>
</dict>
</plist>`)
	case "missing plist":
		fmt.Println(`
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<dict>
	<key>AllDisksAndPartitions</key>
	<array>
		<dict>
			<key>DeviceIdentifier</key>
			<string>disk0</string>
			<key>Size</key>
			<integer>500107862016</integer>
		</dict>
	</array>
</dict>`)
	case "missing AllDisksAndPartitions":
		fmt.Println(`
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE cool document type>
<plist version="1.0">
<dict>
	<key>SomeInfo</key>
	<array>
		<dict>
			<key>DeviceIdentifier</key>
			<string>disk0</string>
			<key>Size</key>
			<integer>500107862016</integer>
		</dict>
	</array>
</dict>
</plist>`)
	case "bad sizes":
		fmt.Println(`
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>AllDisksAndPartitions</key>
	<array>
		<dict>
			<key>Content</key>
			<string>GUID_partition_scheme</string>
			<key>DeviceIdentifier</key>
			<string>disk0</string>
			<key>Partitions</key>
			<array>
				<dict>
					<key>Content</key>
					<string>EFI</string>
					<key>DeviceIdentifier</key>
					<string>disk0s1</string>
					<key>Size</key>
					<integer>half a football field</integer>
					<key>VolumeName</key>
					<string>EFI</string>
				</dict>
			</array>
			<key>Size</key>
			<integer>one million bytes</integer>
		</dict>
	</array>
</dict>
</plist>`)
	case "negative":
		fmt.Println(`
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>AllDisksAndPartitions</key>
	<array>
		<dict>
			<key>Content</key>
			<string>GUID_partition_scheme</string>
			<key>DeviceIdentifier</key>
			<string>disk0</string>
			<key>Partitions</key>
			<array>
				<dict>
					<key>Content</key>
					<string>EFI</string>
					<key>DeviceIdentifier</key>
					<string>disk0s1</string>
					<key>Size</key>
					<integer>-1000000</integer>
					<key>VolumeName</key>
					<string>EFI</string>
				</dict>
			</array>
			<key>Size</key>
			<integer>-20000000</integer>
		</dict>
	</array>
</dict>
</plist>`)
	case "bad":
		fmt.Println(`
<?xml version="1.0" encong="UTF-8"?>
<!DOCTYPE plist 
	<key>Alsk1</string>
		<
		<dict>
			<key>ConFI</string>
				</dict>
				<dict>e_APFS</string>
					<key>Device0s2</string>
				ey>
			<integer>500teger>
		<dict>
		<dict>/key>
					<integer>499888128</integer>
		</dict>
	</arraey>VolumesFsk1</string>
</array></dict></plist>`)
	case "":
		fallthrough
	case "missing":
		os.Exit(0)
	}
}

func TestFakeGpuScreenInfo(_ *testing.T) {
	args, err := testutils.GetFakeCmdArgs()
	if err != nil {
		return
	}
	defer os.Exit(0)

	switch args[0] {
	case "error":
		fmt.Fprint(os.Stderr, "Error requested in fake screen info")
		os.Exit(1)
	case "regular":
		fmt.Println(`
{
  "SPDisplaysDataType" : [
    {
      "_name" : "IntelUHDGraphics",
      "_spdisplays_vram" : "1234 GB",
      "spdisplays_automatic_graphics_switching" : "spdisplays_supported",
      "spdisplays_device-id" : "0xbeef",
      "spdisplays_gmux-version" : "1.0",
      "spdisplays_metalfamily" : "spdisplays_mtlgpufamily",
      "spdisplays_ndrvs" : [
        {
          "_IODisplayEDID" : "{length = 1, bytes = 0xffffffff }",
          "_name" : "Color LCD",
          "_spdisplays_display-product-id" : "abcd",
          "_spdisplays_display-serial-number2" : "0",
          "_spdisplays_display-vendor-id" : "678",
          "_spdisplays_display-week" : "53",
          "_spdisplays_display-year" : "2077",
          "_spdisplays_displayID" : "a73beef4",
          "_spdisplays_displayPath" : "IOService:/AppleACPIPlatformExpert/PCI0@0/AppleACPIPCI/IGPU@2/AppleIntelFramebuffer@0/AppleMCCSControlModule",
          "_spdisplays_displayRegID" : "4231",
          "_spdisplays_edid" : "0xffffffff",
          "_spdisplays_pixels" : "1920 x 1080",
          "_spdisplays_resolution" : "1920 x 1080 @ 59.90Hz",
          "spdisplays_ambient_brightness" : "spdisplays_yes",
          "spdisplays_connection_type" : "spdisplays_internal",
          "spdisplays_depth" : "CGSThirtytwoBitColor",
          "spdisplays_display_type" : "spdisplays_built-in_retinaLCD",
          "spdisplays_main" : "spdisplays_yes",
          "spdisplays_mirror" : "spdisplays_off",
          "spdisplays_online" : "spdisplays_yes",
          "spdisplays_pixelresolution" : "spdisplays_3072x1920Retina"
        },
		{
          "_IODisplayEDID" : "{length = 1, bytes = 0xffffffff }",
          "_name" : "Color OLED",
          "_spdisplays_display-product-id" : "dcba",
          "_spdisplays_display-serial-number2" : "0",
          "_spdisplays_display-vendor-id" : "678",
          "_spdisplays_display-week" : "42",
          "_spdisplays_display-year" : "2042",
          "_spdisplays_displayID" : "a73beef4",
          "_spdisplays_displayPath" : "IOService:/AppleACPIPlatformExpert/PCI0@0/AppleACPIPCI/IGPU@2/AppleIntelFramebuffer@0/AppleMCCSControlModule",
          "_spdisplays_displayRegID" : "4231",
          "_spdisplays_edid" : "0xffffffff",
          "_spdisplays_pixels" : "800 x 600",
          "_spdisplays_resolution" : "800 x 600 @ 60Hz",
          "spdisplays_ambient_brightness" : "spdisplays_no",
          "spdisplays_connection_type" : "spdisplays_internal",
          "spdisplays_depth" : "CGSSixtyfourBitColor",
          "spdisplays_display_type" : "spdisplays_built-in_retinaOLED",
          "spdisplays_main" : "spdisplays_no",
          "spdisplays_mirror" : "spdisplays_on",
          "spdisplays_online" : "spdisplays_no",
          "spdisplays_pixelresolution" : "spdisplays_1920x1080Retina"
        }
      ],
      "spdisplays_revision-id" : "0x0002",
      "spdisplays_vendor" : "Intel",
      "spdisplays_vram_shared" : "1234 GB",
      "sppci_bus" : "spdisplays_builtin",
      "sppci_device_type" : "spdisplays_gpu",
      "sppci_model" : "Intel UHD Graphics"
    },
    {
      "_name" : "IntelUHDGraphics2",
      "spdisplays_automatic_graphics_switching" : "spdisplays_supported",
      "spdisplays_device-id" : "0x5432",
      "spdisplays_efi-version" : "11.11.111",
      "spdisplays_gmux-version" : "1.0.0",
      "spdisplays_metalfamily" : "spdisplays_mtlgpufamilymac2",
      "spdisplays_optionrom-version" : "113-D32206U1-019",
      "spdisplays_pcie_width" : "x32",
      "spdisplays_revision-id" : "0xf000",
      "spdisplays_rom-revision" : "111-11111-111",
      "spdisplays_vbios-version" : "010-1111111-010",
      "spdisplays_vendor" : "Intel",
      "spdisplays_vram" : "48 GB",
      "sppci_bus" : "spdisplays_pcie_device",
      "sppci_device_type" : "spdisplays_gpu",
      "sppci_model" : "Intel UHD Graphics 2"
    }
  ]
}`)
	case "bad":
		fmt.Println(`
{"SP]DisplayDatave-id": "0xbeef""" "spdilays_mtlgpufamily"
      "spdisp>laysrvs[
          "_IODisplayEDID" : "length = 1, bytes = 0xffffffff },
          "_namolor LCD",
          "_sp  "spdisplays_amhtnes" :pdisplays_yes",
          spdis_internal,<
          "spdisplays_de}}}}}]]]])))))
      sppc|i_bus" : "spdislays_builtin",
      "pci_ice_tray"
  ]`)
	case "bad values":
		fmt.Println(`
{
  "SPDisplaysDataType" : [
    {
      "_name" : "IntelUHDGraphics",
      "_spdisplays_vram" : "1234 GB",
      "spdisplays_automatic_graphics_switching" : "spdisplays_supported",
      "spdisplays_device-id" : "0xbeef",
      "spdisplays_gmux-version" : "1.0",
      "spdisplays_metalfamily" : "spdisplays_mtlgpufamily",
      "spdisplays_ndrvs" : [
        {
          "_name": "eman_",
          "_spdisplays_pixels" : 2073600,
          "_spdisplays_resolution" : "one by one 1HZ",
          "spdisplays_online" : "spdisplays_yes",
          "spdisplays_pixelresolution" : "1 by 1"
        }
      ],
      "spdisplays_revision-id" : "0x0002",
      "spdisplays_vendor" : "Intel",
      "spdisplays_vram_shared" : "1234 GB",
      "sppci_bus" : "spdisplays_builtin",
      "sppci_device_type" : "spdisplays_gpu",
      "sppci_model" : "Intel UHD Graphics"
    },
    {
      "_name" : "IntelUHDGraphics2",
      "spdisplays_automatic_graphics_switching" : "spdisplays_supported",
      "spdisplays_device-id" : "0x5432",
      "spdisplays_efi-version" : "11.11.111",
      "spdisplays_gmux-version" : "1.0.0",
      "spdisplays_metalfamily" : "spdisplays_mtlgpufamilymac2",
      "spdisplays_optionrom-version" : "113-D32206U1-019",
      "spdisplays_pcie_width" : "x32",
      "spdisplays_revision-id" : "0xf000",
      "spdisplays_rom-revision" : "111-11111-111",
      "spdisplays_vbios-version" : "010-1111111-010",
      "spdisplays_vendor" : "Intel",
      "spdisplays_vram" : "48 GB",
      "sppci_bus" : "spdisplays_pcie_device",
      "sppci_device_type" : "spdisplays_gpu",
      "sppci_model" : "Intel UHD Graphics 2"
    }
  ]
}`)
	case "":
		fallthrough
	case "missing":
		os.Exit(0)
	}
}

func TestParsePListDictDarwin(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input   string
		wantErr bool
	}{
		// Positive test cases
		"Simple string gets parsed": {
			input: "<key>100 + 200</key> <string>300</string>",
		},

		"Simple empty string gets parsed": {
			input: "<key>nothing</key><string/>",
		},

		"Simple integer gets parsed": {
			input: "<key>100 + 200</key> <integer>300</integer>",
		},

		"Simple multiple keys get parsed": {
			input: "<key>1 + 2</key><string>three</string>\n<key>vv+vv</key><string>ww</string>",
		},

		"Simple empty array is empty": {
			input: "<key>nothing</key><array/>",
		},

		"Simple nested array is nested": {
			input: "<key>nothing nested</key><array><array/></array>",
		},

		"Simple array contains values": {
			input: "<key>arr</key><array><string>I am string</string><integer>3000</integer></array>",
		},

		"Simple empty dict is empty": {
			input: "",
		},

		"Simple empty nested dict is empty": {
			input: "<key>nothing</key><dict/>",
		},

		"Nested data is parsed correctly": {
			input: `<key>red</key><array><dict><key>color</key><integer>43210</integer></dict><string>0xff0000</string></array>
					<key>green</key><array><array><integer>01234</integer></array><string>0x00ff00</string></array>
					<key>blue</key><dict><key>color</key><string>0x0000ff</string></dict>
					`,
		},

		// Negative test cases
		"Invalid integer errors": {
			input:   "<key>100 + 200</key> <integer>three hundred</integer>",
			wantErr: true,
		},

		"Invalid value type errors": {
			input:   "<key>1 + 4</key> <str>five</str>",
			wantErr: true,
		},

		"Invalid value type end errors": {
			input:   "<key>1 + 4</key> <string>five</str>",
			wantErr: true,
		},

		"Array containing key errors": {
			input:   "</key><array><key>key</key></array>",
			wantErr: true,
		},

		"Array containing unexpected token errors": {
			input:   "</key><array>[\"You, have, been, misinformed\"]</array>",
			wantErr: true,
		},

		"Array containing unexpected token type errors": {
			input:   "</key><array></called></array>",
			wantErr: true,
		},

		// Negative key related test cases
		"Invalid key start token errors": {
			input:   "<kingdom>blade</key> <string>X</string>",
			wantErr: true,
		},

		"Invalid key end token errors": {
			input:   "<key>kingdom</blade> <string>X</string>",
			wantErr: true,
		},

		"Duplicate key errors": {
			input:   "<key>key</key><string>a</string>\n<key>key</key><string>b</string>",
			wantErr: true,
		},

		"Unexpected key after key errors": {
			input:   "<key>b</key>\n<key>a</key><string>:(</string>",
			wantErr: true,
		},

		"Unexpected token before key errors": {
			input:   "I am an unexpected token<string>:(</string>",
			wantErr: true,
		},

		"Unexpected token after key errors": {
			input:   "<key>:(</key>I am an unexpected token",
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tc.input = "<dict>" + tc.input + "</dict>"
			r := strings.NewReader(tc.input)

			dec := xml.NewDecoder(r)

			startT, err := dec.Token()
			require.NoError(t, err, "setup: failed to start parsing: %v", err)
			require.NotNil(t, startT, "setup: unexpected EOF")

			start, ok := startT.(xml.StartElement)
			require.True(t, ok, "setup: first xml token was not a start token")

			got, err := hardware.ParsePListDict(start, dec)

			if tc.wantErr {
				require.Error(t, err, "ParsePListDict should return an error and didn’t")
				return
			}
			require.NoError(t, err, "ParsePListDict should not return an error")

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			assert.Equal(t, want, got, "ParsePListDict should return expected information")
		})
	}
}

func TestParseDiskDictDarwin(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input   map[string]any
		wantErr bool

		logs map[slog.Level]uint
	}{
		// Positive test cases
		"Disk and partitions gets parsed": {
			input: map[string]any{
				"DeviceIdentifier": "disk0",

				"Size": 40960000,
				"Partitions": []any{
					map[string]any{
						"DeviceIdentifier": "disk0s0",
						"Size":             10240000,
					},
					map[string]any{
						"DeviceIdentifier": "disk0s1",
						"Size":             30720000,
					},
				},
			},
		},

		"Disk no partitions gets parsed and warns no partitions": {
			input: map[string]any{
				"DeviceIdentifier": "disk0",
				"Size":             40960000,
			},

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Disk negative size warns": {
			input: map[string]any{
				"DeviceIdentifier": "disk0",

				"Size": -4096,
				"Partitions": []any{
					map[string]any{
						"DeviceIdentifier": "disk0s0",
						"Size":             40960000,
					},
				},
			},

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Partitions negative size warns": {
			input: map[string]any{
				"DeviceIdentifier": "disk0",

				"Size": 40960000,
				"Partitions": []any{
					map[string]any{
						"DeviceIdentifier": "disk0s0",
						"Size":             -1024,
					},
					map[string]any{
						"DeviceIdentifier": "disk0s0",
						"Size":             -30720000,
					},
				},
			},

			logs: map[slog.Level]uint{
				slog.LevelWarn: 2,
			},
		},

		"Partitions size arent integral warns": {
			input: map[string]any{
				"DeviceIdentifier": "disk0",

				"Size": 40960000,
				"Partitions": []any{
					map[string]any{
						"DeviceIdentifier": "disk0s0",
						"Size":             "one",
					},
					map[string]any{
						"DeviceIdentifier": "disk0s0",
						"Size":             "three",
					},
				},
			},

			logs: map[slog.Level]uint{
				slog.LevelWarn: 2,
			},
		},

		"Partitions isnt array warns": {
			input: map[string]any{
				"DeviceIdentifier": "disk0",

				"Size": 40960000,
				"Partitions": map[string]any{
					"DeviceIdentifier": "disk0s0",
					"Size":             "one",
				},
			},

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Partitions arent dicts warns": {
			input: map[string]any{
				"DeviceIdentifier": "disk0",

				"Size": 40960000,
				"Partitions": []any{
					"disk0s0",
					"disk0s0",
				},
			},

			logs: map[slog.Level]uint{
				slog.LevelWarn: 2,
			},
		},

		"Disk size missing warns": {
			input: map[string]any{
				"DeviceIdentifier": "disk0",

				"Partitions": []any{
					map[string]any{
						"DeviceIdentifier": "disk0s0",
						"Size":             4096,
					},
				},
			},

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		// Lone error case
		"Virtual disk errors": {
			input: map[string]any{
				"DeviceIdentifier": "disk0",

				"Size":               4096,
				"APFSPhysicalStores": []any{},
				"Partitions":         []any{},
			},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			l := testutils.NewMockHandler(slog.LevelDebug)
			got, err := hardware.ParseDiskDict(tc.input, false, slog.New(&l))

			if tc.wantErr {
				require.Error(t, err, "ParsePListDict should return an error and didn’t")
				return
			}
			require.NoError(t, err, "ParsePListDict should not return an error")

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			assert.Equal(t, want, got, "ParsePListDict should return expected information")

			if !l.AssertLevels(t, tc.logs) {
				l.OutputLogs(t)
			}
		})
	}
}
