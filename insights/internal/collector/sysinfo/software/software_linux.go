package software

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/ubuntu/ubuntu-insights/common/fileutils"
	"github.com/ubuntu/ubuntu-insights/insights/internal/collector/sysinfo/platform"
	"go.yaml.in/yaml/v3"
)

type platformOptions struct {
	root        string
	langFunc    func() (string, bool)
	snapEnvFunc func() string
}

// defaultPlatformOptions returns options for when running under a normal environment.
func defaultPlatformOptions() platformOptions {
	return platformOptions{
		root: "/",
		langFunc: func() (string, bool) {
			return os.LookupEnv("LANG")
		},
		snapEnvFunc: func() string {
			return os.Getenv("SNAP")
		},
	}
}

// isConfinedSnap returns true if running inside a strictly confined or devmode snap.
// It checks for meta/snap.yaml within the given snap directory and verifies the confinement.
func isConfinedSnap(snapDir string) bool {
	if snapDir == "" {
		return false
	}

	content, err := os.ReadFile(filepath.Join(snapDir, "meta", "snap.yaml"))
	if err != nil {
		return false
	}

	var meta struct {
		Confinement string `yaml:"confinement"`
	}
	if err := yaml.Unmarshal(content, &meta); err != nil {
		return false
	}

	return meta.Confinement == "strict" || meta.Confinement == "devmode"
}

// osReleaseFields maps os-release keys to the fields we use.
var osReleaseFields = map[string]string{
	"ID":         "Distribution ID",
	"NAME":       "Name",
	"VERSION_ID": "Release",
}

// lsbReleaseFields maps lsb-release keys to our internal fields.
var lsbReleaseFields = map[string]string{
	"DISTRIB_ID":      "Distribution ID",
	"DISTRIB_RELEASE": "Release",
}

func (s Collector) collectOS() (osInfo, error) {
	// When running inside a confined snap, /etc/lsb-release contains the
	// host distro information and doesn't require additional snap interfaces.
	if isConfinedSnap(s.platform.snapEnvFunc()) {
		s.log.Debug("Detected confined snap environment, prioritizing lsb-release for OS information")
		lsbPath := filepath.Join(s.platform.root, "etc/lsb-release")
		if _, err := os.Stat(lsbPath); err == nil {
			info, err := s.collectOSFromLSBRelease(lsbPath)
			if err == nil {
				return info, nil
			}
			s.log.Debug("lsb-release parse failed, falling back to os-release", "error", err)
		}
	}

	// Per os-release(5), /etc takes priority over /usr/lib.
	// Snap hostfs paths take priority over local paths for confined snap compatibility.
	// Last existing path wins.
	paths := []string{
		filepath.Join(s.platform.root, "usr/lib/os-release"),
		filepath.Join(s.platform.root, "etc/os-release"),
		filepath.Join(s.platform.root, "var/lib/snapd/hostfs/usr/lib/os-release"),
		filepath.Join(s.platform.root, "var/lib/snapd/hostfs/etc/os-release"),
	}

	var osReleasePath string
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			osReleasePath = p
		}
	}

	if osReleasePath == "" {
		return osInfo{}, errors.New("no os-release file found")
	}

	return s.collectOSFromFile(osReleasePath)
}

// parseKeyValueFile parses a key=value file (like os-release or lsb-release),
// mapping keys through fieldMap and returning the resulting data.
func parseKeyValueFile(content []byte, fieldMap map[string]string) map[string]string {
	data := map[string]string{}
	for line := range strings.SplitSeq(string(content), "\n") {
		line = strings.TrimSpace(line)

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, "\"'")

		if field, exists := fieldMap[key]; exists {
			data[field] = value
		}
	}
	return data
}

// collectOSFromLSBRelease reads and parses an lsb-release file.
func (s Collector) collectOSFromLSBRelease(path string) (osInfo, error) {
	s.log.Debug("Collecting OS information from lsb-release", "path", path)

	content, err := os.ReadFile(path)
	if err != nil {
		return osInfo{}, fmt.Errorf("failed to read %s: %v", path, err)
	}

	data := parseKeyValueFile(content, lsbReleaseFields)

	if len(data) == 0 {
		return osInfo{}, fmt.Errorf("%s contained no usable data", path)
	}

	if _, ok := data["Distribution ID"]; !ok {
		s.log.Warn("lsb-release file missing DISTRIB_ID field", "path", path)
	}
	if _, ok := data["Release"]; !ok {
		s.log.Warn("lsb-release file missing DISTRIB_RELEASE field", "path", path)
	}

	return osInfo{
		Family:  runtime.GOOS,
		Distro:  data["Distribution ID"],
		Version: data["Release"],
	}, nil
}

// collectOSFromFile reads and parses an os-release file.
func (s Collector) collectOSFromFile(path string) (osInfo, error) {
	s.log.Debug("Collecting OS information from file", "path", path)

	content, err := os.ReadFile(path)
	if err != nil {
		return osInfo{}, fmt.Errorf("failed to read %s: %v", path, err)
	}

	data := parseKeyValueFile(content, osReleaseFields)

	if len(data) == 0 {
		return osInfo{}, fmt.Errorf("%s contained no usable data", path)
	}

	if _, ok := data["Distribution ID"]; !ok {
		s.log.Warn("os-release file missing ID field", "path", path)
	}
	if _, ok := data["Release"]; !ok {
		s.log.Warn("os-release file missing VERSION_ID field", "path", path)
	}

	// Match lsb_release behavior: capitalize first letter of ID, then
	// prefer NAME if it differs from ID only in capitalization.
	distro := data["Distribution ID"]
	if distro != "" {
		r, size := utf8.DecodeRuneInString(distro)
		distro = string(unicode.ToUpper(r)) + distro[size:]
	}
	if name, ok := data["Name"]; ok && strings.EqualFold(distro, name) {
		distro = name
	}

	return osInfo{
		Family:  runtime.GOOS,
		Distro:  distro,
		Version: data["Release"],
	}, nil
}

func (s Collector) collectLang() (string, error) {
	lang, ok := s.platform.langFunc()
	if !ok {
		return lang, errors.New("LANG environment variable not set")
	}

	l, _, _ := strings.Cut(lang, ".")
	return l, nil
}

func (s Collector) collectBios(pi platform.Info) (bios, error) {
	if pi.WSL.SubsystemVersion != 0 {
		s.log.Debug("skipping BIOS info collection on WSL")
		return bios{}, nil
	}

	info := bios{
		Vendor:  fileutils.ReadFileLogError(filepath.Join(s.platform.root, "sys/class/dmi/id/bios_vendor"), s.log),
		Version: fileutils.ReadFileLogError(filepath.Join(s.platform.root, "sys/class/dmi/id/bios_version"), s.log),
	}

	if strings.ContainsRune(info.Vendor, '\n') {
		s.log.Warn("BIOS info vendor contains invalid value")
		info.Vendor = ""
	}
	if strings.ContainsRune(info.Version, '\n') {
		s.log.Warn("BIOS info version contains invalid value")
		info.Version = ""
	}

	return info, nil
}
