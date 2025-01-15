package sysinfo

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type options struct {
	root       string
	cpuInfoCmd *exec.Cmd
	log        *slog.Logger
}

func defaultOptions() *options {
	return &options{
		root:       "/",
		cpuInfoCmd: exec.Command("lscpu", "-J"),
		log:        slog.Default(),
	}
}

type LscpuEntry struct {
	Field    string       `json:"field"`
	Data     string       `json:"data"`
	Children []LscpuEntry `json:"children,omitempty"`
}

type Lscpu struct {
	Lscpu []LscpuEntry `json:"lscpu"`
}

// readFile returns the data in <file>, or "" on error.
func (s Manager) readFile(file string) string {
	d, err := os.ReadFile(file)
	if err != nil {
		s.opts.log.Warn(err.Error())
		return ""
	}

	return string(d)
}

func (s Manager) collectProduct() map[string]string {
	return map[string]string{
		"Vendor": s.readFile(filepath.Join(s.opts.root, "sys/class/dmi/id/sys_vendor")),
		"Name":   s.readFile(filepath.Join(s.opts.root, "sys/class/dmi/id/product_name")),
		"Family": s.readFile(filepath.Join(s.opts.root, "sys/class/dmi/id/product_family")),
	}
}

var usedCpuFields = map[string]bool{
	"CPU(s):":             true,
	"Socket(s):":          true,
	"Core(s) per socket:": true,
	"Thread(s) per core:": true,
	"Architecture:":       true,
	"Vendor ID:":          true,
	"Model name:":         true,
}

func (s Manager) populateCpuInfo(entries []LscpuEntry, c *CpuInfo) CpuInfo {
	for _, entry := range entries {

		if usedCpuFields[entry.Field] {
			c.Cpu[entry.Field] = entry.Data
		}

		if len(entry.Children) > 0 {
			s.populateCpuInfo(entry.Children, c)
		}
	}

	return *c
}

func (s Manager) collectCPU() CpuInfo {
	o := CpuInfo{Cpu: map[string]string{}}

	r := runCmd(s.opts.cpuInfoCmd)
	result, err := parseJSON(r, &Lscpu{})
	if err != nil {
		s.opts.log.Warn(err.Error())
		return o
	}

	lscpu, ok := result.(*Lscpu)
	if !ok {
		s.opts.log.Warn("couldn't get CPU info, could not convert to a valid Lscpu struct: %v", result)
		return o
	}

	return s.populateCpuInfo(lscpu.Lscpu, &o)
}

func (s Manager) collectGPU(card string) (info GpuInfo, err error) {
	cardDir, err := filepath.EvalSymlinks(filepath.Join(s.opts.root, "sys/class/drm", card))
	if err != nil {
		return GpuInfo{}, err
	}

	devDir, err := filepath.EvalSymlinks(filepath.Join(cardDir, "device"))
	if err != nil {
		return GpuInfo{}, err
	}

	info.Gpu = make(map[string]string)

	info.Gpu["Vendor"] = s.readFile(filepath.Join(devDir, "vendor"))
	info.Gpu["Name"] = s.readFile(filepath.Join(devDir, "label"))

	driverLink, err := os.Readlink(filepath.Join(devDir, "driver"))
	if err == nil {
		info.Gpu["Driver"] = filepath.Base(driverLink)
	} else {
		s.opts.log.Warn(err.Error())
	}

	return info, nil
}

var gpuSymlinkRegex *regexp.Regexp = regexp.MustCompile("^card[0-9]+$")

func (s Manager) collectGPUs() []GpuInfo {
	gpus := make([]GpuInfo, 0, 2)

	ds, err := os.ReadDir(filepath.Join(s.opts.root, "sys/class/drm"))
	if err != nil {
		s.opts.log.Warn(err.Error())
		return gpus
	}

	for _, d := range ds {
		n := d.Name()

		if !gpuSymlinkRegex.MatchString(n) {
			continue
		}

		gpu, err := s.collectGPU(n)
		if err != nil {
			s.opts.log.Warn(err.Error())
			continue
		}

		gpus = append(gpus, gpu)
	}

	return gpus
}

// convertUnitToBytes takes a string bytes unit and converts value to bytes
func (s Manager) convertUnitToBytes(unit string, value int) int {
	switch strings.ToLower(unit) {
	case "":
		fallthrough
	case "b":
		return value
	case "kb":
		fallthrough
	case "kib":
		return value * 1024
	case "mb":
		fallthrough
	case "mib":
		return value * 1024 * 1024
	case "gb":
		fallthrough
	case "gib":
		return value * 1024 * 1024 * 1024
	case "tb":
		fallthrough
	case "tib":
		return value * 1024 * 1024 * 1024 * 1024
	default:
		s.opts.log.Warn(fmt.Sprintf("Unrecognized bytes unit: %s", unit))
		return value
	}
}

var usedMemFields = map[string]bool{
	"MemTotal": true,
}

// lines are in the form `key`:   `bytes` (`unit`)
var meminfoRegex *regexp.Regexp = regexp.MustCompile(`^([^\s:]+):\s*([0-9]+)(?:\s+([^\s]+))?\s*$`)

func (s Manager) collectMemory() MemInfo {
	o := MemInfo{Mem: map[string]int{}}

	f, err := os.ReadFile(filepath.Join(s.opts.root, "proc/meminfo"))
	if err != nil {
		s.opts.log.Warn(err.Error())
		return o
	}

	ls := strings.Split(string(f), "\n")
	for _, l := range ls {
		if l == "" {
			continue
		}

		m := meminfoRegex.FindStringSubmatch(l)
		if m == nil {
			s.opts.log.Warn(fmt.Sprintf("meminfo contains invalid line: %s", l))
			continue
		}

		if !usedMemFields[m[1]] {
			continue
		}

		b, err := strconv.Atoi(m[2])
		if err != nil {
			s.opts.log.Warn(err.Error())
			continue
		}

		o.Mem[m[1]] = s.convertUnitToBytes(m[3], b)
	}

	return o
}

func (s Manager) collectHardware() (hwInfo HwInfo, err error) {

	hwInfo.Product = s.collectProduct()
	hwInfo.Cpu = s.collectCPU()
	hwInfo.Gpus = s.collectGPUs()
	hwInfo.Mem = s.collectMemory()

	return hwInfo, nil
}

func (s Manager) collectSoftware() (swInfo SwInfo, err error) {

	return swInfo, nil
}
