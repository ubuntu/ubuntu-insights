package sysinfo

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type options struct {
	productCmd []string
	cpuCmd     []string
	gpuCmd     []string
	memoryCmd  []string

	diskCmd      []string
	partitionCmd []string

	screenCmd []string

	log *slog.Logger
}

// defaultOptions returns options for when running under a normal environment.
func defaultOptions() *options {
	return &options{
		productCmd: []string{"powershell.exe", "-Command", "Get-CIMInstance", "Win32_ComputerSystem", "|", "Format-List", "-Property", "*"},
		cpuCmd:     []string{"powershell.exe", "-Command", "Get-CIMInstance", "Win32_Processor", "|", "Format-List", "-Property", "*"},
		gpuCmd:     []string{"powershell.exe", "-Command", "Get-CIMInstance", "Win32_VideoController", "|", "Format-List", "-Property", "*"},
		memoryCmd:  []string{"powershell.exe", "-Command", "Get-CIMInstance", "Win32_ComputerSystem", "|", "Format-List", "-Property", "TotalPhysicalMemory"},

		diskCmd:      []string{"powershell.exe", "-Command", "Get-CIMInstance", "Win32_DiskDrive", "|", "Format-List", "-Property", "*"},
		partitionCmd: []string{"powershell.exe", "-Command", "Get-CIMInstance", "Win32_DiskPartition", "|", "Format-List", "-Property", "*"},

		screenCmd: []string{"powershell.exe", "-Command", "Get-CIMInstance", "Win32_DesktopMonitor", "|", "Format-List", "-Property", "*"},

		log: slog.Default(),
	}
}

// collectHardware aggregates the data from all the other hardware collect functions.
func (s Manager) collectHardware() (hwInfo hwInfo, err error) {
	hwInfo.Product, err = s.collectProduct()
	if err != nil {
		s.opts.log.Warn("failed to collect product info", "error", err)
		hwInfo.Product = map[string]string{}
	}

	hwInfo.CPU, err = s.collectCPU()
	if err != nil {
		s.opts.log.Warn("failed to collect cpu info", "error", err)
		hwInfo.CPU = map[string]string{}
	}

	hwInfo.GPUs, err = s.collectGPUs()
	if err != nil {
		s.opts.log.Warn("failed to collect GPU info", "error", err)
		hwInfo.GPUs = []map[string]string{}
	}

	hwInfo.Mem, err = s.collectMemory()
	if err != nil {
		s.opts.log.Warn("failed to collect Memory info", "error", err)
		hwInfo.Mem = map[string]int{}
	}

	hwInfo.Blks, err = s.collectBlocks()
	if err != nil {
		s.opts.log.Warn("failed to collect Block info", "error", err)
		hwInfo.Blks = []diskInfo{}
	}

	hwInfo.Screens, err = s.collectScreens()
	if err != nil {
		s.opts.log.Warn("failed to collect Screen info", "error", err)
		hwInfo.Screens = []screenInfo{}
	}

	return hwInfo, nil
}

// collectSoftware aggregates the data from all the other software collect functions.
func (s Manager) collectSoftware() (swInfo, error) {
	return swInfo{}, nil
}

var usedProductFields = map[string]struct{}{
	"Model":           {},
	"Manufacturer":    {},
	"SystemSKUNumber": {},
}

// collectProduct uses Win32_ComputerSystem to find information about the system.
func (s Manager) collectProduct() (product map[string]string, err error) {
	products, err := s.runWMI(s.opts.productCmd, usedProductFields)
	if err != nil {
		return nil, err
	}
	if len(products) > 1 {
		s.opts.log.Info("product information more than 1 products", "count", len(products))
	}

	product = products[0]

	product["Family"] = product["SystemSKUNumber"]
	delete(product, "SystemSKUNumber")

	product["Vendor"] = product["Manufacturer"]
	delete(product, "Manufacturer")

	return product, nil
}

