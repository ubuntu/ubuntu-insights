package hardware

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ubuntu/ubuntu-insights/internal/cmdutils"
	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/platform"
	"github.com/ubuntu/ubuntu-insights/internal/fileutils"
)

type platformOptions struct {
	root       string
	cpuInfoCmd []string
	lsblkCmd   []string
	screenCmd  []string
}

// defaultOptions returns options for when running under a normal environment.
func defaultPlatformOptions() platformOptions {
	return platformOptions{
		root:       "/",
		cpuInfoCmd: []string{"lscpu", "-J"},
		lsblkCmd:   []string{"lsblk", "-o", "NAME,SIZE,TYPE", "--tree", "-J"},
		screenCmd:  []string{"xrandr"},
	}
}

// collectProduct reads sysfs to find information about the system.
func (h Collector) collectProduct(pi platform.Info) (product, error) {
	if pi.WSL.SubsystemVersion != 0 {
		h.log.Debug("skipping product info collection on WSL")
		return product{}, nil
	}

	info := product{
		Vendor: fileutils.ReadFileLogError(filepath.Join(h.platform.root, "sys/class/dmi/id/sys_vendor"), h.log),
		Name:   fileutils.ReadFileLogError(filepath.Join(h.platform.root, "sys/class/dmi/id/product_name"), h.log),
		Family: fileutils.ReadFileLogError(filepath.Join(h.platform.root, "sys/class/dmi/id/product_family"), h.log),
	}

	if strings.ContainsRune(info.Vendor, '\n') {
		h.log.Warn("product vendor contains invalid value")
		info.Vendor = ""
	}
	if strings.ContainsRune(info.Name, '\n') {
		h.log.Warn("product name contains invalid value")
		info.Name = ""
	}
	if strings.ContainsRune(info.Family, '\n') {
		h.log.Warn("product family contains invalid value")
		info.Family = ""
	}

	return info, nil
}

