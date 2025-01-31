// Package software handles collecting "common" software information for all insight reports.
package software

// Collector describes a type that collects softwareware information.
type Collector interface {
	Collect() (Info, error)
}

// Info is the software specific part.
type Info struct{}

type options struct{}

// Manager handles dependencies for collecting software information.
// Manager implements software.Collector.
type Manager struct {
	opts options
}

// Options are the variadic options available to the manager.
type Options func(*options)

// New returns a new Manager.
func New(args ...Options) Manager {
	// options defaults are platform dependent.
	opts := &options{}
	for _, opt := range args {
		opt(opts)
	}

	return Manager{
		opts: *opts,
	}
}

// Collect aggregates the data from all the other software collect functions.
func (s Manager) Collect() (info Info, err error) {
	return info, err
}
