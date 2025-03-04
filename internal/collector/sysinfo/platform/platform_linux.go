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
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// Info contains platform information for Linux.
type Info struct {
	WSL         WSL  `json:"wsl,omitempty"`
	ProAttached bool `json:"proAttached,omitempty"`
}

type WSL struct {
	WSL              bool   `json:"wsl,omitempty"`
	WSLInterop       string `json:"wslInterop,omitempty"`
	WSLVersion       string `json:"wslVersion,omitempty"`
	WSLKernelVersion string `json:"wslKernelVersion,omitempty"`
}

type platformOptions struct {
	root          string
	interopEnv    string
	detectVirtCmd []string
	wslVersionCmd []string
	proStatusCmd  []string
}

// defaultOptions returns options for when running under a normal environment.
func defaultPlatformOptions() platformOptions {
	return platformOptions{
		root:          "/",
		interopEnv:    "WSL_INTEROP",
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
// This is done by checking for the presence of the WSL_INTEROP environment variable,
// and by checking if /proc/sys/fs/binfmt_misc/WSLInterop exists with enabled.
func (p Collector) interopEnabled() bool {
	if os.Getenv(p.platform.interopEnv) == "" {
		return false
	}

	if _, err := os.Stat(filepath.Join(p.platform.root, "proc/sys/fs/binfmt_misc/WSLInterop")); err != nil {
		return false
	}

	// Read contents of "proc/sys/fs/binfmt_misc/WSLInterop", check if first line is "enabled"
	contents, err := os.ReadFile(filepath.Join(p.platform.root, "proc/sys/fs/binfmt_misc/WSLInterop"))
	if err != nil {
		p.log.Warn("failed to read WSLInterop", "error", err)
		return false
	}

	// Check if first line of file is "enabled"
	lines := strings.Split(string(contents), "\n")
	if !(len(lines) > 0 && lines[0] == "enabled") {
		return false
	}

	p.log.Debug("WSL interop enabled")
	return true
}

// collectWSL collects information about Windows Subsystem for Linux.
func (p Collector) collectWSL() WSL {
	if !p.isWSL() {
		return WSL{}
	}

	info := WSL{
		WSL: true,
	}

	if !p.interopEnabled() {
		info.WSLInterop = "disabled"
		return info
	}
	info.WSLInterop = "enabled"

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
		"WSL":    &info.WSLVersion,
		"Kernel": &info.WSLKernelVersion}
	for entry, value := range entries {
		regex := getWSLRegex(entry)
		matches := regex.FindStringSubmatch(data)
		if len(matches) < 2 {
			p.log.Warn("failed to parse WSL version", "entry", entry)
			continue
		}
		*value = matches[1]
		if len(matches) > 2 {
			p.log.Warn(fmt.Sprintf("parsed multiple %s versions", entry), "matches", matches)
		}
	}

	return info
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
	println(stdout.String())
	err = json.Unmarshal(stdout.Bytes(), &proStatus)
	if err != nil {
		p.log.Warn("failed to parse pro api return", "error", err)
		return false
	}
	return proStatus.Data.Attributes.IsAttached
}
