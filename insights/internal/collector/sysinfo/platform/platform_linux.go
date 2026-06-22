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

	"github.com/godbus/dbus/v5"
	"github.com/ubuntu/decorate"
	"github.com/ubuntu/ubuntu-insights/common/fileutils"
	"github.com/ubuntu/ubuntu-insights/insights/internal/cmdutils"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
	"gopkg.in/ini.v1"
)

// Ubuntu Pro D-Bus interface details, as exposed by ubuntu-advantage-desktop-daemon
// on the system bus. This is used in preference to the `pro` CLI for improved
// compatibility with confined snaps (via the ubuntu-pro-control interface).
const (
	proDBusDest  = "com.canonical.UbuntuAdvantage"
	proDBusPath  = "/com/canonical/UbuntuAdvantage/Manager"
	proDBusIface = "com.canonical.UbuntuAdvantage.Manager"
	proDBusProp  = proDBusIface + ".Attached"
)

// proDBusObject is the subset of dbus.BusObject used to read the attach state.
type proDBusObject interface {
	GetProperty(p string) (dbus.Variant, error)
}

// proDBusConn is the subset of *dbus.Conn used to read the attach state.
// It is an interface to allow injecting a fake connection in tests.
type proDBusConn interface {
	Object(dest string, path dbus.ObjectPath) proDBusObject
	Close() error
}

// systemBusConn wraps a *dbus.Conn to satisfy proDBusConn.
type systemBusConn struct {
	conn *dbus.Conn
}

func (c systemBusConn) Object(dest string, path dbus.ObjectPath) proDBusObject {
	return c.conn.Object(dest, path)
}

func (c systemBusConn) Close() error {
	return c.conn.Close()
}

// connectSystemBus connects to the system bus and returns a proDBusConn.
func connectSystemBus() (proDBusConn, error) {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return nil, err
	}
	return systemBusConn{conn: conn}, nil
}

// Info contains platform information for Linux.
type Info struct {
	WSL         WSL     `json:"wsl,omitzero"`
	Desktop     Desktop `json:"desktop,omitzero"`
	ProAttached bool    `json:"proAttached,omitempty"`
}

// WSL contains platform information specific to Windows Subsystem for Linux.
type WSL struct {
	SubsystemVersion uint8  `json:"subsystemVersion,omitzero"`
	Systemd          string `json:"systemd,omitempty"`
	Interop          string `json:"interop,omitempty"`
	Version          string `json:"version,omitempty"`
	KernelVersion    string `json:"kernelVersion,omitempty"`
}

// Desktop contains platform information for Linux desktop environments.
type Desktop struct {
	DesktopEnvironment string `json:"desktopEnvironment,omitempty"`
	SessionName        string `json:"sessionName,omitempty"`
	SessionType        string `json:"sessionType,omitempty"`
}

type platformOptions struct {
	root              string
	detectVirtCmd     []string
	systemdAnalyzeCmd []string
	wslVersionCmd     []string
	proStatusCmd      []string

	proDBusConnector func() (proDBusConn, error)

	getenv func(key string) string
}

// defaultOptions returns options for when running under a normal environment.
func defaultPlatformOptions() platformOptions {
	return platformOptions{
		root:              "/",
		detectVirtCmd:     []string{"systemd-detect-virt"},
		systemdAnalyzeCmd: []string{"systemd-analyze", "time", "--system"},
		wslVersionCmd:     []string{"wsl.exe", "-v"},
		proStatusCmd:      []string{"pro", "api", "u.pro.status.is_attached.v1"},

		proDBusConnector: connectSystemBus,

		getenv: os.Getenv,
	}
}

func (p Collector) collectPlatform() (info Info, err error) {
	defer func() {
		decorate.OnError(&err, "failed to collect platform information")
	}()
	info.WSL = p.collectWSL()
	if info.WSL.SubsystemVersion == 0 {
		info.Desktop = p.getDesktop()
	}
	info.ProAttached = p.isProAttached()

	return info, nil
}

