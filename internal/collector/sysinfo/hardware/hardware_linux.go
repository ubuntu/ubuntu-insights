package hardware

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ubuntu/ubuntu-insights/internal/cmdutils"
	"github.com/ubuntu/ubuntu-insights/internal/fileutils"
)

type options struct {
	root       string
	cpuInfoCmd []string
	lsblkCmd   []string
	screenCmd  []string
	log        *slog.Logger
}

// defaultOptions returns options for when running under a normal environment.
func defaultOptions() *options {
	return &options{
		root:       "/",
		cpuInfoCmd: []string{"lscpu", "-J"},
		lsblkCmd:   []string{"lsblk", "-o", "NAME,SIZE,TYPE", "--tree", "-J"},
		screenCmd:  []string{"xrandr"},
		log:        slog.Default(),
	}
}

// collectProduct reads sysfs to find information about the system.
func (s Collector) collectProduct() (product, error) {
	info := product{
		"Vendor": fileutils.ReadFileLogError(filepath.Join(s.opts.root, "sys/class/dmi/id/sys_vendor"), s.opts.log),
		"Name":   fileutils.ReadFileLogError(filepath.Join(s.opts.root, "sys/class/dmi/id/product_name"), s.opts.log),
		"Family": fileutils.ReadFileLogError(filepath.Join(s.opts.root, "sys/class/dmi/id/product_family"), s.opts.log),
	}

	for k, v := range info {
		if strings.ContainsRune(v, '\n') {
			s.opts.log.Warn(fmt.Sprintf("product %s contains invalid value", k))
			info[k] = ""
		}
	}

	return info, nil
}

// collectCPU uses lscpu to collect information about the CPUs.
func (s Collector) collectCPU() (info cpu, err error) {
	defer func() {
		if err == nil && len(info) == 0 {
			err = fmt.Errorf("no CPU information found")
		}
	}()

	stdout, stderr, err := cmdutils.RunWithTimeout(context.Background(), 15*time.Second, s.opts.cpuInfoCmd[0], s.opts.cpuInfoCmd[1:]...)
	if err != nil {
		return nil, fmt.Errorf("failed to run lscpu: %v", err)
	}
	if stderr.Len() > 0 {
		s.opts.log.Info("lscpu output to stderr", "stderr", stderr)
	}

	type lscpu struct {
		Lscpu []lscpuEntry `json:"lscpu"`
	}
	var result = &lscpu{}
	err = fileutils.ParseJSON(stdout, result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CPU json: %v", err)
	}

	return s.populateCPUInfo(result.Lscpu, cpu{}), nil
}

// usedCPUFields is a set that defines what json fields we want.
var usedCPUFields = map[string]struct{}{
	"CPU(s):":             {},
	"Socket(s):":          {},
	"Core(s) per socket:": {},
	"Thread(s) per core:": {},
	"Architecture:":       {},
	"Vendor ID:":          {},
	"Model name:":         {},
}

type lscpuEntry struct {
	Field    string       `json:"field"`
	Data     string       `json:"data"`
	Children []lscpuEntry `json:"children,omitempty"`
}

// populateCPUInfo recursively searches the lscpu JSON for desired fields.
func (s Collector) populateCPUInfo(entries []lscpuEntry, info cpu) cpu {
	for _, entry := range entries {
		if _, ok := usedCPUFields[entry.Field]; ok {
			info[entry.Field] = entry.Data
		}

		if len(entry.Children) > 0 {
			s.populateCPUInfo(entry.Children, info)
		}
	}

	return info
}

// gpuSymlinkRegex matches the name of a GPU card folder.
var gpuSymlinkRegex = regexp.MustCompile("^card[0-9]+$")

