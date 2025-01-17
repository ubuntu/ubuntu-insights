package sysinfo

import (
	"os"
	"path/filepath"
)

func (s Manager) collectHardware() (hwInfo HwInfo, err error) {
	// System vendor
	d, err := os.ReadFile(filepath.Join(s.root, "sys/class/dmi/id/sys_vendor"))
	if err != nil {
		return HwInfo{}, err
	}
	hwInfo.Product = map[string]string{
		"Vendor": string(d),
	}

	return hwInfo, nil
}

func (s Manager) collectSoftware() (swInfo SwInfo, err error) {
	return swInfo, nil
}
