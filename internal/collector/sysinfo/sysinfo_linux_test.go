package sysinfo_test

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo"
	"github.com/ubuntu/ubuntu-insights/internal/testutils"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
	}{
		"Instantiate a sys info manager": {},
	}
	for name, _ := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			s := sysinfo.New(sysinfo.WithRoot("/myspecialroot"))

			require.NotEmpty(t, s, "sysinfo manager has custom fields")
		})
	}
}

func TestCollect(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		root         string
		cpuInfo      string
		missingFiles []string

		logs    map[slog.Level]uint
		wantErr bool
	}{
		"Regular hardware information": {
			root:    "regular",
			cpuInfo: "regular",
		},

		"Missing Product information": {
			root:    "regular",
			cpuInfo: "regular",
			missingFiles: []string{
				"sys/class/dmi/id/product_family",
				"sys/class/dmi/id/product_name",
				"sys/class/dmi/id/sys_vendor",
			},

			logs: map[slog.Level]uint{
				slog.LevelWarn: 3,
			},
		},

		"Missing CPU information": {
			root:    "regular",
			cpuInfo: "missing",

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Missing GPUs": {
			root:    "regular",
			cpuInfo: "regular",
			missingFiles: []string{
				"sys/class/drm/card0",
				"sys/class/drm/card1",
			},

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Missing GPU information": {
			root:    "regular",
			cpuInfo: "regular",
			missingFiles: []string{
				"sys/class/drm/c0/d0/driver",
				"sys/class/drm/c0/d0/label",
				"sys/class/drm/c0/d0/vendor",
			},

			logs: map[slog.Level]uint{
				slog.LevelWarn: 3,
			},
		},

		"Missing Memory information": {
			root:         "regular",
			cpuInfo:      "regular",
			missingFiles: []string{"proc/meminfo"},

			logs: map[slog.Level]uint{
				slog.LevelWarn: 1,
			},
		},

		"Missing hardware information is empty": {
			root:    "withoutinfo",
			cpuInfo: "",
			logs: map[slog.Level]uint{
				slog.LevelWarn: 6,
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tmp := t.TempDir()
			err := testutils.CopyDir("testdata/linuxfs", tmp)
			if err != nil {
				fmt.Printf("Setup: failed to copy testdata directory: %v\n", err)
				t.FailNow()
			}

			root := filepath.Join(tmp, "linuxfs", tc.root)
			for _, f := range tc.missingFiles {
				err := os.Remove(filepath.Join(root, f))
				if err != nil {
					fmt.Printf("Setup: failed to remove file %s: %v\n", f, err)
					t.FailNow()
				}
			}

			l := testutils.NewMockHandler()

			options := []sysinfo.Options{
				sysinfo.WithRoot(root),
				sysinfo.WithLogger(&l),
			}

			if tc.cpuInfo != "-" {
				cmdArgs := []string{"env", "GO_WANT_HELPER_PROCESS=1", os.Args[0], "-test.run=TestMockCPUList", "--"}
				cmdArgs = append(cmdArgs, tc.cpuInfo)
				options = append(options, sysinfo.WithCpuInfo(cmdArgs))
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

			l.AssertLevels(t, tc.logs)
		})
	}
}

func TestMockCPUList(_ *testing.T) {
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
	case "exit 1":
		fmt.Fprint(os.Stderr, "Error requested in Mock cpulist")
		os.Exit(1)
	case "regular":
		fmt.Println(`{
   "lscpu": [
      {
         "field": "Architecture:",
         "data": "x86_64",
         "children": [
            {
               "field": "CPU op-mode(s):",
               "data": "32-bit, 64-bit"
            },{
               "field": "Address sizes:",
               "data": "39 bits physical, 48 bits virtual"
            },{
               "field": "Byte Order:",
               "data": "Little Endian"
            }
         ]
      },{
         "field": "CPU(s):",
         "data": "12",
         "children": [
            {
               "field": "On-line CPU(s) list:",
               "data": "0-11"
            }
         ]
      },{
         "field": "Vendor ID:",
         "data": "GenuineIntel",
         "children": [
            {
               "field": "Model name:",
               "data": "Intel(R) Core(TM) i7-8750H CPU @ 2.20GHz",
               "children": [
                  {
                     "field": "CPU family:",
                     "data": "6"
                  },{
                     "field": "Model:",
                     "data": "158"
                  },{
                     "field": "Thread(s) per core:",
                     "data": "2"
                  },{
                     "field": "Core(s) per socket:",
                     "data": "6"
                  },{
                     "field": "Socket(s):",
                     "data": "1"
                  },{
                     "field": "Stepping:",
                     "data": "10"
                  },{
                     "field": "CPU(s) scaling MHz:",
                     "data": "22%"
                  },{
                     "field": "CPU max MHz:",
                     "data": "4100.0000"
                  },{
                     "field": "CPU min MHz:",
                     "data": "800.0000"
                  },{
                     "field": "BogoMIPS:",
                     "data": "4399.99"
                  },{
                     "field": "Flags:",
                     "data": "fpu vme de pse tsc msr pae mce cx8 apic sep mtrr pge mca cmov pat pse36 clflush dts acpi mmx fxsr sse sse2 ss ht tm pbe syscall nx pdpe1gb rdtscp lm constant_tsc art arch_perfmon pebs bts rep_good nopl xtopology nonstop_tsc cpuid aperfmperf pni pclmulqdq dtes64 monitor ds_cpl vmx est tm2 ssse3 sdbg fma cx16 xtpr pdcm pcid sse4_1 sse4_2 x2apic movbe popcnt tsc_deadline_timer aes xsave avx f16c rdrand lahf_lm abm 3dnowprefetch cpuid_fault epb pti ssbd ibrs ibpb stibp tpr_shadow flexpriority ept vpid ept_ad fsgsbase tsc_adjust bmi1 avx2 smep bmi2 erms invpcid mpx rdseed adx smap clflushopt intel_pt xsaveopt xsavec xgetbv1 xsaves dtherm ida arat pln pts hwp hwp_notify hwp_act_window hwp_epp vnmi md_clear flush_l1d arch_capabilities"
                  }
               ]
            }
         ]
      },{
         "field": "Caches (sum of all):",
         "data": null,
         "children": [
            {
               "field": "L1d:",
               "data": "192 KiB (6 instances)"
            },{
               "field": "L1i:",
               "data": "192 KiB (6 instances)"
            },{
               "field": "L2:",
               "data": "1.5 MiB (6 instances)"
            },{
               "field": "L3:",
               "data": "9 MiB (1 instance)"
            }
         ]
      }
   ]
}`)
	case "missing":
		os.Exit(0)

	}
}
