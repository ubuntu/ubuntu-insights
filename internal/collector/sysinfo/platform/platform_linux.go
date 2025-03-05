package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ubuntu/decorate"
	"github.com/ubuntu/ubuntu-insights/internal/cmdutils"
	"github.com/ubuntu/ubuntu-insights/internal/fileutils"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// Info contains platform information for Linux.
type Info struct {
	WSL         WSL  `json:"wsl"`
	ProAttached bool `json:"proAttached,omitempty"`
}

// WSL contains platform information specific to Windows Subsystem for Linux.
type WSL struct {
	WSL           uint8  `json:"wsl,omitzero"`
	Interop       string `json:"wslInterop,omitempty"`
	Version       string `json:"wslVersion,omitempty"`
	KernelVersion string `json:"wslKernelVersion,omitempty"`
}

type platformOptions struct {
	root          string
	detectVirtCmd []string
	wslVersionCmd []string
	proStatusCmd  []string
}

// defaultOptions returns options for when running under a normal environment.
func defaultPlatformOptions() platformOptions {
	return platformOptions{
		root:          "/",
		detectVirtCmd: []string{"systemd-detect-virt"},
		wslVersionCmd: []string{"wsl.exe", "-v"},
		proStatusCmd:  []string{"pro", "api", "u.pro.status.is_attached.v1"},
	}
}

func (p Collector) collectPlatform() (info Info, err error) {
	defer func() {
		decorate.OnError(&err, "failed to collect platform information")
	}()
	info.WSL = p.collectWSL()
	info.ProAttached = p.isProAttached()

	return info, nil
}

// isWSL returns true if the system is running under Windows Subsystem for Linux.
// This is done by checking for the presence of `/proc/sys/fs/binfmt_misc/WSLInterop`.
func (p Collector) isWSL() bool {
	stdout, stderr, err := cmdutils.RunWithTimeout(context.Background(), 15*time.Second, p.platform.detectVirtCmd[0], p.platform.detectVirtCmd[1:]...)
	if err != nil {
		p.log.Warn("failed to run systemd-detect-virt", "error", err)
		return false
	}
	if stderr.Len() > 0 {
		p.log.Info("systemd-detect-virt output to stderr", "stderr", stderr)
	}

	if strings.Contains(stdout.String(), "wsl") {
		p.log.Debug("WSL detected")
		return true
	}

	return false
}

// interopEnabled returns true if WSL interop is enabled.
// It does this by checking the WSLInterop or WSLInterop-late, depending on the detected WSL version.
func (p Collector) interopEnabled() (enabled bool) {
	var path string
	switch p.getWSLVersion() {
	case 1:
		{
			// Check for the presence of /proc/sys/fs/binfmt_misc/WSLInterop
			path = filepath.Join(p.platform.root, "proc/sys/fs/binfmt_misc/WSLInterop")
		}
	case 2:
		{
			// Check for the presence of /proc/sys/fs/binfmt_misc/WSLInterop-late
			path = filepath.Join(p.platform.root, "proc/sys/fs/binfmt_misc/WSLInterop-late")
		}
	default:
		return false
	}
	// If case default, then WSL is not detected, and no log should be written.
	defer func() {
		if enabled {
			p.log.Debug("WSL interop detected enabled")
			return
		}
		p.log.Debug("WSL interop detected disabled")
	}()

	_, err := os.Stat(path)
	if err != nil {
		return false
	}

	// Check if the first line of the file is 'enabled'
	data := fileutils.ReadFileLogError(path, p.log)
	lines := strings.Split(data, "\n")
	return (len(lines) > 0 && lines[0] == "enabled")
}

// collectWSL collects information about Windows Subsystem for Linux.
func (p Collector) collectWSL() WSL {
	if !p.isWSL() {
		return WSL{}
	}

	info := WSL{
		WSL: p.getWSLVersion(),
	}

	if !p.interopEnabled() {
		info.Interop = "disabled"
		return info
	}
	info.Interop = "enabled"

	// Run `wsl.exe -v` and parse it
	stdout, stderr, err := cmdutils.RunWithTimeout(context.Background(), 15*time.Second, p.platform.wslVersionCmd[0], p.platform.wslVersionCmd[1:]...)
	if err != nil {
		p.log.Warn("failed to run wsl.exe -v", "error", err)
		return info
	}
	if stderr.Len() > 0 {
		p.log.Info("wsl output to stderr", "stderr", stderr)
	}

	// Assume little endian on Windows.
	utf16Reader := transform.NewReader(stdout, unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder())
	decodedData, err := io.ReadAll(utf16Reader)
	if err != nil {
		p.log.Warn("failed to decode UTF-16 data", "error", err)
		return info
	}

	data := string(decodedData)

	entries := map[string]*string{
		"WSL":    &info.Version,
		"Kernel": &info.KernelVersion}
	for entry, value := range entries {
		regex := getWSLRegex(entry)
		matches := regex.FindAllStringSubmatch(data, -1)
		if len(matches) == 0 {
			p.log.Warn("failed to parse WSL version", "entry", entry)
			continue
		}
		if len(matches) > 1 {
			p.log.Warn(fmt.Sprintf("parsed multiple %s versions, using the first", entry), "matches", matches)
		}
		*value = matches[0][1]
	}

	return info
}

// getWSLVersion returns the WSL version based on the kernel version naming convention.
// If the kernel version has Microsoft with a capital M, it is WSL 1.
// If the kernel version can't be read, or if the version doesn't match the pattern, it is assumed to be WSL 2.
// If not in WSL, it returns 0.
//
// This could potentially be fooled by a custom kernel with `Microsoft“ in the name.
func (p Collector) getWSLVersion() uint8 {
	if !p.isWSL() {
		return 0
	}

	kVersion := fileutils.ReadFileLogError(filepath.Join(p.platform.root, "proc/version"), p.log)
	if !strings.Contains(kVersion, `-Microsoft (Microsoft@Microsoft.com)`) {
		return 2
	}

	return 1
}

// getWSLRegex returns a regex for matching WSL version.
//
// The regex will look for lines matching '[entry] version: ' followed by non-whitespace characters.
// The version will be captured in a group as the second match. The first match will be the entire line.
//
// Take care that if there are any special characters in the entry, they are properly escaped.
func getWSLRegex(entry string) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprintf(`(?m)^\s*%s\s+version:\s+([\S.]+)\s*$`, entry))
}

// isProAttached returns the attach state of Ubuntu Pro.
func (p Collector) isProAttached() bool {
	stdout, stderr, err := cmdutils.RunWithTimeout(context.Background(), 15*time.Second, p.platform.proStatusCmd[0], p.platform.proStatusCmd[1:]...)
	if err != nil {
		p.log.Warn("failed to get pro status", "error", err)
		return false
	}
	if stderr.Len() > 0 {
		p.log.Info("pro api output to stderr", "stderr", stderr)
	}

	// Parse json output to get is_attached field
	var proStatus struct {
		Data struct {
			Attributes struct {
				IsAttached bool `json:"is_attached"`
			}
		}
	}
	err = json.Unmarshal(stdout.Bytes(), &proStatus)
	if err != nil {
		p.log.Warn("failed to parse pro api return", "error", err)
		return false
	}
	return proStatus.Data.Attributes.IsAttached
}