// isWSL returns true if the system is running under Windows Subsystem for Linux.
// This is done by checking the output of systemd-detect-virt.
func (p Collector) isWSL() bool {
	stdout, stderr, err := cmdutils.RunWithTimeout(context.Background(), 15*time.Second, p.platform.detectVirtCmd[0], p.platform.detectVirtCmd[1:]...)
	if err != nil {
		if !strings.Contains(stdout.String(), "none") {
			p.log.Warn("failed to run systemd-detect-virt", "error", err)
		}
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
// It does this by reading /etc/wsl.conf.
// If /wtc/wsl.conf does not exist, it assumes the default behavior, interop is enabled.
//
// This function does not check if interop is disabled using an alternative methods.
func (p Collector) interopEnabled() bool {
	if p.getWSLSubsystemVersion() == 0 {
		return false
	}

	path := filepath.Join(p.platform.root, "etc/wsl.conf")
	cfg, err := ini.Load(path)
	if os.IsNotExist(err) {
		p.log.Debug("wsl.conf not found, assuming interop is enabled")
		return true
	}
	if err != nil {
		p.log.Warn("failed to read wsl.conf", "error", err)
		return false
	}

	// Check if interop is enabled
	iEnabled, err := cfg.Section("interop").Key("enabled").Bool()
	if err != nil {
		p.log.Debug("Failed to parse interop.enabled in wsl.conf, assuming default behavior True", "error", err)
		return true
	}

	return iEnabled
}

// collectWSL collects information about Windows Subsystem for Linux.
func (p Collector) collectWSL() WSL {
	info := WSL{SubsystemVersion: p.getWSLSubsystemVersion()}
	if info.SubsystemVersion == 0 {
		return info
	}

	// Get the kernel version
	info.KernelVersion = p.getKernelVersion()

	// Check if systemd was used during boot
	info.Systemd = "not used"
	if p.wasSystemdUsed() {
		info.Systemd = "used"
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

	data := strings.TrimSpace(string(decodedData))

	entries := map[string]*string{
		`WSL`: &info.Version}
	for entry, value := range entries {
		regex := getWSLRegex(entry)
		matches := regex.FindAllStringSubmatch(data, -1)
		if len(matches) == 0 {
			p.log.Warn("failed to parse wsl --version", "entry", entry)
			continue
		}
		if len(matches) > 1 {
			p.log.Debug(fmt.Sprintf("parsed multiple %s versions, using the first", entry), "matches", matches)
		}
		*value = matches[0][1]
	}

	return info
}

// getWSLSubsystemVersion returns the WSL subsystem version based on the kernel version naming convention.
// If the kernel version has '-Microsoft \(Microsoft@Microsoft\.com\)' with a capital M, it is WSL 1.
// If the kernel version can't be read, or if the version doesn't match the pattern, it is assumed to be WSL 2.
// If not in WSL, it returns 0.
//
// This could potentially be fooled by a custom kernel with '-Microsoft \(Microsoft@Microsoft\.com\)' in the name.
func (p Collector) getWSLSubsystemVersion() uint8 {
	if !p.isWSL() {
		return 0
	}

	kVersion := fileutils.ReadFileLogError(filepath.Join(p.platform.root, "proc/version"), p.log)
	if !strings.Contains(kVersion, `-Microsoft (Microsoft@Microsoft.com)`) {
		return 2
	}

	return 1
}

// getKernelVersion returns the kernel version of the system.
func (p Collector) getKernelVersion() string {
	k := fileutils.ReadFileLogError(filepath.Join(p.platform.root, "proc/version"), p.log)
	// The kernel version is the third word in the file.
	s := strings.Fields(k)
	if len(s) < 3 {
		p.log.Warn("failed to parse kernel version", "version", k)
		return ""
	}
	return s[2]
}

// getWSLRegex returns a regex for matching WSL version.
//
// The regex will look for lines matching '[entry] version: ' followed by non-whitespace characters.
// The version will be captured in a group as the second match. The first match will be the entire line.
//
// Take care that if there are any special characters in the entry, they are properly escaped.
func getWSLRegex(entry string) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprintf(`(?m)^\s*.*%s\s*.*[:|：]\s+([\S.]+)\s*$`, entry))
}

// wasSystemdUsed checks if systemd was used during boot.
// It executes the systemd-analyze command with a timeout of 15 seconds to determine if systemd was used.
//
// If the command outputs "System has not been booted with systemd as init system" to stderr, it returns false.
// If the command outputs anything else to stderr, it logs the output.
// If the command fails to execute, it logs the error and returns false.
// It returns true if systemd was used during boot, otherwise it returns false.
func (p Collector) wasSystemdUsed() bool {
	stdout, stderr, err := cmdutils.RunWithTimeout(context.Background(), 15*time.Second, p.platform.systemdAnalyzeCmd[0], p.platform.systemdAnalyzeCmd[1:]...)
	if strings.Contains(stderr.String(), "System has not been booted with systemd as init system") {
		return false
	}
	if err != nil {
		p.log.Warn("failed to run systemd-analyze", "error", err)
		return false
	}
	if stderr.Len() > 0 {
		p.log.Info("systemd-analyze output to stderr", "stderr", stderr)
	}

	if stdout.Len() == 0 {
		p.log.Warn("systemd-analyze stdout is empty")
		return false
	}

	if !strings.Contains(stdout.String(), "Startup finished in") {
		return false
	}
	return true
}

// getDesktop returns the desktop environment, session name, and session type in a Desktop struct.
func (p Collector) getDesktop() Desktop {
	return Desktop{
		DesktopEnvironment: p.platform.getenv("XDG_CURRENT_DESKTOP"),
		SessionName:        p.platform.getenv("XDG_SESSION_DESKTOP"),
		SessionType:        p.platform.getenv("XDG_SESSION_TYPE"),
	}
}

// isProAttached returns the attach state of Ubuntu Pro.
//
// It first attempts to query the state over D-Bus, which works inside confined
// snaps via the ubuntu-pro-control interface. If that fails (for example, the
// daemon is not installed or the system bus is unavailable), it falls back to
// the `pro` CLI.
func (p Collector) isProAttached() bool {
	attached, err := p.isProAttachedDBus()
	if err == nil {
		return attached
	}
	p.log.Debug("failed to get pro status over D-Bus, falling back to CLI", "error", err)

	attached, err = p.isProAttachedCLI()
	if err != nil {
		p.log.Warn("failed to get pro status", "error", err)
		return false
	}
	return attached
}

// isProAttachedDBus returns the attach state of Ubuntu Pro by reading the
// Attached property from the ubuntu-advantage-desktop-daemon over the system bus.
func (p Collector) isProAttachedDBus() (bool, error) {
	conn, err := p.platform.proDBusConnector()
	if err != nil {
		return false, fmt.Errorf("failed to connect to system bus: %v", err)
	}
	defer conn.Close()

	v, err := conn.Object(proDBusDest, proDBusPath).GetProperty(proDBusProp)
	if err != nil {
		return false, fmt.Errorf("failed to get %q property: %v", proDBusProp, err)
	}

	attached, ok := v.Value().(bool)
	if !ok {
		return false, fmt.Errorf("unexpected type %T for %q property", v.Value(), proDBusProp)
	}
	return attached, nil
}

// isProAttachedCLI returns the attach state of Ubuntu Pro using the `pro` CLI.
func (p Collector) isProAttachedCLI() (bool, error) {
	stdout, stderr, err := cmdutils.RunWithTimeout(context.Background(), 15*time.Second, p.platform.proStatusCmd[0], p.platform.proStatusCmd[1:]...)
	if err != nil {
		return false, fmt.Errorf("failed to run pro api: %v", err)
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
	if err := json.Unmarshal(stdout.Bytes(), &proStatus); err != nil {
		return false, fmt.Errorf("failed to parse pro api return: %v", err)
	}
	return proStatus.Data.Attributes.IsAttached, nil
}
