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
	"github.com/ubuntu/ubuntu-insights/internal/constants"
)

// Manager is a struct that manages consent files.
type Manager struct {
	folderPath string
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
func New(folderPath string) *Manager {
	return &Manager{folderPath: folderPath}
}

// GetConsentStates gets the consent state for the given sources and the global consent state.
// If any of the sources do not have a consent file, it will be considered as a false state.
// If a specified source does not have a consent file, it will not be included in the returned ConsentStates struct.
func (cm *Manager) GetConsentStates(sources []string) (*States, error) {
	consentStates := States{SourceStates: make(map[string]consentStateResult)}

	sourceFiles, globalFile, err := getMatchingConsentFiles(sources, cm.folderPath)
	if err != nil {
		slog.Error("Error getting consent files", "error", err)
		return nil, err
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

	return &consentStates, nil
}

// SetConsentState sets the consent state for the given source.
// If the source is an empty string, then the global consent state will be set.
// If the target consent file does not exist, it will be created.
func (cm *Manager) SetConsentState(source string, state bool) error {
	var filePath string
	if source == "" {
		filePath = filepath.Join(cm.folderPath, constants.BaseConsentFileName)
	} else {
		filePath = filepath.Join(cm.folderPath, source+constants.ConsentSourceBaseSeparator+constants.BaseConsentFileName)
	}

	consent := consentFile{ConsentState: state}
	return writeConsentFile(filePath, &consent)
}

// getMatchingConsentFiles returns a map of all paths to consent files matching the given sources and a path to the global consent file.
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
		if entry.Name() == constants.BaseConsentFileName {
			globalFile = filepath.Join(folderPath, entry.Name())
			slog.Debug("Found global consent file", "file", globalFile)
			continue
		}

		if len(sources) == 0 {
			// Source file
			if !strings.HasSuffix(entry.Name(), constants.ConsentSourceBaseSeparator+constants.BaseConsentFileName) {
				continue
			}
			source := strings.TrimSuffix(entry.Name(), constants.ConsentSourceBaseSeparator+constants.BaseConsentFileName)
			sourceFiles[source] = filepath.Join(folderPath, entry.Name())
			slog.Debug("Found source consent file", "file", sourceFiles[source])
			continue
		}

		for _, source := range sources {
			if entry.Name() == source+constants.ConsentSourceBaseSeparator+constants.BaseConsentFileName {
				sourceFiles[source] = filepath.Join(folderPath, entry.Name())
				slog.Debug("Found matching source consent file", "file", sourceFiles[source])
				break
			}
		}
	}
	return sourceFiles, globalFile, err
}

func readConsentFile(filePath string) (*consentFile, error) {
	var consent consentFile
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return &consent, nil
	}
	_, err := toml.DecodeFile(filePath, &consent)
	slog.Debug("Read consent file", "file", filePath, "consent", consent.ConsentState)

	return &consent, err
}

// writeConsentFile writes the given consent file to the given path atomically, replacing it if it already exists.
// Not atomic in Windows.
func writeConsentFile(filePath string, consent *consentFile) (err error) {
	dir := filepath.Dir(filePath)
	tempFile, err := os.CreateTemp(dir, "consent-*.tmp")
	if err != nil {
		return fmt.Errorf("could not create temporary file: %w", err)
	}
	defer func() {
		tempFile.Close()
		os.Remove(tempFile.Name())
	}()

	if err := toml.NewEncoder(tempFile).Encode(consent); err != nil {
		return fmt.Errorf("could not encode consent file: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("could not close temporary file: %w", err)
	}

	if err := os.Rename(tempFile.Name(), filePath); err != nil {
		return fmt.Errorf("could not rename temporary file: %w", err)
	}
	slog.Debug("Wrote consent file", "file", filePath, "consent", consent.ConsentState)

	return nil
}
