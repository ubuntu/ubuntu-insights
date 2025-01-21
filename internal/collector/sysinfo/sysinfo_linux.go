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

func defaultOptions() *options {
	return &options{
		root:       "/",
		cpuInfoCmd: []string{"lscpu", "-J"},
		lsblkCmd:   []string{"lsblk", "-o", "NAME,SIZE,TYPE", "--tree", "-J"},
		screenCmd:  []string{"xrandr"},
		log:        slog.Default(),
	}
}

type lscpuEntry struct {
	Field    string       `json:"field"`
	Data     string       `json:"data"`
	Children []lscpuEntry `json:"children,omitempty"`
}

type lscpu struct {
	Lscpu []lscpuEntry `json:"lscpu"`
}

type lsblkEntry struct {
	Name     string       `json:"name"`
	Size     string       `json:"size"`
	Type     string       `json:"type"`
	Children []lsblkEntry `json:"children,omitempty"`
}

type lsblk struct {
	Lsblk []lsblkEntry `json:"blockdevices"`
}

// readFile returns the data in <file>, or "" on error.
func (s Manager) readFile(file string) string {
	f, err := os.ReadFile(file)
	if err != nil {
		s.opts.log.Warn(fmt.Sprintf("failed to read file %s: %v", file, err))
		return ""
	}

	return strings.TrimSpace(string(f))
}

func (s Manager) collectProduct() map[string]string {
	return map[string]string{
		"Vendor": s.readFile(filepath.Join(s.opts.root, "sys/class/dmi/id/sys_vendor")),
		"Name":   s.readFile(filepath.Join(s.opts.root, "sys/class/dmi/id/product_name")),
		"Family": s.readFile(filepath.Join(s.opts.root, "sys/class/dmi/id/product_family")),
	}
}

var usedCPUFields = map[string]bool{
	"CPU(s):":             true,
	"Socket(s):":          true,
	"Core(s) per socket:": true,
	"Thread(s) per core:": true,
	"Architecture:":       true,
	"Vendor ID:":          true,
	"Model name:":         true,
}

func (s Manager) populateCPUInfo(entries []lscpuEntry, info *cpuInfo) cpuInfo {
	for _, entry := range entries {
		if usedCPUFields[entry.Field] {
			info.CPU[entry.Field] = entry.Data
		}

		if len(entry.Children) > 0 {
			s.populateCPUInfo(entry.Children, info)
		}
	}

	return *info
}

func (s Manager) collectCPU() cpuInfo {
	info := cpuInfo{CPU: map[string]string{}}

	stdout, stderr, err := runCmd(context.Background(), s.opts.cpuInfoCmd[0], s.opts.cpuInfoCmd[1:]...)
	if err != nil {
		s.opts.log.Warn(fmt.Sprintf("failed to run lscpu: %v", err))
		return info
	}
	if stderr.Len() > 0 {
		s.opts.log.Warn(fmt.Sprintf("lscpu output to stderr: %v", stderr))
	}

	result, err := parseJSON(&stdout, &lscpu{})
	if err != nil {
		s.opts.log.Warn(fmt.Sprintf("failed to parse CPU json: %v", err))
		return info
	}

	lscpu, ok := result.(*lscpu)
	if !ok {
		s.opts.log.Warn(fmt.Sprintf("failed to convert json to a valid lscpu struct: %v", result))
		return info
	}

	return s.populateCPUInfo(lscpu.Lscpu, &info)
}

func (s Manager) collectGPU(card string) (info gpuInfo, err error) {
	cardDir, err := filepath.EvalSymlinks(filepath.Join(s.opts.root, "sys/class/drm", card))
	if err != nil {
		return gpuInfo{}, err
	}

	devDir, err := filepath.EvalSymlinks(filepath.Join(cardDir, "device"))
	if err != nil {
		return gpuInfo{}, err
	}

	info.GPU = make(map[string]string)

	info.GPU["Vendor"] = s.readFile(filepath.Join(devDir, "vendor"))
	info.GPU["Name"] = s.readFile(filepath.Join(devDir, "label"))

	driverLink, err := os.Readlink(filepath.Join(devDir, "driver"))
	if err != nil {
		s.opts.log.Warn(fmt.Sprintf("failed to get driver for %s: %v", card, err))
		return info, nil
	}
	info.GPU["Driver"] = filepath.Base(driverLink)

	return info, nil
}

var gpuSymlinkRegex = regexp.MustCompile("^card[0-9]+$")

