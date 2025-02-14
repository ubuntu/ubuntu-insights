package software

import (
	"context"
	"runtime"
	"time"

	"github.com/ubuntu/ubuntu-insights/internal/cmdutils"
)

type platformOptions struct {
	osCmd   []string
	langCmd []string
}

func defaultPlatformOptions() platformOptions {
	return platformOptions{
		osCmd:   []string{"sw_vers"},
		langCmd: []string{"defaults", "read", "-g", "AppleLocale"},
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
	stdout, stderr, err := cmdutils.RunWithTimeout(context.Background(), 15*time.Second, s.platform.langCmd[0], s.platform.langCmd[1:]...)
	if err != nil {
		return "", err
	}
	if stderr.Len() > 0 {
		s.log.Info("locale command output to stderr", "stderr", stderr)
	}

	return stdout.String(), nil
}

func (s Collector) collectBios() (bios, error) {
	return bios{}, nil
}
