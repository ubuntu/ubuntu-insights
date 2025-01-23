package sysinfo

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
)

type options struct {
	gpuCmd []string
	log    *slog.Logger
}

func defaultOptions() *options {
	return &options{
		gpuCmd: []string{"Get-WmiObject", "Win32_VideoController"},
		log:    slog.Default(),
	}
}

func (s Manager) collectHardware() (hwInfo hwInfo, err error) {

	hwInfo.GPUs, err = s.collectGPUs()
	if err != nil {
		s.opts.log.Warn(fmt.Sprintf("%v", err))
		hwInfo.GPUs = []map[string]string{}
	}

	return hwInfo, nil
}

func (s Manager) collectSoftware() (swInfo, error) {
	return swInfo{}, nil
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

	stdout, stderr, err := runCmd(context.Background(), args[0], args[1:]...)
	if err != nil {
		return nil, err
	}
	if stderr.Len() > 0 {
		s.opts.log.Info(fmt.Sprintf("Get-WmiObject %v output to stderr: %v", args, stderr))
	}

	sections := strings.Split(stdout.String(), "\n\n")
	out := make([]map[string]string, 0, len(sections))

	for _, section := range sections {
		if section == "" {
			continue
		}

		entries := wmiEntryRegex.FindAllStringSubmatch(section, -1)
		if len(entries) == 0 {
			s.opts.log.Info(fmt.Sprintf("Get-WmiObject %v output has malformed section: %s", args, section))
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
