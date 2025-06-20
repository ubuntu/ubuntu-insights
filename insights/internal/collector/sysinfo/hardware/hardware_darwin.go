package hardware

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ubuntu/ubuntu-insights/insights/internal/cmdutils"
	"github.com/ubuntu/ubuntu-insights/insights/internal/collector/sysinfo/platform"
	"github.com/ubuntu/ubuntu-insights/shared/fileutils"
)

type platformOptions struct {
	cpuCmd    []string
	gpuCmd    []string
	memCmd    []string
	diskCmd   []string
	screenCmd []string
}

func defaultPlatformOptions() platformOptions {
	return platformOptions{
		cpuCmd:    []string{"sysctl", "-a", "hw.packages", "machdep.cpu"},
		gpuCmd:    []string{"system_profiler", "-json", "SPDisplaysDataType"},
		memCmd:    []string{"sysctl", "-a", "hw.memsize"},
		diskCmd:   []string{"diskutil", "list", "-plist"},
		screenCmd: []string{"system_profiler", "-json", "SPDisplaysDataType"},
	}
}

type gpuAndScreens struct {
	Gpus []struct {
		// Type is expected to be "spdisplays_gpu"
		Type    string `json:"sppci_device_type"`
		Name    string `json:"sppci_model"`
		Vendor  string `json:"spdisplays_vendor"`
		Version string `json:"spdisplays_revision-id"`

		Displays []struct {
			Name               string `json:"_name"`
			Online             string `json:"spdisplays_online"`
			ResolutionRefresh  string `json:"_spdisplays_resolution"`
			PhysicalResolution string `json:"spdisplays_pixelresolution"`
		} `json:"spdisplays_ndrvs"`
	} `json:"SPDisplaysDataType"`
}

func (s Collector) collectProduct(platform.Info) (product, error) {
	return product{
		Family: "Apple Motherboard",
		Name:   "",
		Vendor: "Apple",
	}, nil
}

func (s Collector) collectCPU() (cpu, error) {
	var usedCPUFields = map[string]struct{}{
		"machdep.cpu.brand_string":        {},
		"machdep.cpu.vendor":              {},
		"hw.packages":                     {},
		"machdep.cpu.cores_per_package":   {},
		"machdep.cpu.logical_per_package": {},
	}

	cpus, err := cmdutils.RunListFmt(s.platform.cpuCmd, usedCPUFields, s.log)
	if err != nil {
		return cpu{}, err
	}
	if len(cpus) > 1 {
		s.log.Warn("cpu information improperly formatted, more than 1 section", "count", len(cpus))
	}

	c := cpus[0]

	sockets, err := strconv.ParseUint(c["hw.packages"], 10, 64)
	if err != nil {
		s.log.Warn("CPU info contained invalid sockets", "value", c["hw.packages"])
		sockets = 1
	}
	cores, err := strconv.ParseUint(c["machdep.cpu.cores_per_package"], 10, 64)
	if err != nil || cores == 0 {
		s.log.Warn("CPU info contained invalid cores per socket", "value", c["machdep.cpu.cores_per_package"])
		cores = 0
	}
	threads, err := strconv.ParseUint(c["machdep.cpu.logical_per_package"], 10, 64)
	if err != nil {
		s.log.Warn("CPU info contained invalid threads per socket", "value", c["machdep.cpu.logical_per_package"])
		threads = 0
	}

	threadsPerCore := uint64(0)
	if cores != 0 {
		threadsPerCore = threads / cores
	}

	return cpu{
		Name:    c["machdep.cpu.brand_string"],
		Vendor:  c["machdep.cpu.vendor"],
		Arch:    s.arch,
		Cpus:    threads * sockets,
		Sockets: sockets,
		Cores:   cores * sockets,
		Threads: threadsPerCore,
	}, nil
}

