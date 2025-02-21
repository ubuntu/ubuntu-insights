package software

import (
	"context"
	"errors"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/ubuntu/ubuntu-insights/internal/cmdutils"
)

type platformOptions struct {
	osCmd   []string
	langCmd []string
	biosCmd []string
}

func defaultPlatformOptions() platformOptions {
	return platformOptions{
		osCmd:   []string{"sw_vers"},
		langCmd: []string{"defaults", "read", "-g", "AppleLocale"},
		biosCmd: []string{"system_profiler", "SPHardwareDataType", "-detailLevel", "mini"},
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

	l := strings.TrimSpace(stdout.String())
	if l == "" {
		return "", errors.New("locale was empty")
	}

	return l, nil
}

var biosRegex = regexp.MustCompile(`(?m)^\s*Boot ROM Version\s*:\s*(.+?)\s*$`)

func (s Collector) collectBios() (bios, error) {
	stdout, stderr, err := cmdutils.RunWithTimeout(context.Background(), 15*time.Second, s.platform.biosCmd[0], s.platform.biosCmd[1:]...)
	if err != nil {
		return bios{}, err
	}
	if stderr.Len() > 0 {
		s.log.Info("BIOS command output to stderr", "stderr", stderr)
	}

	m := biosRegex.FindStringSubmatch(stdout.String())
	if len(m) != 2 {
		return bios{}, errors.New("failed to parse BIOS info")
	}

	return bios{
		Vendor:  "Apple",
		Version: m[1],
	}, nil
}