// collectCPU uses lscpu to collect information about the CPUs.
func (h Collector) collectCPU() (cpu, error) {
	stdout, stderr, err := cmdutils.RunWithTimeout(context.Background(), 15*time.Second, h.platform.cpuInfoCmd[0], h.platform.cpuInfoCmd[1:]...)
	if err != nil {
		return cpu{}, fmt.Errorf("failed to run lscpu: %v", err)
	}
	if stderr.Len() > 0 {
		h.log.Info("lscpu output to stderr", "stderr", stderr)
	}

	type lscpu struct {
		Lscpu []lscpuEntry `json:"lscpu"`
	}
	var result = &lscpu{}
	err = fileutils.ParseJSON(stdout, result)
	if err != nil {
		return cpu{}, fmt.Errorf("failed to parse CPU json: %v", err)
	}

	data := h.populateCPUInfo(result.Lscpu, map[string]string{})

	sockets, err := strconv.ParseUint(data["Socket(s):"], 10, 64)
	if err != nil {
		h.log.Warn("CPU info contained invalid sockets", "value", data["Socket(s):"])
		sockets = 0
	}
	cores, err := strconv.ParseUint(data["Core(s) per socket:"], 10, 64)
	if err != nil {
		h.log.Warn("CPU info contained invalid cores per socket", "value", data["Core(s) per socket:"])
		cores = 0
	}
	threads, err := strconv.ParseUint(data["Thread(s) per core:"], 10, 64)
	if err != nil {
		h.log.Warn("CPU info contained invalid threads per core", "value", data["Thread(s) per core:"])
		threads = 0
	}
	cpus, err := strconv.ParseUint(data["CPU(s):"], 10, 64)
	if err != nil {
		h.log.Warn("CPU info contained invalid cpus", "value", data["CPU(s):"])
		cpus = threads * cores * sockets
	}

	arch := h.arch
	if v, ok := data["Architecture:"]; ok {
		arch = v
	}

	return cpu{
		Name:    data["Model name:"],
		Vendor:  data["Vendor ID:"],
		Arch:    arch,
		Cpus:    cpus,
		Sockets: sockets,
		Cores:   cores,
		Threads: threads,
	}, nil
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
func (h Collector) populateCPUInfo(entries []lscpuEntry, info map[string]string) map[string]string {
	for _, entry := range entries {
		if _, ok := usedCPUFields[entry.Field]; ok {
			info[entry.Field] = entry.Data
		}

		if len(entry.Children) > 0 {
			h.populateCPUInfo(entry.Children, info)
		}
	}

	return info
}

// gpuSymlinkRegex matches the name of a GPU card folder.
var gpuSymlinkRegex = regexp.MustCompile("^card[0-9]+$")

// collectGPUs uses sysfs to collect information about the GPUs.
func (h Collector) collectGPUs(pi platform.Info) (gpus []gpu, err error) {
	defer func() {
		if err == nil && len(gpus) == 0 && pi.WSL.SubsystemVersion == 0 {
			err = fmt.Errorf("no GPU information found")
		}
	}()

	if pi.WSL.SubsystemVersion != 0 {
		h.log.Debug("skipping GPU info collection on WSL")
		return []gpu{}, nil
	}

	// Using ReadDir instead of WalkDir since we don't want recursive directories.
	ds, err := os.ReadDir(filepath.Join(h.platform.root, "sys/class/drm"))
	if err != nil {
		return nil, fmt.Errorf("failed to read GPU directory in sysfs: %v", err)
	}

	for _, d := range ds {
		n := d.Name()

		if !gpuSymlinkRegex.MatchString(n) {
			continue
		}

		gpu, err := h.collectGPU(n)
		if err != nil {
			h.log.Warn("failed to get GPU info", "GPU", n, "error", err)
			continue
		}

		gpus = append(gpus, gpu)
	}

	return gpus, nil
}

// collectGPU handles gathering information for a single GPU.
func (h Collector) collectGPU(card string) (info gpu, err error) {
	cardDir, err := filepath.EvalSymlinks(filepath.Join(h.platform.root, "sys/class/drm", card))
	if err != nil {
		return info, fmt.Errorf("failed to follow %s symlink: %v", card, err)
	}

	devDir, err := filepath.EvalSymlinks(filepath.Join(cardDir, "device"))
	if err != nil {
		return info, fmt.Errorf("failed to follow %s device symlink: %v", card, err)
	}

	info.Vendor = fileutils.ReadFileLogError(filepath.Join(devDir, "vendor"), h.log)
	info.Name = fileutils.ReadFileLogError(filepath.Join(devDir, "label"), h.log)

	if strings.ContainsRune(info.Vendor, '\n') {
		h.log.Warn("gpu vendor contains invalid value", "GPU", card)
		info.Vendor = ""
	}
	if strings.ContainsRune(info.Name, '\n') {
		h.log.Warn("gpu name contains invalid value", "GPU", card)
		info.Name = ""
	}

	driverLink, err := os.Readlink(filepath.Join(devDir, "driver"))
	if err != nil {
		h.log.Warn("failed to get GPU driver", "GPU", card, "error", err)
		return info, nil
	}
	info.Driver = filepath.Base(driverLink)

	return info, nil
}

// Lines are in the form `key`:   `bytes` (`unit`).
// For example: "MemTotal: 123 kb" or "MemTotal:   421".
var meminfoRegex = regexp.MustCompile(`^([^\s:]+):\s*([0-9]+)(?:\s+([^\s]+))?\s*$`)

// collectMemory uses meminfo to collect information about RAM.
func (h Collector) collectMemory() (memory, error) {
	f, err := os.ReadFile(filepath.Join(h.platform.root, "proc/meminfo"))
	if err != nil {
		return memory{}, fmt.Errorf("failed to read meminfo: %v", err)
	}

	data := map[string]int{}
	lines := strings.Split(string(f), "\n")
	for i, l := range lines {
		if l == "" {
			continue
		}

		m := meminfoRegex.FindStringSubmatch(l)
		if len(m) != 4 {
			h.log.Warn("meminfo contains invalid line", "line", l, "linenum", i)
			continue
		}

		if m[1] != "MemTotal" {
			continue
		}

		v, err := strconv.Atoi(m[2])
		if err != nil {
			h.log.Warn("meminfo value was not an integer", "value", v, "error", err, "linenum", i)
			break
		}

		data[m[1]], err = fileutils.ConvertUnitToStandard(m[3], v)
		if err != nil {
			h.log.Warn("meminfo had invalid unit", "unit", m[3], "error", err, "linenum", i)
		}
		break
	}

	return memory{
		Total: data["MemTotal"],
	}, nil
}

type lsblkEntry struct {
	Name     string       `json:"name"`
	Size     string       `json:"size"`
	Type     string       `json:"type"`
	Children []lsblkEntry `json:"children,omitempty"`
}

// blockSizeRegex matches a number with unit prefix.
// For example:  " 5.5G " is matched with "5.5" and "G".
var blockSizeRegex = regexp.MustCompile(`^\s*([0-9]+(?:\.[0-9]*)?)\s*([^\s]*)\s*$`)

// populateBlkInfo parses lsblkEntries to diskInfo structs.
func (h Collector) populateBlkInfo(entries []lsblkEntry) []disk {
	getSize := func(s string) uint64 {
		m := blockSizeRegex.FindStringSubmatch(s)
		if len(m) != 3 {
			h.log.Warn("block info contains invalid size", "value", s)
			return 0
		}
		v, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			h.log.Warn("block info contains invalid size", "value", m[1])
			return 0
		}

		r, err := fileutils.ConvertUnitToStandard(m[2], v)
		if err != nil {
			h.log.Warn("block info contains invalid unit", "unit", m[2])
		}
		return uint64(r)
	}

	info := []disk{}
	for _, e := range entries {
		switch strings.ToLower(e.Type) {
		case "disk":
			info = append(info, disk{
				Name:       e.Name,
				Size:       getSize(e.Size),
				Partitions: h.populateBlkInfo(e.Children),
			})
		case "part":
			info = append(info, disk{
				Name:       e.Name,
				Size:       getSize(e.Size),
				Partitions: []disk{},
			})
		}
	}

	return info
}