func (s Collector) collectGPUs(platform.Info) ([]gpu, error) {
	stdout, stderr, err := cmdutils.RunWithTimeout(context.Background(), 15*time.Second, s.platform.gpuCmd[0], s.platform.gpuCmd[1:]...)
	if err != nil {
		return []gpu{}, fmt.Errorf("failed to run system_profiler: %v", err)
	}
	if stderr.Len() > 0 {
		s.log.Info("system_profiler output to stderr", "stderr", stderr)
	}

	var data gpuAndScreens
	err = json.Unmarshal(stdout.Bytes(), &data)
	if err != nil {
		return []gpu{}, err
	}

	out := make([]gpu, 0, len(data.Gpus))
	for _, g := range data.Gpus {
		// skip devices that aren't gpus.
		if g.Type != "spdisplays_gpu" {
			continue
		}

		out = append(out, gpu{
			Name:   g.Name,
			Vendor: g.Vendor,
			Driver: g.Version,
		})
	}

	return out, nil
}

func (s Collector) collectMemory() (memory, error) {
	memorys, err := cmdutils.RunListFmt(s.platform.memCmd, nil, s.log)
	if err != nil {
		return memory{}, err
	}
	if len(memorys) > 1 {
		s.log.Warn("memory information improperly formatted, more than 1 section", "count", len(memorys))
	}
	m := memorys[0]

	ms := m["hw.memsize"]
	v, err := strconv.Atoi(ms)
	if err != nil {
		return memory{}, err
	}
	if v < 0 {
		return memory{}, errors.New("memory information contained negative memory")
	}
	v, _ = fileutils.ConvertUnitToStandard("b", v)

	return memory{
		Total: v,
	}, nil
}

func (s Collector) collectDisks() (out []disk, err error) {
	defer func() {
		if err != nil && len(out) == 0 {
			err = errors.New("no disk information found")
		}
	}()
	out = []disk{}

	stdout, stderr, err := cmdutils.RunWithTimeout(context.Background(), 15*time.Second, s.platform.diskCmd[0], s.platform.diskCmd[1:]...)
	if err != nil {
		return out, fmt.Errorf("failed to run diskutil: %v", err)
	}
	if stderr.Len() > 0 {
		s.log.Info("diskutil output to stderr", "stderr", stderr)
	}

	// Using a multipass(!) approach since XML is difficult to work with
	jsonified, err := parseDiskXML(stdout)
	if err != nil {
		return out, err
	}

	disksAndPartsI := jsonified["AllDisksAndPartitions"]

	disksAndParts, ok := disksAndPartsI.([]any)
	if !ok {
		return out, errors.New("diskutil did not contain AllDisksAndPartitions")
	}

	for _, diskI := range disksAndParts {
		disk, ok := diskI.(map[string]any)
		if !ok {
			s.log.Warn("AllDisksAndPartitions contained non-dict data")
			continue
		}

		d, err := parseDiskDict(disk, false, s.log)
		if err != nil {
			s.log.Info("AllDisksAndPartitions contained a fake disk", "error", err)
			continue
		}
		out = append(out, d)
	}

	return out, nil
}

func parseDiskDict(data map[string]any, partition bool, log *slog.Logger) (out disk, err error) {
	out.Children = []disk{}
	if _, ok := data["APFSPhysicalStores"]; ok {
		return out, errors.New("disk is a virtual APFS disk")
	}

	// use lambda to reduce nesting.
	out.Size = func() uint64 {
		sizeI, ok := data["Size"]
		if !ok {
			log.Warn("disk missing Size")
			return 0
		}
		size, ok := sizeI.(int)
		if !ok {
			log.Warn("disk Size was not an integer")
			return 0
		}

		if size < 0 {
			log.Warn("disk Size is negative", "value", size)
			return 0
		}
		v, err := fileutils.ConvertUnitToStandard("b", uint64(size))
		if err != nil {
			log.Warn("could not convert bytes to standard", "error", err)
			return 0
		}
		return v
	}()

	// partition is true if we are currently parsing a partition.
	if partition {
		out.Type = "part"
		return out, err
	}

	// otherwise, we want to get the current disk's partitions.
	out.Type = "disk"
	partsI, ok := data["Partitions"]
	if !ok {
		log.Warn("disk missing partitions")
		return out, err
	}
	parts, ok := partsI.([]any)
	if !ok {
		log.Warn("disk partitions aren't an array")
		return out, err
	}

	for _, partI := range parts {
		part, ok := partI.(map[string]any)
		if !ok {
			log.Warn("partitions contained non-dict data")
			continue
		}

		d, err := parseDiskDict(part, true, log)
		if err != nil {
			log.Warn("partitions contained a fake partition", "error", err)
			continue
		}
		out.Children = append(out.Children, d)
	}

	return out, err
}

