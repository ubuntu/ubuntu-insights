// Package software handles collecting "common" software information for all insight reports.
package software

import "runtime"

// Info is the software specific part.
type Info struct {
	OS       osInfo
	Src      Source
	Type     string
	Timezone string
	Lang     string
}

// Source is info about the collection source.
type Source struct {
	Name    string
	Version string
}

// Theses types represent how collection was triggered.
const (
	TypeRegular = "regular"
	TypeInstall = "install"
	TypeManual  = "manual"
)

type osInfo = map[string]string

// Collector handles dependencies for collecting software information.
// Collector implements CollectorT[software.Info].
type Collector struct {
	Src  Source
	Type string

	opts options
}

// Options are the variadic options available to the Collector.
type Options func(*options)

// New returns a new Collector.
// Since "type" is a keyword, the parameter is tipe instead.
func New(source Source, tipe string, args ...Options) Collector {
	// options defaults are platform dependent.
	opts := defaultOptions()
	for _, opt := range args {
		opt(opts)
	}

	return Collector{
		Src:  source,
		Type: tipe,

		opts: *opts,
	}
}

// Collect aggregates the data from all the other software collect functions.
func (s Collector) Collect() (info Info, err error) {
	s.opts.log.Debug("collecting software info")

	info.Src = s.Src
	info.Type = s.Type
	info.Timezone = s.opts.timezone()

	info.OS, err = s.collectOS()
	if err != nil {
		s.opts.log.Warn("failed to collect OS info", "error", err)
		info.OS = osInfo{
			"Family": runtime.GOOS,
		}
	}

	info.Lang, err = s.collectLang()
	if err != nil {
		s.opts.log.Warn("failed to collect language info", "error", err)
	}

	return info, nil
}
