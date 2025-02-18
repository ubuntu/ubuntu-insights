// Package software handles collecting "common" software information for all insight reports.
package software

import (
	"log/slog"
	"runtime"
	"time"
)

// Info is the software specific part.
type Info struct {
	OS       osInfo `json:"os"`
	Timezone string `json:"timezone"`
	Lang     string `json:"language"`
	Bios     bios   `json:"bios"`
}

// Source is info about the collection source.
type Source struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type osInfo struct {
	Family  string `json:"family"`
	Distro  string `json:"distribution"`
	Version string `json:"version"`
	Edition string `json:"edition,omitempty"`
}

type bios struct {
	Vendor  string `json:"vendor"`
	Version string `json:"version"`
}

// Collector handles dependencies for collecting software information.
// Collector implements CollectorT[software.Info].
type Collector struct {
	Src     Source
	Trigger string

	log      *slog.Logger
	timezone func() string
	platform platformOptions
}

// Options are the variadic options available to the Collector.
type Options func(*options)

type options struct {
	log      *slog.Logger
	timezone func() string

	platform platformOptions
}

// New returns a new Collector.
func New(args ...Options) Collector {
	opts := &options{
		log: slog.Default(),
		timezone: func() string {
			zone, _ := time.Now().Zone()
			return zone
		},
	}
	opts.platform = defaultPlatformOptions()
	for _, opt := range args {
		opt(opts)
	}

	return Collector{
		log:      opts.log,
		timezone: opts.timezone,
		platform: opts.platform,
	}
}

// Collect aggregates the data from all the other software collect functions.
func (s Collector) Collect() (info Info, err error) {
	s.log.Debug("collecting software info")

	info.Timezone = s.timezone()

	info.OS, err = s.collectOS()
	if err != nil {
		s.log.Warn("failed to collect OS info", "error", err)
		info.OS = osInfo{
			Family: runtime.GOOS,
		}
	}

	info.Lang, err = s.collectLang()
	if err != nil {
		s.log.Warn("failed to collect language info", "error", err)
	}

	info.Bios, err = s.collectBios()
	if err != nil {
		s.log.Warn("failed to collect BIOS info", "error", err)
	}

	return info, nil
}