func isXMLProcInst(got xml.Token, target string) error {
	if v, ok := got.(xml.ProcInst); ok {
		if target != "" && v.Target != target {
			return fmt.Errorf("ProcInst Token didn't have target %s, had %s", target, v.Target)
		}
		return nil
	}
	return fmt.Errorf("token was not a ProcInst Token with target %s", target)
}

func isXMLDirective(got xml.Token) error {
	if _, ok := got.(xml.Directive); !ok {
		return errors.New("token was not a Directive Token")
	}
	return nil
}

func getXMLStartElement(tok xml.Token) (xml.StartElement, error) {
	if v, ok := tok.(xml.StartElement); ok {
		return v, nil
	}
	return xml.StartElement{}, errors.New("token was not a StartElement Token")
}

// tokenSkipWhitespace returns the next Token from `dec` that is not whitespace.
func tokenSkipWhitespace(dec *xml.Decoder) (xml.Token, error) {
	for {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		if tok == nil {
			return tok, nil
		}

		if str, ok := tok.(xml.CharData); ok {
			if strings.TrimSpace(string(str)) != "" {
				return tok, nil
			}
			continue
		}
		return tok, nil
	}
}

// parseDiskXML parses Apple's PList XML format into a more sane JSON-esque format.
func parseDiskXML(data *bytes.Buffer) (map[string]any, error) {
	decoder := xml.NewDecoder(data)
	decoder.Strict = false

	// check for initial <?xml ... ?> tag.
	xmlVer, err := tokenSkipWhitespace(decoder)
	if err != nil {
		return map[string]any{}, err
	}
	if err = isXMLProcInst(xmlVer, "xml"); err != nil {
		return map[string]any{}, err
	}

	// check for initial <!DOCTYPE ... > tag.
	doctype, err := tokenSkipWhitespace(decoder)
	if err != nil {
		return map[string]any{}, err
	}
	if err = isXMLDirective(doctype); err != nil {
		return map[string]any{}, err
	}

	// check for initial <plist ...> tag.
	plistTok, err := tokenSkipWhitespace(decoder)
	if err != nil {
		return map[string]any{}, err
	}
	plist, err := getXMLStartElement(plistTok)
	if err != nil {
		return map[string]any{}, err
	}
	if plist.Name.Local != "plist" {
		return map[string]any{}, fmt.Errorf("XML had \"%s\" instead of \"plist\"", plist.Name.Local)
	}

	// check that the first value is a <dict>.
	dTok, err := tokenSkipWhitespace(decoder)
	if err != nil {
		return map[string]any{}, err
	}
	d, err := getXMLStartElement(dTok)
	if err != nil {
		return map[string]any{}, err
	}
	if d.Name.Local != "dict" {
		return map[string]any{}, fmt.Errorf("XML had \"%s\" instead of initial \"dict\"", d.Name.Local)
	}

	return parsePListDict(d, decoder)
}

// parsePListString converts a PList <string> to a string.
func parsePListString(start xml.StartElement, dec *xml.Decoder) (string, error) {
	v := struct {
		Val string `xml:",chardata"`
	}{}
	err := dec.DecodeElement(&v, &start)
	return v.Val, err
}

// parsePListString converts a PList <integer> to a int.
func parsePListInt(start xml.StartElement, dec *xml.Decoder) (int, error) {
	v := struct {
		Val int `xml:",chardata"`
	}{}
	err := dec.DecodeElement(&v, &start)
	return v.Val, err
}

// parsePListArray converts a PList <array> to an array of values.
func parsePListArray(start xml.StartElement, dec *xml.Decoder) (out []any, err error) {
	end := start.End()
	out = []any{}
	for {
		curTok, err := tokenSkipWhitespace(dec)
		if err != nil {
			return out, err
		}
		if curTok == nil {
			return out, errors.New("unexpected EOF while parsing array")
		}
		if curTok == end {
			break
		}

		cur, err := getXMLStartElement(curTok)
		if err != nil {
			return out, err
		}

		d, err := parsePListValue(cur, dec)
		if err != nil {
			return out, err
		}
		out = append(out, d)
	}

	return out, nil
}

