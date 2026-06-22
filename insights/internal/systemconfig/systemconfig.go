// Package systemconfig manages the system-wide configuration.
// The configuration is stored in a single TOML file accessible to all users but only writable by the administrator.
// Its first setting is the opt-out: when it is true, all collect and upload operations behave as if consent is
// false, regardless of per-user or per-source consent settings.
package systemconfig

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/ubuntu/decorate"
	"github.com/ubuntu/ubuntu-insights/common/fileutils"
	"github.com/ubuntu/ubuntu-insights/insights/internal/constants"
)

// Manager manages the system-wide configuration file.
type Manager struct {
	path string

	log *slog.Logger
}

// Config is the TOML representation of the system configuration file.
type Config struct {
	SystemOptOut bool `toml:"system_opt_out"`
}

// New returns a new Manager.
// path is the directory in which the system configuration file is stored.
func New(l *slog.Logger, path string) *Manager {
	return &Manager{log: l, path: path}
}

// IsOptedOut reports whether the system opt-out is currently active.
//
// If the configuration file or its parent directory does not exist, IsOptedOut returns false with no error,
// preserving backward compatibility with systems that have not set a system opt-out.
// A malformed configuration file returns an error.
func (m Manager) IsOptedOut() (bool, error) {
	var f Config
	_, err := toml.DecodeFile(m.getFile(), &f)
	if err != nil {
		var pe *os.PathError
		if errors.As(err, &pe) && errors.Is(pe.Err, os.ErrNotExist) {
			m.log.Debug("System config file not found, treating as not opted out", "file", m.getFile())
			return false, nil
		}
		return false, fmt.Errorf("failed to read system config file: %v", err)
	}

	m.log.Debug("Read system config file", "file", m.getFile(), "system_opt_out", f.SystemOptOut)
	return f.SystemOptOut, nil
}

// SetOptOut writes the system opt-out state to the configuration file.
//
// The parent directory is created with permissions 0755 if it does not exist,
// and the file is written with permissions 0644 so that all users can read it
// but only privileged users can write it.
func (m Manager) SetOptOut(state bool) (err error) {
	defer decorate.OnError(&err, "could not set system opt-out state")

	f := Config{SystemOptOut: state}
	return f.write(m.log, m.getFile())
}

// getFile returns the path to the system configuration file.
func (m Manager) getFile() string {
	return filepath.Join(m.path, constants.SystemConfigFileName)
}

// write writes the Config to the given path atomically, replacing it if it already exists.
// The directory is created with permissions 0755 if it does not exist.
// The file itself is created with permissions 0644, so that all users can read the system
// configuration; only privileged users will be able to write it in practice.
// Not atomic on Windows.
func (f Config) write(l *slog.Logger, path string) (err error) {
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(f); err != nil {
		return fmt.Errorf("could not encode system config file: %v", err)
	}

	if err := fileutils.AtomicWriteWithPerm(path, buf.Bytes(), 0755, 0644); err != nil {
		return err
	}
	l.Debug("Wrote system config file", "file", path, "system_opt_out", f.SystemOptOut)

	return nil
}