var usedCPUFields = map[string]struct{}{
	"NumberOfLogicalProcessors": {},
	"NumberOfCores":             {},
	"Manufacturer":              {},
	"Name":                      {},
}

// collectCPU uses Win32_Processor to collect information about the CPUs.
func (s Manager) collectCPU() (cpu map[string]string, err error) {
	cpus, err := s.runWMI(s.opts.cpuCmd, usedCPUFields)
	if err != nil {
		return nil, err
	}

	// we are assuming all CPUs are the same
	cpus[0]["Sockets"] = strconv.Itoa(len(cpus))

	return cpus[0], nil
}

var usedGPUFields = map[string]struct{}{
	"Name":                    {},
	"InstalledDisplayDrivers": {},
	"AdapterCompatibility":    {},
}

// collectGPUs uses Win32_VideoController to collect information about the GPUs.
func (s Manager) collectGPUs() (gpus []map[string]string, err error) {
	gpus, err = s.runWMI(s.opts.gpuCmd, usedGPUFields)
	if err != nil {
		return gpus, err
	}

	for _, g := range gpus {
		// InstalledDisplayDrivers is a comma separated list of paths to drivers
		v, _, _ := strings.Cut(g["InstalledDisplayDrivers"], ",")
		vs := strings.Split(v, `\`)

		g["Driver"] = vs[len(vs)-1]
		delete(g, "InstalledDisplayDrivers")

		g["Vendor"] = g["AdapterCompatibility"]
		delete(g, "AdapterCompatibility")
	}

	return gpus, nil
}

var usedMemoryFields = map[string]struct{}{
	"TotalPhysicalMemory": {},
}

// collectMemory uses Win32_ComputerSystem to collect information about RAM.
func (s Manager) collectMemory() (mem map[string]int, err error) {
	oses, err := s.runWMI(s.opts.memoryCmd, usedMemoryFields)
	if err != nil {
		return nil, err
	}

	var size = 0
	for _, os := range oses {
		sm := os["TotalPhysicalMemory"]
		v, err := strconv.Atoi(sm)
		if err != nil {
			s.opts.log.Warn("memory info contained non-integer memory", "value", sm)
			continue
		}
		size += v
	}

	return map[string]int{
		"MemTotal": size,
	}, nil
}

var usedDiskFields = map[string]struct{}{
	"Name":       {},
	"Size":       {},
	"Partitions": {},
}

var usedPartitionFields = map[string]struct{}{
	"DiskIndex": {},
	"Index":     {},
	"Name":      {},
	"Size":      {},
}

// collectBlocks uses Win32_DiskDrive and Win32_DiskPartition to collect information about disks.
func (s Manager) collectBlocks() (blks []diskInfo, err error) {
	disks, err := s.runWMI(s.opts.diskCmd, usedDiskFields)
	if err != nil {
		return nil, err
	}

	blks = make([]diskInfo, 0, len(disks))
	for _, d := range disks {
		parts, err := strconv.Atoi(d["Partitions"])
		if err != nil {
			s.opts.log.Warn("disk partitions was not an integer", "error", err)
			parts = 0
		}
		if parts < 0 {
			s.opts.log.Warn("disk partitions was negative", "value", parts)
			parts = 0
		}

		c := diskInfo{
			Name:       d["Name"],
			Size:       d["Size"],
			Partitions: make([]diskInfo, parts),
		}
		for i := range c.Partitions {
			c.Partitions[i].Partitions = []diskInfo{}
		}
		blks = append(blks, c)
	}

	parts, err := s.runWMI(s.opts.partitionCmd, usedPartitionFields)
	if err != nil {
		s.opts.log.Warn("can't get partitions", "error", err)
		return blks, nil
	}

	for _, p := range parts {
		disk, err := strconv.Atoi(p["DiskIndex"])
		if err != nil {
			s.opts.log.Warn("partition disk index was not an integer", "error", err)
			continue
		}
		if disk < 0 {
			s.opts.log.Warn("partition disk index was negative", "value", disk)
			continue
		}
		if disk >= len(blks) {
			s.opts.log.Warn("partition disk index was larger than disks", "value", disk)
			continue
		}

		idx, err := strconv.Atoi(p["Index"])
		if err != nil {
			s.opts.log.Warn("partition index was not an integer", "error", err, "disk", disk)
			continue
		}
		if idx < 0 {
			s.opts.log.Warn("partition index was negative", "value", idx, "disk", disk)
			continue
		}
		if idx >= len(blks[disk].Partitions) {
			s.opts.log.Warn("partition index was larger than partitions", "value", idx, "disk", disk)
			continue
		}

		blks[disk].Partitions[idx] = diskInfo{
			Name:       p["Name"],
			Size:       p["Size"],
			Partitions: []diskInfo{},
		}
	}

	return blks, nil
}

var usedScreenFields = map[string]struct{}{
	"Name":         {},
	"ScreenWidth":  {},
	"ScreenHeight": {},
}

// collectScreens uses Win32_DesktopMonitor to collect information about screens.
func (s Manager) collectScreens() (screens []screenInfo, err error) {
	monitors, err := s.runWMI(s.opts.screenCmd, usedScreenFields)
	if err != nil {
		return nil, err
	}

	screens = make([]screenInfo, 0, len(monitors))

	for _, screen := range monitors {
		screens = append(screens, screenInfo{
			Name:       screen["Name"],
			Resolution: fmt.Sprintf("%sx%s", screen["ScreenWidth"], screen["ScreenHeight"]),
		})
	}

	return screens, nil
}

// wmiEntryRegex matches the key and value (if any) from gwmi output.
// For example: "Status   : OK " matches and has "Status", "OK".
// Or: "DitherType:" matches and has "DitherType", "".
// However: "   : OK" does not match.
var wmiEntryRegex = regexp.MustCompile(`(?m)^\s*(\S+)\s*:[^\S\n]*(.*?)\s*$`)

var wmiReplaceRegex = regexp.MustCompile(`\r?\n\s*`)

// wmiSplitRegex splits on two consecutive newlines, but \r needs special handling.
var wmiSplitRegex = regexp.MustCompile(`\r?\n\r?\n`)

// runWMI runs the cmdlet specified by args and only includes fields in the filter.
func (s Manager) runWMI(args []string, filter map[string]struct{}) (out []map[string]string, err error) {
	defer func() {
		if err == nil && len(out) == 0 {
			err = fmt.Errorf("%v output contained no sections", args)
		}
	}()

	if len(filter) == 0 {
		return nil, fmt.Errorf("empty filter will always produce nothing for cmdlet %v", args)
	}

	stdout, stderr, err := runCmdWithTimeout(context.Background(), 15*time.Second, args[0], args[1:]...)
	if err != nil {
		return nil, err
	}
	if stderr.Len() > 0 {
		s.opts.log.Info(fmt.Sprintf("%v output to stderr", args), "stderr", stderr)
	}

	sections := wmiSplitRegex.Split(stdout.String(), -1)
	out = make([]map[string]string, 0, len(sections))

	for _, section := range sections {
		if section == "" {
			continue
		}

		entries := wmiEntryRegex.FindAllStringSubmatch(section, -1)
		if len(entries) == 0 {
			s.opts.log.Info(fmt.Sprintf("%v output has malformed section", args), "section", section)
			continue
		}

		v := make(map[string]string, len(filter))
		for _, e := range entries {
			if _, ok := filter[e[1]]; !ok {
				continue
			}

			// Get-WmiObject injects newlines and whitespace into values for formatting
			v[e[1]] = wmiReplaceRegex.ReplaceAllString(e[2], "")
		}

		out = append(out, v)
	}

	return out, nil
}
