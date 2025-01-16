package testutils

import (
	"io/fs"
	"os"
	"path/filepath"
)

// CopyDir copies the directory src to dest, symlinks are copied but not followed.
func CopyDir(src string, dest string) error {
	dir, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	dest = filepath.Join(dest, filepath.Base(src))

	err = os.MkdirAll(dest, 0750)
	if err != nil {
		return err
	}

	for _, f := range dir {
		s := filepath.Join(src, f.Name())
		d := filepath.Join(dest, f.Name())

		info, err := f.Info()
		if err != nil {
			return err
		}

		if info.Mode()&fs.ModeSymlink > 0 {
			l, err := os.Readlink(s)
			if err != nil {
				return err
			}

			err = os.Symlink(l, d)
			if err != nil {
				return err
			}
		} else if info.IsDir() {
			err := CopyDir(s, dest)
			if err != nil {
				return err
			}
		} else {
			i, err := os.ReadFile(s)
			if err != nil {
				return err
			}

			err = os.WriteFile(d, i, 0600)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
