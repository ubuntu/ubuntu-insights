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
	log        *slog.Logger
}

func defaultOptions() *options {
	return &options{
		productCmd: []string{"powershell.exe", "-Command", "Get-CIMInstance", "Win32_ComputerSystem", "|", "Format-List", "-Property", "*"},
		cpuCmd:     []string{"powershell.exe", "-Command", "Get-CIMInstance", "Win32_Processor", "|", "Format-List", "-Property", "*"},
		gpuCmd:     []string{"powershell.exe", "-Command", "Get-CIMInstance", "Win32_VideoController", "|", "Format-List", "-Property", "*"},
		log:        slog.Default(),
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

// wmiEntryRegex matches the key and value (if any) from gwmi output.
// For example: "Status   : OK " matches and has "Status", "OK".
// Or: "DitherType:" matches and has "DitherType", "".
// However: "   : OK" does not match.
var wmiEntryRegex = regexp.MustCompile(`(?m)^\s*(\S+)\s*:[^\S\n]*(.*?)\s*$`)

var wmiReplaceRegex = regexp.MustCompile(`\n\s*`)

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

	sections := strings.Split(stdout.String(), "\n\n")
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