// collectGPUs uses sysfs to collect information about the GPUs.
func (s Collector) collectGPUs() (gpus []gpu, err error) {
	defer func() {
		if err == nil && len(gpus) == 0 {
			err = fmt.Errorf("no GPU information found")
		}
	}()

	// Using ReadDir instead of WalkDir since we don't want recursive directories.
	ds, err := os.ReadDir(filepath.Join(s.opts.root, "sys/class/drm"))
	if err != nil {
		return nil, fmt.Errorf("failed to read GPU directory in sysfs: %v", err)
	}

	for _, d := range ds {
		n := d.Name()

		if !gpuSymlinkRegex.MatchString(n) {
			continue
		}

		gpu, err := s.collectGPU(n)
		if err != nil {
			s.opts.log.Warn("failed to get GPU info", "GPU", n, "error", err)
			continue
		}

		gpus = append(gpus, gpu)
	}

	return gpus, nil
}

// collectGPU handles gathering information for a single GPU.
func (s Collector) collectGPU(card string) (info gpu, err error) {
	cardDir, err := filepath.EvalSymlinks(filepath.Join(s.opts.root, "sys/class/drm", card))
	if err != nil {
		return nil, fmt.Errorf("failed to follow %s symlink: %v", card, err)
	}

	devDir, err := filepath.EvalSymlinks(filepath.Join(cardDir, "device"))
	if err != nil {
		return nil, fmt.Errorf("failed to follow %s device symlink: %v", card, err)
	}

	info = gpu{}
	info["Vendor"] = fileutils.ReadFileLogError(filepath.Join(devDir, "vendor"), s.opts.log)
	info["Name"] = fileutils.ReadFileLogError(filepath.Join(devDir, "label"), s.opts.log)

	driverLink, err := os.Readlink(filepath.Join(devDir, "driver"))
	if err != nil {
		s.opts.log.Warn("failed to get GPU driver", "GPU", card, "error", err)
		return info, nil
	}
	info["Driver"] = filepath.Base(driverLink)

	for k, v := range info {
		if strings.ContainsRune(v, '\n') {
			s.opts.log.Warn(fmt.Sprintf("GPU info contains invalid value for %s", k), "GPU", card)
			info[k] = ""
		}
	}

	return info, nil
}

// usedMemFields is a set which defines which fields from meminfo we want.
var usedMemFields = map[string]struct{}{
	"MemTotal": {},
}

// Lines are in the form `key`:   `bytes` (`unit`).
// For example: "MemTotal: 123 kb" or "MemTotal:   421".
var meminfoRegex = regexp.MustCompile(`^([^\s:]+):\s*([0-9]+)(?:\s+([^\s]+))?\s*$`)

// collectMemory uses meminfo to collect information about RAM.
func (s Collector) collectMemory() (info memory, err error) {
	defer func() {
		if err == nil && len(info) == 0 {
			err = fmt.Errorf("no Memory information found")
		}
	}()

	f, err := os.ReadFile(filepath.Join(s.opts.root, "proc/meminfo"))
	if err != nil {
		return nil, fmt.Errorf("failed to read meminfo: %v", err)
	}

	info = memory{}
	lines := strings.Split(string(f), "\n")
	for i, l := range lines {
		if l == "" {
			continue
		}

		m := meminfoRegex.FindStringSubmatch(l)
		if m == nil {
			s.opts.log.Warn("meminfo contains invalid line", "line", l, "linenum", i)
			continue
		}

		if _, ok := usedMemFields[m[1]]; !ok {
			continue
		}

		v, err := strconv.Atoi(m[2])
		if err != nil {
			s.opts.log.Warn("meminfo value was not an integer", "value", v, "error", err, "linenum", i)
			continue
		}

		info[m[1]], err = fileutils.ConvertUnitToBytes(m[3], v)
		if err != nil {
			s.opts.log.Warn("meminfo had invalid unit", "unit", m[3], "error", err, "linenum", i)
			continue
		}
	}

	return info, nil
}

type lsblkEntry struct {
	Name     string       `json:"name"`
	Size     string       `json:"size"`
	Type     string       `json:"type"`
	Children []lsblkEntry `json:"children,omitempty"`
}