func (s Manager) collectGPUs() []gpuInfo {
	gpus := make([]gpuInfo, 0, 2)

	ds, err := os.ReadDir(filepath.Join(s.opts.root, "sys/class/drm"))
	if err != nil {
		s.opts.log.Warn(fmt.Sprintf("failed to read GPU directory in sysfs: %v", err))
		return gpus
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

	if len(gpus) == 0 {
		s.opts.log.Warn("no GPU information found")
	}

	return gpus
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

var usedMemFields = map[string]bool{
	"MemTotal": true,
}

// lines are in the form `key`:   `bytes` (`unit`).
var meminfoRegex = regexp.MustCompile(`^([^\s:]+):\s*([0-9]+)(?:\s+([^\s]+))?\s*$`)

func (s Manager) collectMemory() memInfo {
	info := memInfo{Mem: map[string]int{}}

	f, err := os.ReadFile(filepath.Join(s.opts.root, "proc/meminfo"))
	if err != nil {
		s.opts.log.Warn(fmt.Sprintf("failed to read meminfo: %v", err))
		return info
	}

	lines := strings.Split(string(f), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		m := meminfoRegex.FindStringSubmatch(line)
		if m == nil {
			s.opts.log.Warn(fmt.Sprintf("meminfo contains invalid line: %s", line))
			continue
		}

		if !usedMemFields[m[1]] {
			continue
		}

		val, err := strconv.Atoi(m[2])
		if err != nil {
			s.opts.log.Warn(fmt.Sprintf("meminfo value was not an integer: %v", err))
			continue
		}

		info.Mem[m[1]] = s.convertUnitToBytes(m[3], val)
	}

	return info
}

func (s Manager) populateBlkInfo(entries []lsblkEntry) []diskInfo {
	info := make([]diskInfo, 0, 2)

	for _, entry := range entries {
		switch strings.ToLower(entry.Type) {
		case "disk":
			info = append(info, diskInfo{
				Name:       entry.Name,
				Size:       entry.Size,
				Partitions: s.populateBlkInfo(entry.Children),
			})
		case "part":
			info = append(info, diskInfo{
				Name:       entry.Name,
				Size:       entry.Size,
				Partitions: []diskInfo{},
			})
		default:
			continue
		}
	}

	return info
}

func (s Manager) collectBlocks() []diskInfo {
	stdout, stderr, err := runCmd(context.Background(), s.opts.lsblkCmd[0], s.opts.lsblkCmd[1:]...)
	if err != nil {
		s.opts.log.Warn(fmt.Sprintf("failed to run lsblk: %v", err))
		return []diskInfo{}
	}
	if stderr.Len() > 0 {
		s.opts.log.Warn(fmt.Sprintf("lsblk output to stderr: %v", stderr))
	}

	result, err := parseJSON(&stdout, &lsblk{})
	if err != nil {
		s.opts.log.Warn(fmt.Sprintf("failed to convert json to a valid lsblk struct: %v", err))
		return []diskInfo{}
	}

	lsblk, ok := result.(*lsblk)
	if !ok {
		s.opts.log.Warn("could not convert json to a valid lsblk struct: %v", result)
		return []diskInfo{}
	}

	blks := s.populateBlkInfo(lsblk.Lsblk)
	if len(blks) == 0 {
		s.opts.log.Warn("no Block information found")
	}

	return blks
}

// This regex matches the name, primary status, real resolution, and physical size from xrandr.
var screenHeaderRegex = regexp.MustCompile(`(?m)^(\S+)\s+connected\s+(?:(primary)\s+)?([0-9]+x[0-9]+).*?([0-9]+mm\s+x\s+[0-9]+mm).*$`)

// This regex matches the resolution and current refresh rate from xrandr.
var screenConfigRegex = regexp.MustCompile(`(?m)^\s*([0-9]+x[0-9]+)\s.*?([0-9]+\.[0-9]+)\+?\*\+?.*$`)

func (s Manager) collectScreens() []screenInfo {
	stdout, stderr, err := runCmd(context.Background(), s.opts.screenCmd[0], s.opts.screenCmd[1:]...)
	if err != nil {
		s.opts.log.Warn(fmt.Sprintf("failed to run xrandr: %v", err))
		return []screenInfo{}
	}
	if stderr.Len() > 0 {
		s.opts.log.Warn(fmt.Sprintf("xrandr output to stderr: %v", stderr))
	}

	data := stdout.String()
	screens := screenHeaderRegex.Split(data, -1)
	headers := screenHeaderRegex.FindAllStringSubmatch(data, -1)

	if len(headers) == 0 {
		s.opts.log.Warn("no Screen information found")
		return []screenInfo{}
	}

	info := make([]screenInfo, 0, len(headers))

	for i, header := range headers {
		v := screenConfigRegex.FindStringSubmatch(screens[i+1])

		info = append(info, screenInfo{
			Name:               header[1],
			PhysicalResolution: header[3],
			Size:               header[4],

			Resolution:  v[1],
			RefreshRate: v[2],
		})
	}

	return info
}

func (s Manager) collectHardware() (hwInfo hwInfo, err error) {
	hwInfo.Product = s.collectProduct()
	hwInfo.CPU = s.collectCPU()
	hwInfo.GPUs = s.collectGPUs()
	hwInfo.Mem = s.collectMemory()
	hwInfo.Blks = s.collectBlocks()
	hwInfo.Screens = s.collectScreens()

	return hwInfo, nil
}

func (s Manager) collectSoftware() (swInfo swInfo, err error) {
	return swInfo, nil
}
