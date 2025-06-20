package platform

type platformOptions struct {
}

// defaultOptions returns options for when running under a normal environment.
func defaultPlatformOptions() platformOptions {
	return platformOptions{}
}

// Info contains platform information for Windows.
type Info struct {
}

func (p Collector) collectPlatform() (info Info, err error) {
	return info, nil
}