// collectBlocks uses lsblk to collect information about Blocks.
func (h Collector) collectDisks(pi platform.Info) (info []disk, err error) {
	stdout, stderr, err := cmdutils.RunWithTimeout(context.Background(), 15*time.Second, h.platform.lsblkCmd[0], h.platform.lsblkCmd[1:]...)
	if err != nil && pi.WSL.SubsystemVersion == 1 {
		h.log.Debug("skipping block info collection on WSL1")
		return []disk{}, nil
	}
	// Late defer to avoid WSL 1 case.
	defer func() {
		if err == nil && len(info) == 0 {
			err = fmt.Errorf("no Block information found")
		}
	}()
	if err != nil {
		return nil, fmt.Errorf("failed to run lsblk: %v", err)
	}
	if stderr.Len() > 0 {
		h.log.Info("lsblk output to stderr", "stderr", stderr)
	}

	type lsblk struct {
		Lsblk []lsblkEntry `json:"blockdevices"`
	}
	var result = &lsblk{}
	err = fileutils.ParseJSON(stdout, result)
	if err != nil {
		return nil, fmt.Errorf("failed to convert json to a valid lsblk struct: %v", err)
	}

	return h.populateBlkInfo(result.Lsblk), nil
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
func (h Collector) collectScreens(pi platform.Info) (info []screen, err error) {
	stdout, stderr, err := cmdutils.RunWithTimeout(context.Background(), 15*time.Second, h.platform.screenCmd[0], h.platform.screenCmd[1:]...)
	if err != nil {
		if pi.WSL.SubsystemVersion != 0 {
			h.log.Debug("skipping screen info collection on WSL")
			return []screen{}, nil
		}

		return nil, fmt.Errorf("failed to run xrandr: %v", err)
	}
	if stderr.Len() > 0 {
		h.log.Info("xrandr output to stderr", "stderr", stderr)
	}

	defer func() {
		if err == nil && len(info) == 0 {
			err = fmt.Errorf("no Screen information found")
		}
	}()

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

		if len(v) != 3 || len(header) != 5 {
			h.log.Warn("xrandr screen info malformed", "screen", header[1])
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
