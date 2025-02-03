package hardware

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ubuntu/ubuntu-insights/internal/cmdutils"
	"github.com/ubuntu/ubuntu-insights/internal/fileutils"
)

type platformOptions struct {
	productCmd []string
	cpuCmd     []string
	gpuCmd     []string
	memoryCmd  []string

	diskCmd      []string
	partitionCmd []string

	screenCmd []string
}

// defaultOptions returns options for when running under a normal environment.
func defaultPlatformOptions() platformOptions {
	return platformOptions{
		productCmd: []string{"powershell.exe", "-Command", "Get-CIMInstance", "Win32_ComputerSystem", "|", "Format-List", "-Property", "*"},
		cpuCmd:     []string{"powershell.exe", "-Command", "Get-CIMInstance", "Win32_Processor", "|", "Format-List", "-Property", "*"},
		gpuCmd:     []string{"powershell.exe", "-Command", "Get-CIMInstance", "Win32_VideoController", "|", "Format-List", "-Property", "*"},
		memoryCmd:  []string{"powershell.exe", "-Command", "Get-CIMInstance", "Win32_ComputerSystem", "|", "Format-List", "-Property", "TotalPhysicalMemory"},

		diskCmd:      []string{"powershell.exe", "-Command", "Get-CIMInstance", "Win32_DiskDrive", "|", "Format-List", "-Property", "*"},
		partitionCmd: []string{"powershell.exe", "-Command", "Get-CIMInstance", "Win32_DiskPartition", "|", "Format-List", "-Property", "*"},

		screenCmd: []string{"powershell.exe", "-Command", "Get-CIMInstance", "Win32_DesktopMonitor", "|", "Format-List", "-Property", "*"},
	}
}

var usedProductFields = map[string]struct{}{
	"Model":           {},
	"Manufacturer":    {},
	"SystemSKUNumber": {},
}

// collectProduct uses Win32_ComputerSystem to find information about the system.
func (s Collector) collectProduct() (product, error) {
	products, err := s.runWMI(s.platform.productCmd, usedProductFields)
	if err != nil {
		return product{}, err
	}
	if len(products) > 1 {
		s.log.Info("product information more than 1 products", "count", len(products))
	}

	return product{
		Family: products[0]["SystemSKUNumber"],
		Name:   products[0]["Model"],
		Vendor: products[0]["Manufacturer"],
	}, nil
}

var usedCPUFields = map[string]struct{}{
	"NumberOfLogicalProcessors": {},
	"NumberOfCores":             {},
	"Manufacturer":              {},
	"Name":                      {},
}

// collectCPU uses Win32_Processor to collect information about the CPUs.
func (s Collector) collectCPU() (info cpu, err error) {
	cpus, err := s.runWMI(s.platform.cpuCmd, usedCPUFields)
	if err != nil {
		return cpu{}, err
	}

	// we are assuming all CPUs are the same
	info.Sockets = uint64(len(cpus))
	info.Arch = s.arch

	info.Name = cpus[0]["Name"]
	info.Vendor = cpus[0]["Manufacturer"]

	total, err := strconv.ParseUint(cpus[0]["NumberOfLogicalProcessors"], 10, 64)
	if err != nil {
		s.log.Warn("CPU info contained invalid cpus", "value", cpus[0]["NumberOfLogicalProcessors"])
		total = 0
	}
	cores, err := strconv.ParseUint(cpus[0]["NumberOfCores"], 10, 64)
	if err != nil {
		s.log.Warn("CPU info contained invalid cores per socket", "value", cpus[0]["NumberOfCores"])
		cores = 0
	}

	return cpu{
		Name:    cpus[0]["Name"],
		Vendor:  cpus[0]["Manufacturer"],
		Arch:    s.arch,
		Cpus:    total,
		Sockets: uint64(len(cpus)),
		Cores:   cores,
		Threads: total / uint64(len(cpus)) / cores,
	}, nil
}

var usedGPUFields = map[string]struct{}{
	"Name":                    {},
	"InstalledDisplayDrivers": {},
	"AdapterCompatibility":    {},
}

