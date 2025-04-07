package hardware

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strconv"
	"strings"
	"time"

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

	screenResCmd     []string
	screenPhysResCmd []string
	displaySizeCmd   []string
}

// defaultOptions returns options for when running under a normal environment.
func defaultPlatformOptions() platformOptions {
	return platformOptions{
		productCmd: []string{"powershell.exe", "-Command", "Get-CIMInstance", "Win32_ComputerSystem", "|", "Format-List", "-Property", "*"},
		cpuCmd:     []string{"powershell.exe", "-Command", "Get-CIMInstance", "Win32_Processor", "|", "Format-List", "-Property", "*"},
		gpuCmd:     []string{"powershell.exe", "-Command", "Get-CIMInstance", "Win32_VideoController", "|", "Format-List", "-Property", "*"},
		memoryCmd:  []string{"powershell.exe", "-Command", "Get-CIMInstance", "Win32_ComputerSystem", "|", "Format-List", "-Property", "TotalPhysicalMemory"},

		diskCmd:      []string{"powershell.exe", "-Command", "Get-WmiObject", "Win32_DiskDrive", "|", "Select-Object", "Model, MediaType, Index, Size, Partitions", "|", "ConvertTo-Json", "-Depth", "3"},
		partitionCmd: []string{"powershell.exe", "-Command", "Get-WmiObject", "Win32_DiskPartition", "|", "Select-Object", "DiskIndex, Size, Type", "|", "ConvertTo-Json", "-Depth", "3"},

		screenResCmd: []string{"powershell.exe", "-Command",
			"Add-Type", "-AssemblyName", "System.Windows.Forms", ";", "[System.Windows.Forms.Screen]::AllScreens", "|",
			"Select-Object", "Bounds", "|",
			"ConvertTo-Json", "-Depth", "3"},
		screenPhysResCmd: []string{"powershell.exe", "-Command",
			"Get-CIMInstance", "Win32_DesktopMonitor", "|",
			"Select-Object", "ScreenWidth, ScreenHeight", "|",
			"ConvertTo-Json", "-Depth", "3"},
		displaySizeCmd: []string{"powershell.exe", "-Command",
			"Get-CIMInstance", "-Namespace", "root\\wmi", "WmiMonitorBasicDisplayParams", "|",
			"Select-Object", "MaxHorizontalImageSize, MaxVerticalImageSize", "|",
			"ConvertTo-Json", "-Depth", "3"},
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
	type diskOut struct {
		Model      string
		MediaType  string
		Index      uint64
		Size       uint64
		Partitions uint64
	}

	type partOut struct {
		DiskIndex uint64
		Size      uint64
		Type      string
	}

	disksOut, err := runJSONCommand[diskOut]("disk", s.log, s.platform.diskCmd[0], s.platform.diskCmd[1:]...)
	if err != nil {
		return nil, err
	}

	diskIndicesSeen := make(map[uint64]bool)
	diskMap := make(map[uint64]int) // Map between blk index and disk index
	for _, d := range disksOut {
		if value, ok := diskIndicesSeen[d.Index]; ok && value {
			s.log.Warn("Skipping duplicate disk index", "index", d.Index)
			continue
		}
		diskIndicesSeen[d.Index] = true

		if d.MediaType != "Fixed hard disk media" {
			s.log.Info("Skipping non-fixed disk", "mediaType", d.MediaType)
			continue
		}

		if d.Model == "Microsoft Virtual Disk" {
			s.log.Info("Skipping virtual disk", "index", d.Index)
			continue
		}

		if d.Partitions > 128 {
			s.log.Warn("Skipping disk with too many partitions", "partitions", d.Partitions)
			continue
		}

		d.Size, err = fileutils.ConvertUnitToStandard("b", d.Size)
		if err != nil {
			s.log.Warn("Failed to convert disk size to standard unit", "error", err)
			continue
		}

		blks = append(blks, disk{
			Size:     d.Size,
			Type:     "disk",
			Children: make([]disk, 0, d.Partitions),
		})
		diskMap[d.Index] = len(blks) - 1
	}

	// Paritions
	partsOut, err := runJSONCommand[partOut]("partition", s.log, s.platform.partitionCmd[0], s.platform.partitionCmd[1:]...)
	if err != nil {
		return nil, err
	}

	for _, p := range partsOut {
		if valid, ok := diskIndicesSeen[p.DiskIndex]; !ok || !valid {
			s.log.Warn("Skipping partition with unknown disk index", "diskIndex", p.DiskIndex)
			continue
		}

		i, ok := diskMap[p.DiskIndex]
		if !ok {
			s.log.Info("Skipping partition with discarded disk index", "diskIndex", p.DiskIndex)
			continue
		}

		p.Size, err = fileutils.ConvertUnitToStandard("b", p.Size)
		if err != nil {
			s.log.Warn("Failed to convert partition size to standard unit", "error", err)
			continue
		}

		blks[i].Children = append(blks[i].Children, disk{
			Size: p.Size,
			Type: p.Type,
		})
	}

	return blks, nil
}

// collectScreens uses Win32_DesktopMonitor to collect information about screens.
func (s Collector) collectScreens(_ platform.Info) (screens []screen, err error) {
	defer func() {
		if err == nil && len(screens) == 0 {
			err = errors.New("no screens found")
		}
	}()

	type screenPhysResFields struct {
		ScreenWidth  uint
		ScreenHeight uint
	}

	type screenResFields struct {
		Bounds struct {
			Width  uint
			Height uint
		}
	}

	type displaySizeFields struct {
		MaxHorizontalImageSize uint
		MaxVerticalImageSize   uint
	}

	// Screen resolutions
	screenRes, err := runJSONCommand[screenResFields]("screen resolution", s.log, s.platform.screenResCmd[0], s.platform.screenResCmd[1:]...)
	if err != nil {
		screenRes = nil
	}

	screens = make([]screen, 0, len(screenRes))
	for _, sc := range screenRes {
		screens = append(screens, screen{Resolution: fmt.Sprintf("%dx%d", sc.Bounds.Width, sc.Bounds.Height)})
	}

	displaySizes, err := runJSONCommand[displaySizeFields]("physical display size", s.log, s.platform.displaySizeCmd[0], s.platform.displaySizeCmd[1:]...)
	if err != nil {
		displaySizes = nil
	}

	if len(displaySizes) != len(screens) && len(displaySizes) != 0 {
		if len(screens) != 0 {
			// Ignore display physical sizes and move on
			s.log.Warn("Different number of screens than display physical size returned", "screens", len(screens), "physicalSizes", len(displaySizes))
			displaySizes = nil
		} else {
			// Make do with what we have.
			s.log.Info("No screen resolution available, using physical size only")
			screens = make([]screen, len(displaySizes))
		}
	}

	// assuming that the order of the monitors returned is the same.
	for i, d := range displaySizes {
		screens[i].Size = fmt.Sprintf("%d0mm x %d0mm", d.MaxHorizontalImageSize, d.MaxVerticalImageSize)
	}

	// Physical reolution - Should be last and lowest priority due to its often inconsistent behavior.
	screenPhysRes, err := runJSONCommand[screenPhysResFields]("screen physical resolution", s.log, s.platform.screenPhysResCmd[0], s.platform.screenPhysResCmd[1:]...)
	if err != nil {
		screenPhysRes = nil
	}

	if len(screenPhysRes) != len(screens) && len(screenPhysRes) != 0 {
		if len(screens) != 0 {
			s.log.Warn("Different number of screens than screen physical resolution returned", "screens", len(screens), "screenPhysRes", len(screenPhysRes))
			return screens, nil
		}
		// Make do with what we have.
		s.log.Info("No screen resolution available, using physical size only")
		screens = make([]screen, len(screenPhysRes))
	}

	for i, m := range screenPhysRes {
		screens[i].PhysicalResolution = fmt.Sprintf("%dx%d", m.ScreenWidth, m.ScreenHeight)
	}

	return screens, nil
}

// runJSONCommand runs a command and returns the output as a list of objects.
func runJSONCommand[T any](cmdName string, log *slog.Logger, cmd string, cmdArgs ...string) ([]T, error) {
	stdout, stderr, err := cmdutils.RunWithTimeout(context.Background(), 15*time.Second, cmd, cmdArgs...)
	if err != nil {
		log.Warn(fmt.Sprintf("Failed to run %s command", cmdName), "error", err, "stderr", stderr)
		return nil, err
	}

	if stderr.String() != "" {
		log.Info(fmt.Sprintf("Command %s returned stderr", cmdName), "stderr", stderr)
	}

	r, err := fileutils.UnmarshalJSON[T](stdout.Bytes())
	// Filter out empty results
	results := make([]T, 0, len(r))
	var empty T
	for _, res := range r {
		if reflect.DeepEqual(res, empty) {
			log.Info(fmt.Sprintf("Command %s returned an empty result", cmdName))
			continue
		}
		results = append(results, res)
	}

	if len(results) == 0 {
		log.Warn(fmt.Sprintf("Command %s returned no results", cmdName))
	}

	return results, err
}
