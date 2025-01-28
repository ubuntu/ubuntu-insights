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

	return hwInfo, nil
}

func (s Manager) collectSoftware() (swInfo, error) {
	return swInfo{}, nil
}

var usedProductFields = map[string]struct{}{
	"Model":           {},
	"Manufacturer":    {},
	"SystemSKUNumber": {},
}

func (s Manager) collectProduct() (product map[string]string, err error) {
	products, err := s.runWMI(s.opts.productCmd, usedProductFields)
	if err != nil {
		return nil, err
	}
	if len(products) == 0 {
		return nil, fmt.Errorf("product information missing")
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

func (s Manager) collectCPU() (cpu map[string]string, err error) {
	cpus, err := s.runWMI(s.opts.cpuCmd, usedCPUFields)
	if err != nil {
		return nil, err
	}
	if len(cpus) == 0 {
		return nil, fmt.Errorf("cpu info has no cpus")
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

func (s Manager) collectGPUs() (gpus []map[string]string, err error) {
	defer func() {
		if err == nil && len(gpus) == 0 {
			err = fmt.Errorf("no GPU information found")
		}
	}()

	gpus, err = s.runWMI(s.opts.gpuCmd, usedGPUFields)
	if err != nil {
		return gpus, err
	}

	for _, g := range gpus {
		// InstalledDisplayDrivers is a comma seperated list of paths to drivers
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

func (s Manager) collectMemory() (mem map[string]int, err error) {
	defer func() {
		if err == nil && len(mem) == 0 {
			err = fmt.Errorf("no memory information found")
		}
	}()

	oses, err := s.runWMI(s.opts.memoryCmd, usedMemoryFields)
	if err != nil {
		return nil, err
	}
	if len(oses) == 0 {
		return nil, fmt.Errorf("memory info has no info")
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

func (s Manager) collectBlocks() (blks []diskInfo, err error) {
	disks, err := s.runWMI(s.opts.diskCmd, usedDiskFields)
	if err != nil {
		return nil, err
	}
	if len(disks) == 0 {
		return nil, fmt.Errorf("block info has no disks")
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

		blks = append(blks, diskInfo{
			Name:       d["Name"],
			Size:       d["Size"],
			Partitions: make([]diskInfo, parts),
		})
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

// wmiEntryRegex matches the key and value (if any) from gwmi output.
// For example: "Status   : OK " matches and has "Status", "OK".
// Or: "DitherType:" matches and has "DitherType", "".
// However: "   : OK" does not match.
var wmiEntryRegex = regexp.MustCompile(`(?m)^\s*(\S+)\s*:[^\S\n]*(.*?)\s*$`)

var wmiReplaceRegex = regexp.MustCompile(`\r?\n\s*`)

// wmiSplitRegex splits on two consecutive newlines, but \r needs special handling
var wmiSplitRegex = regexp.MustCompile(`\r?\n\r?\n`)

// runWMI runs the cmdlet "Get-WmiObject args..." and only includes fields in the filter.
func (s Manager) runWMI(args []string, filter map[string]struct{}) ([]map[string]string, error) {
	if len(filter) == 0 {
		return nil, fmt.Errorf("empty filter will always produce nothing for cmdlet Get-WmiObject %v", args)
	}

	stdout, stderr, err := runCmdWithTimeout(context.Background(), 5*time.Second, args[0], args[1:]...)
	if err != nil {
		return nil, err
	}
	if stderr.Len() > 0 {
		s.opts.log.Info(fmt.Sprintf("Get-WmiObject %v output to stderr", args[1:]), "stderr", stderr)
	}

	sections := wmiSplitRegex.Split(stdout.String(), -1)
	out := make([]map[string]string, 0, len(sections))

	for _, section := range sections {
		if section == "" {
			continue
		}

		entries := wmiEntryRegex.FindAllStringSubmatch(section, -1)
		if len(entries) == 0 {
			s.opts.log.Info(fmt.Sprintf("Get-WmiObject %v output has malformed section", args[1:]), "section", section)
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