// collectGPUs uses Win32_VideoController to collect information about the GPUs.
func (s Collector) collectGPUs() (info []gpu, err error) {
	gpus, err := s.runWMI(s.platform.gpuCmd, usedGPUFields)
	if err != nil {
		return []gpu{}, err
	}

	info = make([]gpu, 0, len(gpus))
	for _, g := range gpus {
		// InstalledDisplayDrivers is a comma separated list of paths to drivers
		v, _, _ := strings.Cut(g["InstalledDisplayDrivers"], ",")
		vs := strings.Split(v, `\`)

		info = append(info, gpu{
			Name:   g["Name"],
			Vendor: g["AdapterCompatibility"],
			Driver: vs[len(vs)-1],
		})
	}

	return info, nil
}

var usedMemoryFields = map[string]struct{}{
	"TotalPhysicalMemory": {},
}

// collectMemory uses Win32_ComputerSystem to collect information about RAM.
func (s Collector) collectMemory() (mem memory, err error) {
	oses, err := s.runWMI(s.platform.memoryCmd, usedMemoryFields)
	if err != nil {
		return memory{}, err
	}

	var size = 0
	for _, os := range oses {
		sm := os["TotalPhysicalMemory"]
		v, err := strconv.Atoi(sm)
		if err != nil {
			s.log.Warn("memory info contained non-integer memory", "value", sm)
			continue
		}
		if v < 0 {
			s.log.Warn("memory info contained negative memory", "value", sm)
			continue
		}
		size += v
	}

	m, _ := fileutils.ConvertUnitToStandard("b", size)
	return memory{
		Total: m,
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

// collectDisks uses Win32_DiskDrive and Win32_DiskPartition to collect information about disks.
func (s Collector) collectDisks() (blks []disk, err error) {
	getSize := func(b string) uint64 {
		v, err := strconv.ParseUint(b, 10, 64)
		if err != nil {
			s.log.Warn("disk partition contains invalid size")
			return 0
		}
		v, _ = fileutils.ConvertUnitToStandard("b", v)
		return v
	}

	disks, err := s.runWMI(s.platform.diskCmd, usedDiskFields)
	if err != nil {
		return nil, err
	}

	const maxPartitions = 128

	blks = make([]disk, 0, len(disks))
	for _, d := range disks {
		parts, err := strconv.Atoi(d["Partitions"])
		if err != nil {
			s.log.Warn("disk partitions was not an integer", "error", err)
			parts = 0
		}
		if parts < 0 {
			s.log.Warn("disk partitions was negative", "value", parts)
			parts = 0
		}
		if parts > maxPartitions {
			s.log.Warn("disk partitions too large", "value", parts)
			parts = maxPartitions
		}

		c := disk{
			Name:       d["Name"],
			Size:       getSize(d["Size"]),
			Partitions: make([]disk, parts),
		}
		for i := range c.Partitions {
			c.Partitions[i].Partitions = []disk{}
		}
		blks = append(blks, c)
	}

	parts, err := s.runWMI(s.platform.partitionCmd, usedPartitionFields)
	if err != nil {
		s.log.Warn("can't get partitions", "error", err)
		return blks, nil
	}

	for _, p := range parts {
		d, err := strconv.Atoi(p["DiskIndex"])
		if err != nil {
			s.log.Warn("partition disk index was not an integer", "error", err)
			continue
		}
		if d < 0 {
			s.log.Warn("partition disk index was negative", "value", d)
			continue
		}
		if d >= len(blks) {
			s.log.Warn("partition disk index was larger than disks", "value", d)
			continue
		}

		idx, err := strconv.Atoi(p["Index"])
		if err != nil {
			s.log.Warn("partition index was not an integer", "error", err, "disk", d)
			continue
		}
		if idx < 0 {
			s.log.Warn("partition index was negative", "value", idx, "disk", d)
			continue
		}
		if idx >= len(blks[d].Partitions) {
			s.log.Warn("partition index was larger than partitions", "value", idx, "disk", d)
			continue
		}

		blks[d].Partitions[idx] = disk{
			Name:       p["Name"],
			Size:       getSize(p["Size"]),
			Partitions: []disk{},
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
func (s Collector) collectScreens() (screens []screen, err error) {
	monitors, err := s.runWMI(s.platform.screenCmd, usedScreenFields)
	if err != nil {
		return nil, err
	}

	screens = make([]screen, 0, len(monitors))

	for _, s := range monitors {
		screens = append(screens, screen{
			Name:       s["Name"],
			Resolution: fmt.Sprintf("%sx%s", s["ScreenWidth"], s["ScreenHeight"]),
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
func (s Collector) runWMI(args []string, filter map[string]struct{}) (out []map[string]string, err error) {
	defer func() {
		if err == nil && len(out) == 0 {
			err = fmt.Errorf("%v output contained no sections", args)
		}
	}()

	if len(filter) == 0 {
		return nil, fmt.Errorf("empty filter will always produce nothing for cmdlet %v", args)
	}

	stdout, stderr, err := cmdutils.RunWithTimeout(context.Background(), 15*time.Second, args[0], args[1:]...)
	if err != nil {
		return nil, err
	}
	if stderr.Len() > 0 {
		s.log.Info(fmt.Sprintf("%v output to stderr", args), "stderr", stderr)
	}

	sections := wmiSplitRegex.Split(stdout.String(), -1)
	out = make([]map[string]string, 0, len(sections))

	for _, section := range sections {
		if section == "" {
			continue
		}

		entries := wmiEntryRegex.FindAllStringSubmatch(section, -1)
		if len(entries) == 0 {
			s.log.Info(fmt.Sprintf("%v output has malformed section", args), "section", section)
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
