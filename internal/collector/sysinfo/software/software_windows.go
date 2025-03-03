package software

import (
	"runtime"

	"github.com/ubuntu/ubuntu-insights/internal/cmdutils"
	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/platform"
)

type platformOptions struct {
	osCmd   []string
	langCmd []string
	biosCmd []string
}

func defaultPlatformOptions() platformOptions {
	return platformOptions{
		osCmd:   []string{"powershell.exe", "-Command", "Get-CimInstance", "Win32_OperatingSystem", "|", "Format-List", "*"},
		langCmd: []string{"powershell.exe", "-Command", "Get-WinSystemLocale", "|", "Format-List", "IetfLanguageTag"},
		biosCmd: []string{"powershell.exe", "-Command", "Get-CimInstance", "Win32_BIOS", "|", "Format-List"},
	}
}

var usedOSFields = map[string]struct{}{
	"Caption":            {},
	"Version":            {},
	"OperatingSystemSKU": {},
}

func (s Collector) collectOS() (osInfo, error) {
	os, err := cmdutils.RunListFmt(s.platform.osCmd, usedOSFields, s.log)
	if err != nil {
		return osInfo{}, err
	}

	if len(os) > 1 {
		s.log.Warn("multiple operating systems were reported")
	}

	return osInfo{
		Family:  runtime.GOOS,
		Distro:  os[0]["Caption"],
		Version: os[0]["Version"],
		Edition: os[0]["OperatingSystemSKU"],
	}, nil
}

func (s Collector) collectLang() (string, error) {
	lang, err := cmdutils.RunListFmt(s.platform.langCmd, nil, s.log)
	if err != nil {
		return "", err
	}

	if len(lang) > 1 {
		s.log.Warn("multiple system locales were reported")
	}

	return lang[0]["IetfLanguageTag"], nil
}

var usedBIOSFields = map[string]struct{}{
	"Manufacturer": {},
	"Version":      {},
}

func (s Collector) collectBios(platform.Info) (bios, error) {
	b, err := cmdutils.RunListFmt(s.platform.biosCmd, usedBIOSFields, s.log)
	if err != nil {
		return bios{}, err
	}

	if len(b) > 1 {
		s.log.Warn("multiple BIOS were reported")
	}

	return bios{
		Vendor:  b[0]["Manufacturer"],
		Version: b[0]["Version"],
	}, nil
}
