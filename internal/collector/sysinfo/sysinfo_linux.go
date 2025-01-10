package sysinfo

import (
	"os"
	"path/filepath"
	"regexp"
)

// readFile returns the data in <file>, or "" on error.
func (s Manager) readFile(file string) string {
	d, err := os.ReadFile(file)
	if err != nil {
		s.log.Warn(err.Error())
		return ""
	}

	return string(d)
}

func (s Manager) collectProduct() map[string]string {
	return map[string]string{
		"Vendor": s.readFile(filepath.Join(s.root, "sys/class/dmi/id/sys_vendor")),
		"Name":   s.readFile(filepath.Join(s.root, "sys/class/dmi/id/product_name")),
		"Family": s.readFile(filepath.Join(s.root, "sys/class/dmi/id/product_family")),
	}
}

func (s Manager) collectGPU(card string) (info GpuInfo, err error) {
	cardDir, err := filepath.EvalSymlinks(filepath.Join(s.root, "sys/class/drm", card))
	if err != nil {
		return GpuInfo{}, err
	}

	devDir, err := filepath.EvalSymlinks(filepath.Join(cardDir, "device"))
	if err != nil {
		return GpuInfo{}, err
	}

	info.Gpu = make(map[string]string)

	info.Gpu["Vendor"] = s.readFile(filepath.Join(devDir, "vendor"))
	info.Gpu["Name"] = s.readFile(filepath.Join(devDir, "label"))

	driverLink, err := os.Readlink(filepath.Join(devDir, "driver"))
	if err == nil {
		info.Gpu["Driver"] = filepath.Base(driverLink)
	} else {
		s.log.Warn(err.Error())
	}

	return info, nil
}

var gpuSymlinkRegex *regexp.Regexp = regexp.MustCompile("^card[0-9]+$")

func (s Manager) collectGPUs() []GpuInfo {
	gpus := make([]GpuInfo, 0, 2)

	ds, err := os.ReadDir(filepath.Join(s.root, "sys/class/drm"))
	if err != nil {
		s.log.Warn(err.Error())
		return gpus
	}

	for _, d := range ds {
		n := d.Name()

		if !gpuSymlinkRegex.MatchString(n) {
			continue
		}

		gpu, err := s.collectGPU(n)
		if err != nil {
			s.log.Warn(err.Error())
			continue
		}

		gpus = append(gpus, gpu)
	}

	return gpus
}

func (s Manager) collectHardware() (hwInfo HwInfo, err error) {

	hwInfo.Product = s.collectProduct()
	hwInfo.Gpus = s.collectGPUs()

	return hwInfo, nil
}

func (s Manager) collectSoftware() (swInfo SwInfo, err error) {

	return swInfo, nil
}
