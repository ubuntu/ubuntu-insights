// Package software handles collecting "common" software information for all insight reports.
package software

import "runtime"

// Info is the software specific part.
type Info struct {
	OS   os
	Src  Source
	Type string
}

type Source struct {
	Name    string
	Version string
}

const (
	TypeRegular = "regular"
	TypeInstall = "install"
	TypeManual  = "manual"
)

type os = map[string]string

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

	info.OS, err = s.collectOS()
	if err != nil {
		s.opts.log.Warn("failed to collect OS info", "error", err)
		info.OS = os{
			"Family": runtime.GOOS,
		}
	}

	return info, nil
}
