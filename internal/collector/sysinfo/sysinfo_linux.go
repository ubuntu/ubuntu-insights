package sysinfo

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
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

// collectHardware aggregates the data from all the other hardware collect functions.
func (s Manager) collectHardware() (hwInfo hwInfo, err error) {
	hwInfo.Product = s.collectProduct()

	hwInfo.CPU, err = s.collectCPU()
	if err != nil {
		s.opts.log.Warn(fmt.Sprintf("%v", err))
		hwInfo.CPU = map[string]string{}
	}

	hwInfo.GPUs, err = s.collectGPUs()
	if err != nil {
		s.opts.log.Warn(fmt.Sprintf("%v", err))
		hwInfo.GPUs = []map[string]string{}
	}

	hwInfo.Mem, err = s.collectMemory()
	if err != nil {
		s.opts.log.Warn(fmt.Sprintf("%v", err))
		hwInfo.Mem = map[string]int{}
	}

	hwInfo.Blks, err = s.collectBlocks()
	if err != nil {
		s.opts.log.Warn(fmt.Sprintf("%v", err))
		hwInfo.Blks = []diskInfo{}
	}

	hwInfo.Screens, err = s.collectScreens()
	if err != nil {
		s.opts.log.Warn(fmt.Sprintf("%v", err))
		hwInfo.Screens = []screenInfo{}
	}

	return hwInfo, nil
}

// collectSoftware aggregates the data from all the other software collect functions.
func (s Manager) collectSoftware() (swInfo swInfo, err error) {
	return swInfo, nil
}

// collectProduct reads sysfs to find information about the system.
func (s Manager) collectProduct() map[string]string {
	info := map[string]string{
		"Vendor": s.readFileDiscardError(filepath.Join(s.opts.root, "sys/class/dmi/id/sys_vendor")),
		"Name":   s.readFileDiscardError(filepath.Join(s.opts.root, "sys/class/dmi/id/product_name")),
		"Family": s.readFileDiscardError(filepath.Join(s.opts.root, "sys/class/dmi/id/product_family")),
	}

	for k, v := range info {
		if strings.ContainsRune(v, '\n') {
			s.opts.log.Warn(fmt.Sprintf("product %s contains invalid value", k))
			info[k] = ""
		}
	}

	return info
}

// collectCPU uses lscpu to collect information about the CPUs.
func (s Manager) collectCPU() (info map[string]string, err error) {
	defer func() {
		if err == nil && len(info) == 0 {
			err = fmt.Errorf("no CPU information found")
		}
	}()

	stdout, stderr, err := runCmd(context.Background(), s.opts.cpuInfoCmd[0], s.opts.cpuInfoCmd[1:]...)
	if err != nil {
		return nil, fmt.Errorf("failed to run lscpu: %v", err)
	}
	if stderr.Len() > 0 {
		s.opts.log.Info(fmt.Sprintf("lscpu output to stderr: %v", stderr))
	}

	type lscpu struct {
		Lscpu []lscpuEntry `json:"lscpu"`
	}
	var result = &lscpu{}
	err = parseJSON(stdout, result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CPU json: %v", err)
	}

	return s.populateCPUInfo(result.Lscpu, map[string]string{}), nil
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
func (s Manager) populateCPUInfo(entries []lscpuEntry, info map[string]string) map[string]string {
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
func (s Manager) collectGPUs() (gpus []map[string]string, err error) {
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
			s.opts.log.Warn(fmt.Sprintf("failed to get GPU info for %s: %v", n, err))
			continue
		}

		gpus = append(gpus, gpu)
	}

	return gpus, nil
}

