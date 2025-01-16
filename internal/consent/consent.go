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

// States is a struct that represents the consent states for a list of sources and the global consent state.
type States struct {
	SourceStates map[string]consentStateResult
	GlobalState  consentStateResult
}

type consentStateResult struct {
	Source  string // the source for the consent state
	State   bool   // the consent state for the source
	ReadErr bool   // true if there was an error reading the consent file
}

// consentFile is a struct that represents a consent file.
type consentFile struct {
	ConsentState bool `toml:"consent_state"`
}

// New returns a new ConsentManager.
// path is the folder the consents are stored into.
func New(path string) *Manager {
	return &Manager{path: path}
}

// GetConsentStates gets the consent state for the given sources and the global consent state.
// If any of the sources do not have a consent file, it will be considered as a false state.
// If a specified source does not have a consent file, it will not be included in the returned ConsentStates struct.
// TODO: Simplify to single source, update spec
func (cm Manager) GetConsentStates(sources []string) (consentStates States, err error) {
	consentStates = States{SourceStates: make(map[string]consentStateResult)}

	sourceFiles, globalFile, err := getMatchingConsentFiles(sources, cm.path)
	if err != nil {
		slog.Error("Error getting consent files", "error", err)
		return States{}, err
	}

	results := make(chan consentStateResult, len(sourceFiles))
	defer close(results)

	// Global consent file
	if globalFile != "" {
		globalResult := make(chan consentStateResult, 1)
		defer close(globalResult)

		go func() {
			globalConsent, err := readConsentFile(globalFile)
			if err != nil {
				slog.Error("Error reading global consent file", "file", globalFile, "error", err)
				globalResult <- consentStateResult{Source: "global", State: false, ReadErr: true}
				return
			}
			globalResult <- consentStateResult{Source: "global", State: globalConsent.ConsentState, ReadErr: false}
		}()

		consentStates.GlobalState = <-globalResult
	}

	// Goroutine to read the consent files for each source, excluding the global consent file.
	for source, filePath := range sourceFiles {
		go func(source, filePath string) {
			consent, err := readConsentFile(filePath)
			if err != nil {
				slog.Error("Error reading consent file", "source", source, "error", err)
				results <- consentStateResult{Source: source, State: false, ReadErr: true}
				return
			}
			results <- consentStateResult{Source: source, State: consent.ConsentState, ReadErr: false}
		}(source, filePath)
	}

	for range sourceFiles {
		res := <-results
		consentStates.SourceStates[res.Source] = res
	}

	return consentStates, nil
}

var consentSourceFilePattern = `%s` + constants.ConsentSourceBaseSeparator + constants.GlobalFileName

// SetConsentState updates the consent state for the given source.
// If the source is an empty string, then the global consent state will be set.
// If the target consent file does not exist, it will be created.
func (cm Manager) SetConsentState(source string, state bool) (err error) {
	defer decorate.OnError(&err, "could not set consent state")

	p := filepath.Join(cm.path, constants.GlobalFileName)
	if source != "" {
		p = filepath.Join(cm.path, fmt.Sprintf(consentSourceFilePattern, source))
	}

	consent := consentFile{ConsentState: state}
	return consent.write(p)
}

// getMatchingConsentFiles returns a map of all paths to consent files matching the given sources and a path
// to the global consent file.
// If sources is empty, all consent files in the folder will be returned.
// If a source does not have a consent file, it will be represented as an empty string
// Does not traverse subdirectories.
func getMatchingConsentFiles(sources []string, folderPath string) (sourceFiles map[string]string, globalFile string, err error) {
	sourceFiles = make(map[string]string)

	entries, err := os.ReadDir(folderPath)
	if err != nil {
		return sourceFiles, globalFile, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Global file
		if entry.Name() == constants.GlobalFileName {
			globalFile = filepath.Join(folderPath, entry.Name())
			slog.Debug("Found global consent file", "file", globalFile)
			continue
		}

		if len(sources) == 0 {
			// Source file
			if !strings.HasSuffix(entry.Name(), constants.ConsentSourceBaseSeparator+constants.GlobalFileName) {
				continue
			}
			source := strings.TrimSuffix(entry.Name(), constants.ConsentSourceBaseSeparator+constants.GlobalFileName)
			sourceFiles[source] = filepath.Join(folderPath, entry.Name())
			slog.Debug("Found source consent file", "file", sourceFiles[source])
			continue
		}

		for _, source := range sources {
			if entry.Name() == fmt.Sprintf(consentSourceFilePattern, source) {
				sourceFiles[source] = filepath.Join(folderPath, entry.Name())
				slog.Debug("Found matching source consent file", "file", sourceFiles[source])
				break
			}
		}
	}
	return sourceFiles, globalFile, err
}

func readConsentFile(path string) (consentFile, error) {
	var consent consentFile
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &consent, nil
	}
	_, err := toml.DecodeFile(path, &consent)
	slog.Debug("Read consent file", "file", path, "consent", consent.ConsentState)

	return consent, err
}

// writeConsentFile writes the given consent file to the given path atomically, replacing it if it already exists.
// Not atomic on Windows.
func (cf consentFile) write(path string) (err error) {
	tmp, err := os.CreateTemp(filepath.Dir(path), "consent-*.tmp")
	if err != nil {
		return fmt.Errorf("could not create temporary file: %v", err)
	}
	defer func() {
		_ = tmp.Close()
		if err := os.Remove(tmp.Name()); err != nil {
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