// populateBlkInfo parses lsblkEntries to diskInfo structs.
func (s Collector) populateBlkInfo(entries []lsblkEntry) []disk {
	info := []disk{}

	for _, e := range entries {
		switch strings.ToLower(e.Type) {
		case "disk":
			info = append(info, disk{
				Name:       e.Name,
				Size:       e.Size,
				Partitions: s.populateBlkInfo(e.Children),
			})
		case "part":
			info = append(info, disk{
				Name:       e.Name,
				Size:       e.Size,
				Partitions: []disk{},
			})
		}
	}

	return info
}

// collectBlocks uses lsblk to collect information about Blocks.
func (s Collector) collectDisks() (info []disk, err error) {
	defer func() {
		if err == nil && len(info) == 0 {
			err = fmt.Errorf("no Block information found")
		}
	}()

	stdout, stderr, err := cmdutils.RunWithTimeout(context.Background(), 15*time.Second, s.opts.lsblkCmd[0], s.opts.lsblkCmd[1:]...)
	if err != nil {
		return nil, fmt.Errorf("failed to run lsblk: %v", err)
	}
	if stderr.Len() > 0 {
		s.opts.log.Info("lsblk output to stderr", "stderr", stderr)
	}

	type lsblk struct {
		Lsblk []lsblkEntry `json:"blockdevices"`
	}
	var result = &lsblk{}
	err = fileutils.ParseJSON(stdout, result)
	if err != nil {
		return nil, fmt.Errorf("failed to convert json to a valid lsblk struct: %v", err)
	}

	return s.populateBlkInfo(result.Lsblk), nil
}

// This regex matches the name, primary status, real resolution, and physical size from xrandr.
// For example: "HDMI-0 connected 1234x567+50+0 598mm x 336mm" matches and has "HDMI-0", "", "1234x567", "598mm x 336mm".
// Or: "eDP-1-1 connected primary 1x1+0+0 1mm x 1mm" matches and has "eDP-1-1", "primary", "1x1", "1mm x 1mm".
// However: "HDMI-1 disconnected 1x1+0+0 1mm x 1mm" does not match.
var screenHeaderRegex = regexp.MustCompile(`(?m)^(\S+)\s+connected\s+(?:(primary)\s+)?([0-9]+x[0-9]+).*?([0-9]+mm\s+x\s+[0-9]+mm).*$`)

// This regex matches the resolution and current refresh rate from xrandr.
// For example: "   1920x1080  60.00 100.00+ 74.97*" matches and has "1920x1080", "74.97".
// Or: "720x480 60.00*+ 120.00" matches and has "720x480", "60.00".
// However: "720x480 60.00+ 120.00" does not match.
var screenConfigRegex = regexp.MustCompile(`(?m)^\s*([0-9]+x[0-9]+)\s.*?([0-9]+\.[0-9]+)\+?\*\+?.*$`)

// collectScreens uses xrandr to collect information about screens.
func (s Collector) collectScreens() (info []screen, err error) {
	defer func() {
		if err == nil && len(info) == 0 {
			err = fmt.Errorf("no Screen information found")
		}
	}()

	stdout, stderr, err := cmdutils.RunWithTimeout(context.Background(), 15*time.Second, s.opts.screenCmd[0], s.opts.screenCmd[1:]...)
	if err != nil {
		return nil, fmt.Errorf("failed to run xrandr: %v", err)
	}
	if stderr.Len() > 0 {
		s.opts.log.Info("xrandr output to stderr", "stderr", stderr)
	}

	data := stdout.String()
	screens := screenHeaderRegex.Split(data, -1)
	headers := screenHeaderRegex.FindAllStringSubmatch(data, -1)
	if len(headers) == 0 {
		// setting error is handled by decorator.
		return nil, nil
	}

	info = make([]screen, 0, len(headers))

	for i, header := range headers {
		v := screenConfigRegex.FindStringSubmatch(screens[i+1])

		if len(v) < 3 {
			s.opts.log.Warn("xrandr screen info malformed", "screen", header[1])
			continue
		}

		info = append(info, screen{
			Name:               header[1],
			PhysicalResolution: header[3],
			Size:               header[4],

			Resolution:  v[1],
			RefreshRate: v[2],
		})
	}

	return info, nil
}
