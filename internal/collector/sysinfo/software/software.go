// Package software handles collecting "common" software information for all insight reports.
package software

// Info is the software specific part.
type Info struct{}

type options struct{}

// Collector handles dependencies for collecting software information.
// Collector implements CollectorT[software.Info].
type Collector struct {
	opts options
}

// Options are the variadic options available to the Collector.
type Options func(*options)

// New returns a new Collector.
func New(args ...Options) Collector {
	// options defaults are platform dependent.
	opts := &options{}
	for _, opt := range args {
		opt(opts)
	}

	return Collector{
		opts: *opts,
	}
}

// Collect aggregates the data from all the other software collect functions.
func (s Collector) Collect() (info Info, err error) {
	return info, err
}
