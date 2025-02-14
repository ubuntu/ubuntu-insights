package software

import (
	"runtime"

	"github.com/ubuntu/ubuntu-insights/internal/cmdutils"
)

type platformOptions struct {
	osCmd []string
}

func defaultPlatformOptions() platformOptions {
	return platformOptions{
		osCmd: []string{"sw_vers"},
	}
}

func (s Collector) collectOS() (osInfo, error) {
	os, err := cmdutils.RunListFmt(s.platform.osCmd, nil, s.log)
	if err != nil {
		return osInfo{
			Family: runtime.GOOS,
		}, err
	}
	if len(os) != 1 {
		s.log.Warn("os info contained multiple oses", "value", len(os))
	}

	return osInfo{
		Family:  runtime.GOOS,
		Distro:  os[0]["ProductName"],
		Version: os[0]["ProductVersion"],
		Edition: os[0]["BuildVersion"],
	}, nil
}

func (s Collector) collectLang() (string, error) {
	return "", nil
}

func (s Collector) collectBios() (bios, error) {
	return bios{}, nil
}
