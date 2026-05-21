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
)

type platformOptions struct {
	root     string
	langFunc func() (string, bool)
}

// defaultOptions returns options for when running under a normal environment.
func defaultPlatformOptions() platformOptions {
	return platformOptions{
		root: "/",
		langFunc: func() (string, bool) {
			return os.LookupEnv("LANG")
		},
	}
}

// osReleaseFields maps os-release keys to the fields we use.
var osReleaseFields = map[string]string{
	"ID":         "Distributor ID",
	"NAME":       "Name",
	"VERSION_ID": "Release",
}

func (s Collector) collectOS() (osInfo, error) {
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

// collectOSFromFile reads and parses an os-release file.
func (s Collector) collectOSFromFile(path string) (osInfo, error) {
	s.log.Debug("collecting OS information from file", "path", path)

	content, err := os.ReadFile(path)
	if err != nil {
		return osInfo{}, fmt.Errorf("failed to read %s: %v", path, err)
	}

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

		if field, exists := osReleaseFields[key]; exists {
			data[field] = value
		}
	}

	if len(data) == 0 {
		return osInfo{}, fmt.Errorf("%s contained no usable data", path)
	}

	if _, ok := data["Distributor ID"]; !ok {
		s.log.Warn("os-release file missing ID field", "path", path)
	}
	if _, ok := data["Release"]; !ok {
		s.log.Warn("os-release file missing VERSION_ID field", "path", path)
	}

	// Match lsb_release behavior: capitalize first letter of ID, then
	// prefer NAME if it differs from ID only in capitalization.
	distro := data["Distributor ID"]
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