// collectGPU handles gathering information for a single GPU.
func (s Manager) collectGPU(card string) (info map[string]string, err error) {
	cardDir, err := filepath.EvalSymlinks(filepath.Join(s.opts.root, "sys/class/drm", card))
	if err != nil {
		return nil, fmt.Errorf("failed to follow %s symlink: %v", card, err)
	}

	devDir, err := filepath.EvalSymlinks(filepath.Join(cardDir, "device"))
	if err != nil {
		return nil, fmt.Errorf("failed to follow %s device symlink: %v", card, err)
	}

	info = map[string]string{}
	info["Vendor"] = s.readFileDiscardError(filepath.Join(devDir, "vendor"))
	info["Name"] = s.readFileDiscardError(filepath.Join(devDir, "label"))

	driverLink, err := os.Readlink(filepath.Join(devDir, "driver"))
	if err != nil {
		s.opts.log.Warn(fmt.Sprintf("failed to get driver for %s: %v", card, err))
		return info, nil
	}
	info["Driver"] = filepath.Base(driverLink)

	for k, v := range info {
		if strings.ContainsRune(v, '\n') {
			s.opts.log.Warn(fmt.Sprintf("gpu %s's %s contains invalid value", card, k))
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
func (s Manager) collectMemory() (info map[string]int, err error) {
	defer func() {
		if err == nil && len(info) == 0 {
			err = fmt.Errorf("no Memory information found")
		}
	}()

	f, err := os.ReadFile(filepath.Join(s.opts.root, "proc/meminfo"))
	if err != nil {
		return nil, fmt.Errorf("failed to read meminfo: %v", err)
	}

	info = map[string]int{}
	lines := strings.Split(string(f), "\n")
	for _, l := range lines {
		if l == "" {
			continue
		}

		m := meminfoRegex.FindStringSubmatch(l)
		if m == nil {
			s.opts.log.Warn(fmt.Sprintf("meminfo contains invalid line: %s", l))
			continue
		}

		if _, ok := usedMemFields[m[1]]; !ok {
			continue
		}

		v, err := strconv.Atoi(m[2])
		if err != nil {
			s.opts.log.Warn(fmt.Sprintf("meminfo value was not an integer: %v", err))
			continue
		}

		info[m[1]] = s.convertUnitToBytes(m[3], v)
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
func (s Manager) populateBlkInfo(entries []lsblkEntry) []diskInfo {
	info := []diskInfo{}

	for _, e := range entries {
		switch strings.ToLower(e.Type) {
		case "disk":
			info = append(info, diskInfo{
				Name:       e.Name,
				Size:       e.Size,
				Partitions: s.populateBlkInfo(e.Children),
			})
		case "part":
			info = append(info, diskInfo{
				Name:       e.Name,
				Size:       e.Size,
				Partitions: []diskInfo{},
			})
		}
	}

	return info
}

// collectBlocks uses lsblk to collect information about Blocks.
func (s Manager) collectBlocks() (info []diskInfo, err error) {
	defer func() {
		if err == nil && len(info) == 0 {
			err = fmt.Errorf("no Block information found")
		}
	}()

	stdout, stderr, err := runCmd(context.Background(), s.opts.lsblkCmd[0], s.opts.lsblkCmd[1:]...)
	if err != nil {
		return nil, fmt.Errorf("failed to run lsblk: %v", err)
	}
	if stderr.Len() > 0 {
		s.opts.log.Info(fmt.Sprintf("lsblk output to stderr: %v", stderr))
	}

	type lsblk struct {
		Lsblk []lsblkEntry `json:"blockdevices"`
	}
	var result = &lsblk{}
	err = parseJSON(stdout, result)
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
func (s Manager) collectScreens() (info []screenInfo, err error) {
	defer func() {
		if err == nil && len(info) == 0 {
			err = fmt.Errorf("no Screen information found")
		}
	}()

	stdout, stderr, err := runCmd(context.Background(), s.opts.screenCmd[0], s.opts.screenCmd[1:]...)
	if err != nil {
		return nil, fmt.Errorf("failed to run xrandr: %v", err)
	}
	if stderr.Len() > 0 {
		s.opts.log.Info(fmt.Sprintf("xrandr output to stderr: %v", stderr))
	}

	data := stdout.String()
	screens := screenHeaderRegex.Split(data, -1)
	headers := screenHeaderRegex.FindAllStringSubmatch(data, -1)
	if len(headers) == 0 {
		// setting error is handled by decorator.
		return nil, nil
	}

	info = make([]screenInfo, 0, len(headers))

	for i, header := range headers {
		v := screenConfigRegex.FindStringSubmatch(screens[i+1])

		if len(v) < 3 {
			s.opts.log.Warn(fmt.Sprintf("xrandr screen info for %s malformed", header[1]))
			continue
		}

		info = append(info, screenInfo{
			Name:               header[1],
			PhysicalResolution: header[3],
			Size:               header[4],

			Resolution:  v[1],
			RefreshRate: v[2],
		})
	}

	return info, nil
}

// readFile returns the data in the file path, trimming whitespace, or "" on error.
func (s Manager) readFileDiscardError(path string) string {
	f, err := os.ReadFile(path)
	if err != nil {
		s.opts.log.Warn(fmt.Sprintf("failed to read file %s: %v", path, err))
		return ""
	}

	return strings.TrimSpace(string(f))
}

// convertUnitToBytes takes a string bytes unit and converts value to bytes.
func (s Manager) convertUnitToBytes(unit string, value int) int {
	switch strings.ToLower(unit) {
	case "":
		fallthrough
	case "b":
		return value
	case "k":
		fallthrough
	case "kb":
		fallthrough
	case "kib":
		return value * 1024
	case "m":
		fallthrough
	case "mb":
		fallthrough
	case "mib":
		return value * 1024 * 1024
	case "g":
		fallthrough
	case "gb":
		fallthrough
	case "gib":
		return value * 1024 * 1024 * 1024
	case "t":
		fallthrough
	case "tb":
		fallthrough
	case "tib":
		return value * 1024 * 1024 * 1024 * 1024
	default:
		s.opts.log.Warn(fmt.Sprintf("unrecognized bytes unit: %s", unit))
		return value
	}
}
