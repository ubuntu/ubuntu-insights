package hardware

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ubuntu/ubuntu-insights/internal/cmdutils"
	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/platform"
	"github.com/ubuntu/ubuntu-insights/internal/fileutils"
)

// platformOptions are platform specific options.
type platformOptions struct {
	productCmd []string
	cpuCmd     []string
	gpuCmd     []string
	memoryCmd  []string

	diskCmd      []string
	partitionCmd []string

	screenResCmd  []string
	screenSizeCmd []string
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

		screenResCmd:  []string{"powershell.exe", "-Command", "Get-CIMInstance", "Win32_DesktopMonitor", "|", "Format-List", "-Property", "*"},
		screenSizeCmd: []string{"powershell.exe", "-Command", "Get-CIMInstance", "-Namespace", "root\\wmi", "WmiMonitorBasicDisplayParams", "|", "Format-List", "-Property", "*"},
	}
}

// collectProduct uses Win32_ComputerSystem to find information about the system.
func (s Collector) collectProduct(_ platform.Info) (product, error) {
	var usedProductFields = map[string]struct{}{
		"Model":           {},
		"Manufacturer":    {},
		"SystemSKUNumber": {},
	}

	products, err := cmdutils.RunListFmt(s.platform.productCmd, usedProductFields, s.log)
	if err != nil {
		return product{}, err
	}
	if len(products) > 1 {
		s.log.Warn("product information more than 1 products", "count", len(products))
	}

	return product{
		Family: products[0]["SystemSKUNumber"],
		Name:   products[0]["Model"],
		Vendor: products[0]["Manufacturer"],
	}, nil
}

// collectCPU uses Win32_Processor to collect information about the CPUs.
func (s Collector) collectCPU() (cpu, error) {
	var usedCPUFields = map[string]struct{}{
		"NumberOfLogicalProcessors": {},
		"NumberOfCores":             {},
		"Manufacturer":              {},
		"Name":                      {},
	}

	cpus, err := cmdutils.RunListFmt(s.platform.cpuCmd, usedCPUFields, s.log)
	if err != nil {
		return cpu{}, err
	}

	// we are assuming all CPUs are the same

	total, err := strconv.ParseUint(cpus[0]["NumberOfLogicalProcessors"], 10, 64)
	if err != nil {
		s.log.Warn("CPU info contained invalid cpus", "value", cpus[0]["NumberOfLogicalProcessors"])
		total = 0
	}
	cores, err := strconv.ParseUint(cpus[0]["NumberOfCores"], 10, 64)
	if err != nil {
		s.log.Warn("CPU info contained invalid cores per socket", "value", cpus[0]["NumberOfCores"])
		cores = 1
	}

	if cores == 0 {
		s.log.Warn("CPU info contained 0 cores")
		cores = 1
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

// collectGPUs uses Win32_VideoController to collect information about the GPUs.
func (s Collector) collectGPUs(_ platform.Info) (info []gpu, err error) {
	var usedGPUFields = map[string]struct{}{
		"Name":                    {},
		"InstalledDisplayDrivers": {},
		"AdapterCompatibility":    {},
	}

	gpus, err := cmdutils.RunListFmt(s.platform.gpuCmd, usedGPUFields, s.log)
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

// collectMemory uses Win32_ComputerSystem to collect information about RAM.
func (s Collector) collectMemory() (mem memory, err error) {
	var usedMemoryFields = map[string]struct{}{
		"TotalPhysicalMemory": {},
	}

	oses, err := cmdutils.RunListFmt(s.platform.memoryCmd, usedMemoryFields, s.log)
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

// collectDisks uses Win32_DiskDrive and Win32_DiskPartition to collect information about disks.
func (s Collector) collectDisks() (blks []disk, err error) {
	var usedDiskFields = map[string]struct{}{
		"Size":       {},
		"Index":      {},
		"Partitions": {},
	}

	var usedPartitionFields = map[string]struct{}{
		"DiskIndex": {},
		"Index":     {},
		"Size":      {},
	}

	getSize := func(b string) uint64 {
		v, err := strconv.ParseUint(b, 10, 64)
		if err != nil {
			s.log.Warn("disk partition contains invalid size")
			return 0
		}
		v, _ = fileutils.ConvertUnitToStandard("b", v)
		return v
	}

	disks, err := cmdutils.RunListFmt(s.platform.diskCmd, usedDiskFields, s.log)
	if err != nil {
		return nil, err
	}

	const maxPartitions = 128

	blks = make([]disk, len(disks))
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
			Size:       getSize(d["Size"]),
			Partitions: make([]disk, parts),
		}

		idx, err := strconv.ParseUint(d["Index"], 10, 64)
		if err != nil {
			s.log.Warn("disk index was not an unsigned integer", "error", err)
			continue
		}
		if idx >= uint64(len(blks)) {
			s.log.Warn("disk index was larger than disks", "value", idx)
			continue
		}
		if blks[idx].Size != 0 {
			s.log.Warn("duplicate disk index", "value", idx)
			continue
		}
		blks[idx] = c
	}

	parts, err := cmdutils.RunListFmt(s.platform.partitionCmd, usedPartitionFields, s.log)
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
			Size:       getSize(p["Size"]),
			Partitions: []disk{},
		}
	}

	return blks, nil
}

// collectScreens uses Win32_DesktopMonitor to collect information about screens.
func (s Collector) collectScreens(_ platform.Info) (screens []screen, err error) {
	var usedScreenResFields = map[string]struct{}{
		"ScreenWidth":  {},
		"ScreenHeight": {},
	}

	var usedScreenSizeFields = map[string]struct{}{
		"MaxHorizontalImageSize": {},
		"MaxVerticalImageSize":   {},
	}

	monitors, err := cmdutils.RunListFmt(s.platform.screenResCmd, usedScreenResFields, s.log)
	if err != nil {
		return nil, err
	}

	screens = make([]screen, 0, len(monitors))

	for _, s := range monitors {
		screens = append(screens, screen{
			Resolution: fmt.Sprintf("%sx%s", s["ScreenWidth"], s["ScreenHeight"]),
		})
	}

	monitors, err = cmdutils.RunListFmt(s.platform.screenSizeCmd, usedScreenSizeFields, s.log)
	if err != nil {
		s.log.Warn("physical screen size could not be determined", "error", err)
		return screens, nil
	}
	if len(monitors) != len(screens) {
		s.log.Warn("different number of screens than physical size returned", "value", len(monitors))
		return screens, nil
	}

	// assuming that the order of the monitors returned is the same.
	for i, s := range monitors {
		str := fmt.Sprintf("%s0mm x %s0mm", s["MaxHorizontalImageSize"], s["MaxVerticalImageSize"])
		screens[i].Size = str
	}

	return screens, nil
}
