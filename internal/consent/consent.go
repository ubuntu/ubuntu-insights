// Package consent is the implementation of the consent manager component.
// The consent manager is responsible for managing consent files, which are used to store the consent state for a source or the global consent state.
package consent

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/ubuntu/decorate"
	"github.com/ubuntu/ubuntu-insights/internal/constants"
)

// Manager is a struct that manages consent files.
type Manager struct {
	path string
}

// CFile is a struct that represents a consent file.
type CFile struct {
	ConsentState bool `toml:"consent_state"`
}

// New returns a new ConsentManager.
// path is the folder the consents are stored into.
func New(path string) *Manager {
	return &Manager{path: path}
}

// GetState gets the consent state for the given source.
// If the source do not have a consent file, it will be considered as a false state.
// If the source is an empty string, then the global consent state will be returned.
// If the target consent file does not exist, it will not be created.
func (cm Manager) GetState(source string) (bool, error) {
	sourceConsent, err := readFile(cm.getFile(source))
	if err != nil {
		return false, err
	}

	return sourceConsent.ConsentState, nil
}

var consentSourceFilePattern = `%s` + constants.ConsentSourceBaseSeparator + constants.GlobalFileName

// SetState updates the consent state for the given source.
// If the source is an empty string, then the global consent state will be set.
// If the target consent file does not exist, it will be created.
func (cm Manager) SetState(source string, state bool) (err error) {
	defer decorate.OnError(&err, "could not set consent state")

	consent := CFile{ConsentState: state}
	return consent.write(cm.getFile(source))
}

// getFile returns the expected path to the consent file for the given source.
// If source is blank, it returns the path to the global consent file.
// It does not check if the file exists, or if it is valid.
func (cm Manager) getFile(source string) string {
	p := filepath.Join(cm.path, constants.GlobalFileName)
	if source != "" {
		p = filepath.Join(cm.path, fmt.Sprintf(consentSourceFilePattern, source))
	}

	return p
}

// getSourceConsentFiles returns a map of all paths to validly named consent files in the folder, other than the global file.
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
		if !strings.HasSuffix(entry.Name(), constants.ConsentSourceBaseSeparator+constants.GlobalFileName) {
			continue
		}
		source := strings.TrimSuffix(entry.Name(), constants.ConsentSourceBaseSeparator+constants.GlobalFileName)
		sourceFiles[source] = filepath.Join(cm.path, entry.Name())
		slog.Debug("Found source consent file", "file", sourceFiles[source])
	}

	return sourceFiles, nil
}

func readFile(path string) (CFile, error) {
	var consent CFile
	_, err := toml.DecodeFile(path, &consent)
	slog.Debug("Read consent file", "file", path, "consent", consent.ConsentState)

	return consent, err
}

// writeConsentFile writes the given consent file to the given path atomically, replacing it if it already exists.
// Not atomic on Windows.
// Makes dir if it does not exist.
func (cf CFile) write(path string) (err error) {
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return fmt.Errorf("could not create directory: %v", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "consent-*.tmp")
	if err != nil {
		return fmt.Errorf("could not create temporary file: %v", err)
	}
	defer func() {
		_ = tmp.Close()
		if err := os.Remove(tmp.Name()); err != nil && !os.IsNotExist(err) {
			slog.Warn("Failed to remove temporary file when writing consent file", "file", tmp.Name(), "error", err)
		}
	}()

	if err := toml.NewEncoder(tmp).Encode(cf); err != nil {
		return fmt.Errorf("could not encode consent file: %v", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("could not close temporary file: %v", err)
	}

	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("could not rename temporary file: %v", err)
	}
	slog.Debug("Wrote consent file", "file", path, "consent", cf.ConsentState)

	return nil
}
