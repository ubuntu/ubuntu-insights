package software

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/ubuntu/ubuntu-insights/internal/cmdutils"
)

type options struct {
	osCmd    []string
	langFunc func() (string, bool)

	timezone func() string
	log      *slog.Logger
}

// defaultOptions returns options for when running under a normal environment.
func defaultOptions() *options {
	return &options{
		osCmd: []string{"lsb_release", "-a"},
		langFunc: func() (string, bool) {
			return os.LookupEnv("LANG")
		},

		timezone: func() string {
			zone, _ := time.Now().Zone()
			return zone
		},
		log: slog.Default(),
	}
}

// osInfoRegex matches lines in the form `key` : `value`.
var osInfoRegex = regexp.MustCompile(`(?m)^\s*(.+?)\s*:\s*(.+?)\s*$`)

var usedOSFields = map[string]struct{}{
	"Distributor ID": {},
	"Release":        {},
}

func (s Collector) collectOS() (info osInfo, err error) {
	stdout, stderr, err := cmdutils.RunWithTimeout(context.Background(), 15*time.Second, s.opts.osCmd[0], s.opts.osCmd[1:]...)
	if err != nil {
		return nil, fmt.Errorf("failed to run lsb_release: %v", err)
	}
	if stderr.Len() > 0 {
		s.opts.log.Info("lsb_release output to stderr", "stderr", stderr)
	}

	info = osInfo{
		"Family": runtime.GOOS,
	}

	entries := osInfoRegex.FindAllStringSubmatch(stdout.String(), -1)
	for _, entry := range entries {
		if _, ok := usedOSFields[entry[1]]; ok {
			info[entry[1]] = entry[2]
		}
	}

	if len(info) == 1 {
		s.opts.log.Warn("lsb_release contained invalid data")
	}

	return info, nil
}

func (s Collector) collectLang() (string, error) {
	lang, ok := s.opts.langFunc()
	if !ok {
		return lang, errors.New("LANG environment variable not set")
	}

	l, _, _ := strings.Cut(lang, ".")
	return l, nil
}