// parsePListDict converts a PList <dict> to a map of string to values.
func parsePListDict(start xml.StartElement, dec *xml.Decoder) (out map[string]any, err error) {
	end := start.End()
	out = map[string]any{}
	for {
		// get the <key> tag.
		curTok, err := tokenSkipWhitespace(dec)
		if err != nil {
			return out, err
		}
		if curTok == nil {
			return out, errors.New("unexpected EOF while parsing dict key")
		}
		if curTok == end {
			break
		}

		cur, err := getXMLStartElement(curTok)
		if err != nil {
			return out, err
		}
		if cur.Name.Local != "key" {
			return out, errors.New("unexpected element while parsing dict key")
		}
		key, err := parsePListString(cur, dec)
		if err != nil {
			return out, err
		}

		// get the value tag.
		curTok, err = tokenSkipWhitespace(dec)
		if err != nil {
			return out, err
		}
		if curTok == nil {
			return out, errors.New("unexpected EOF while parsing dict value")
		}
		if curTok == end {
			return out, errors.New("unexpected end element while parsing dict value")
		}
		cur, err = getXMLStartElement(curTok)
		if err != nil {
			return out, err
		}

		d, err := parsePListValue(cur, dec)
		if err != nil {
			return out, err
		}

		if _, ok := out[key]; ok {
			return out, fmt.Errorf("dict contained duplicate keys %s", key)
		}
		out[key] = d
	}

	return out, nil
}

// parsePListValue converts a PList value to a value.
func parsePListValue(start xml.StartElement, dec *xml.Decoder) (any, error) {
	switch start.Name.Local {
	case "integer":
		return parsePListInt(start, dec)
	case "string":
		return parsePListString(start, dec)
	case "array":
		return parsePListArray(start, dec)
	case "dict":
		return parsePListDict(start, dec)
	case "key":
		return nil, errors.New("unexpected key while parsing value")
	default:
		return nil, fmt.Errorf("unexpected XML element %s while parsing", start.Name.Local)
	}
}

var screenResolutionRegex *regexp.Regexp = regexp.MustCompile(`^([0-9]+)\s*x\s*([0-9]+)\s*@\s*([0-9]+(?:\.[0-9]+)?)\s*Hz\s*$`)
var screenPhysicalRegex *regexp.Regexp = regexp.MustCompile(`^\s*spdisplays_([0-9]+x[0-9]+).*$`)

func (s Collector) collectScreens(platform.Info) ([]screen, error) {
	stdout, stderr, err := cmdutils.RunWithTimeout(context.Background(), 15*time.Second, s.platform.screenCmd[0], s.platform.screenCmd[1:]...)
	if err != nil {
		return []screen{}, fmt.Errorf("failed to run system_profiler: %v", err)
	}
	if stderr.Len() > 0 {
		s.log.Info("system_profiler output to stderr", "stderr", stderr)
	}

	var data gpuAndScreens
	err = json.Unmarshal(stdout.Bytes(), &data)
	if err != nil {
		return []screen{}, err
	}

	out := []screen{}
	for _, g := range data.Gpus {
		for _, display := range g.Displays {
			if display.Online != "spdisplays_yes" {
				continue
			}

			scr := screen{}

			m := screenResolutionRegex.FindStringSubmatch(display.ResolutionRefresh)
			if len(m) != 4 {
				s.log.Warn("display resolution and refresh formatted wrong", "value", display.ResolutionRefresh)
			} else {
				scr.Resolution = m[1] + "x" + m[2]
				scr.RefreshRate = m[3]
			}

			m = screenPhysicalRegex.FindStringSubmatch(display.PhysicalResolution)
			if len(m) != 2 {
				s.log.Warn("display physical resolution formatted wrong", "value", display.PhysicalResolution)
			} else {
				scr.PhysicalResolution = m[1]
			}

			out = append(out, scr)
		}
	}

	return out, nil
}
