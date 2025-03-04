package software

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/ubuntu/ubuntu-insights/internal/cmdutils"
	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/platform"
	"github.com/ubuntu/ubuntu-insights/internal/fileutils"
)

type platformOptions struct {
	root     string
	osCmd    []string
	langFunc func() (string, bool)
}

// defaultOptions returns options for when running under a normal environment.
func defaultPlatformOptions() platformOptions {
	return platformOptions{
		root:  "/",
		osCmd: []string{"lsb_release", "-a"},
		langFunc: func() (string, bool) {
			return os.LookupEnv("LANG")
		},
	}
}

// osInfoRegex matches lines in the form `key` : `value`.
var osInfoRegex = regexp.MustCompile(`(?m)^\s*(.+?)\s*:\s*(.+?)\s*$`)

var usedOSFields = map[string]struct{}{
	"Distributor ID": {},
	"Release":        {},
}

func (s Collector) collectOS() (osInfo, error) {
	stdout, stderr, err := cmdutils.RunWithTimeout(context.Background(), 15*time.Second, s.platform.osCmd[0], s.platform.osCmd[1:]...)
	if err != nil {
		return osInfo{}, fmt.Errorf("failed to run lsb_release: %v", err)
	}
	if stderr.Len() > 0 {
		s.log.Info("lsb_release output to stderr", "stderr", stderr)
	}

	data := map[string]string{}
	entries := osInfoRegex.FindAllStringSubmatch(stdout.String(), -1)
	for _, entry := range entries {
		if _, ok := usedOSFields[entry[1]]; ok {
			data[entry[1]] = entry[2]
		}
	}

	if len(data) != 0 {
		if _, ok := data["Distributor ID"]; !ok {
			s.log.Warn("lsb_release missing distributor data")
		}
		if _, ok := data["Release"]; !ok {
			s.log.Warn("lsb_release missing release data")
		}
	} else {
		s.log.Warn("lsb_release contained no data")
	}

	return osInfo{
		Family:  runtime.GOOS,
		Distro:  data["Distributor ID"],
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
	if pi.WSL.WSL {
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
