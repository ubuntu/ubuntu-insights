// Package consent is the implementation of the consent manager component.
// The consent manager is responsible for managing consent files, which are used to store the consent state for a source.
package consent

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/ubuntu/decorate"
	"github.com/ubuntu/ubuntu-insights/common/fileutils"
	"github.com/ubuntu/ubuntu-insights/insights/internal/constants"
	"github.com/ubuntu/ubuntu-insights/insights/internal/systemconfig"
)

var (
	// ErrConsentFileNotFound is returned when a consent file is not found.
	ErrConsentFileNotFound = fmt.Errorf("consent file not found")
)

// Manager is a struct that manages consent files.
type Manager struct {
	path string

	// optOut, when set, makes GetState honor the system-wide opt-out state.
	// When nil, only the per-source consent files are consulted.
	optOut *systemconfig.Manager

	log *slog.Logger
}

// CFile is a struct that represents a consent file.
type CFile struct {
	ConsentState bool `toml:"consent_state"`
}

// New returns a new ConsentManager.
// path is the folder the consents are stored into.
func New(l *slog.Logger, path string) *Manager {
	return &Manager{log: l, path: path}
}

// NewWithSystemConfig returns a new Manager that also honors the system-wide opt-out state.
// consentDir is the folder the per-source consent files are stored into, and systemConfigDir
// is the directory containing the system-wide configuration file.
//
// When the system opt-out is active, GetState reports false for every source regardless of the
// per-source consent files. SetState is unaffected.
func NewWithSystemConfig(l *slog.Logger, consentDir, systemConfigDir string) *Manager {
	return &Manager{
		log:    l,
		path:   consentDir,
		optOut: systemconfig.New(l, systemConfigDir),
	}
}

// GetState gets the consent state for the given source.
// If the source do not have a consent file, it will be considered as a false state.
// If the source is an empty string, then the platform source consent state will be returned.
// If the target consent file does not exist, it will not be created.
// If the target consent file does not exist, then ErrConsentFileNotFound will be returned.
func (cm Manager) GetState(source string) (state bool, err error) {
	defer func() {
		// If the err is a path error and the error is not found, return proper error
		var pe *os.PathError
		if errors.As(err, &pe) && errors.Is(pe.Err, os.ErrNotExist) {
			err = errors.Join(ErrConsentFileNotFound, err)
		}
	}()

	if cm.optOut != nil {
		optedOut, optErr := cm.optOut.IsOptedOut()
		if optErr != nil {
			return false, fmt.Errorf("failed to check system opt-out: %v", optErr)
		}
		if optedOut {
			cm.log.Info("System opt-out is active, treating consent as false", "source", source)
			return false, nil
		}
	}

	sourceConsent, err := readFile(cm.log, cm.getFile(source))
	if err != nil {
		return false, err
	}

	return sourceConsent.ConsentState, nil
}

var consentSourceFilePattern = `%s` + constants.ConsentFilenameSuffix

// SetState updates the consent state for the given source.
// If the source is an empty string, then the platform source consent state will be set.
// If the target consent file does not exist, it will be created.
func (cm Manager) SetState(source string, state bool) (err error) {
	defer decorate.OnError(&err, "could not set consent state")

	consent := CFile{ConsentState: state}
	return consent.write(cm.log, cm.getFile(source))
}

// getFile returns the expected path to the consent file for the given source.
// If source is blank, it returns the path to the platform source consent file.
// It does not check if the file exists, or if it is valid.
func (cm Manager) getFile(source string) string {
	if source == "" {
		source = constants.PlatformSource
	}
	return filepath.Join(cm.path, fmt.Sprintf(consentSourceFilePattern, source))
}

// getSourceConsentFiles returns a map of all paths to validly named consent files in the folder.
func (cm Manager) getFiles() (map[string]string, error) {
	sourceFiles := make(map[string]string)

	entries, err := os.ReadDir(cm.path)
	if err != nil {
		return sourceFiles, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Source file
		if !strings.HasSuffix(entry.Name(), constants.ConsentFilenameSuffix) {
			continue
		}
		source := strings.TrimSuffix(entry.Name(), constants.ConsentFilenameSuffix)
		sourceFiles[source] = filepath.Join(cm.path, entry.Name())
		cm.log.Debug("Found source consent file", "file", sourceFiles[source])
	}

	return sourceFiles, nil
}

func readFile(l *slog.Logger, path string) (CFile, error) {
	var consent CFile
	_, err := toml.DecodeFile(path, &consent)
	l.Debug("Read consent file", "file", path, "consent", consent.ConsentState)

	return consent, err
}

// writeConsentFile writes the given consent file to the given path atomically, replacing it if it already exists.
// Not atomic on Windows.
// Makes dir if it does not exist.
func (cf CFile) write(l *slog.Logger, path string) (err error) {
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(cf); err != nil {
		return fmt.Errorf("could not encode consent file: %v", err)
	}

	if err := fileutils.AtomicWriteWithPerm(path, buf.Bytes(), 0750, 0600); err != nil {
		return err
	}
	l.Debug("Wrote consent file", "file", path, "consent", cf.ConsentState)

	return nil
}
