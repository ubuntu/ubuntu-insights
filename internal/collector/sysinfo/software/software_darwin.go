package software

type platformOptions struct {
}

func defaultPlatformOptions() platformOptions {
	return platformOptions{}
}

func (s Collector) collectOS() (info osInfo, err error) {
	return info, nil
}

func (s Collector) collectLang() (string, error) {
	return "", nil
}

func (s Collector) collectBios() (bios, error) {
	return bios{}, nil
}
