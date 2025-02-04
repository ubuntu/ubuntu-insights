package software

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"runtime"
	"time"

	"github.com/ubuntu/ubuntu-insights/internal/cmdutils"
)

type options struct {
	osCmd []string

	log *slog.Logger
}

// defaultOptions returns options for when running under a normal environment.
func defaultOptions() *options {
	return &options{
		osCmd: []string{"lsb_release", "-a"},

		log: slog.Default(),
	}
}

// osInfoRegex matches lines in the form `key` : `value`.
var osInfoRegex = regexp.MustCompile(`(?m)^\s*(.+?)\s*:\s*(.+?)\s*$`)

var usedOSFields = map[string]struct{}{
	"Distributor ID": {},
	"Release":        {},
}

func (s Collector) collectOS() (info os, err error) {
	stdout, stderr, err := cmdutils.RunWithTimeout(context.Background(), 15*time.Second, s.opts.osCmd[0], s.opts.osCmd[1:]...)
	if err != nil {
		return nil, fmt.Errorf("failed to run lsb_release: %v", err)
	}
	if stderr.Len() > 0 {
		s.opts.log.Info("lsb_release output to stderr", "stderr", stderr)
	}

	info = os{
		"Family": runtime.GOOS,
	}

	entries := osInfoRegex.FindAllStringSubmatch(stdout.String(), -1)
	for _, entry := range entries {
		if _, ok := usedOSFields[entry[1]]; ok {
			info[entry[1]] = entry[2]
		}
	}

	return info, nil
}
