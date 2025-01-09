package sysinfo

import (
	"os"
	"path/filepath"
)

// readSysFile returns the data in /sys/<file>, or "" on error.
func (s Manager) readSysFile(file string) string {
	d, err := os.ReadFile(filepath.Join(s.root, "sys", file))
	if err != nil {
		// @TODO log error
		return ""
	}

	return string(d)
}

func (s Manager) collectProduct() map[string]string {
	return map[string]string{
		"Vendor": s.readSysFile("class/dmi/id/sys_vendor"),
		"Name":   s.readSysFile("class/dmi/id/product_name"),
		"Family": s.readSysFile("class/dmi/id/product_family"),
	}
}

func (s Manager) collectHardware() (hwInfo HwInfo, err error) {

	hwInfo.Product = s.collectProduct()

	return hwInfo, nil
}

func (s Manager) collectSoftware() (swInfo SwInfo, err error) {

	return swInfo, nil
}
