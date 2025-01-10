package consent

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/gofrs/flock"
	"github.com/ubuntu/ubuntu-insights/internal/constants"
)

// ConsentManager is a struct that manages consent files
type ConsentManager struct {
	folderPath string
}

// ConsentStates is a struct that represents the consent states for a list of sources and the global consent state
type ConsentStates struct {
	SourceStates map[string]consentStateResult
	GlobalState  consentStateResult
}

type consentStateResult struct {
	Source  string // the source for the consent state
	State   bool   // the consent state for the source
	ReadErr bool   // true if there was an error reading the consent file
}

// consentFile is a struct that represents a consent file
type consentFile struct {
	ConsentState bool `toml:"consent_state"`
}

// New returns a new ConsentManager
func New(folderPath string) *ConsentManager {
	return &ConsentManager{folderPath: folderPath}
}

// GetConsentStates, gets the consent state for the given sources and the global consent state
// If any of the sources do not have a consent file, it will be considered as a false state.
// If a specified source does not have a consent file, it will not be included in the returned ConsentStates struct
func (cm *ConsentManager) GetConsentStates(sources []string) (*ConsentStates, error) {
	consentStates := ConsentStates{SourceStates: make(map[string]consentStateResult)}

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

	// Goroutine to read the consent files for each source, exlcuding the global consent file
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

// getMatchingConsentFiles returns a map of all paths to consent files matching the given sources and a path to the global consent file.
// If sources is empty, all consent files in the folder will be returned.
// If a source does not have a consent file, it will be represented as an empty string
// Does not traverse subdirectories
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

		if len(sources) == 0 {
			// Global file
			if entry.Name() == constants.BaseConsentFileName {
				globalFile = filepath.Join(folderPath, entry.Name())
				continue
			}
			// Source file
			if !strings.HasSuffix(entry.Name(), constants.ConsentSourceBaseSeparator+constants.BaseConsentFileName) {
				continue
			}
			source := strings.TrimSuffix(entry.Name(), constants.ConsentSourceBaseSeparator+constants.BaseConsentFileName)
			sourceFiles[source] = filepath.Join(folderPath, entry.Name())
			continue
		}

		for _, source := range sources {
			if entry.Name() == source+constants.ConsentSourceBaseSeparator+constants.BaseConsentFileName {
				sourceFiles[source] = filepath.Join(folderPath, entry.Name())
				break
			} else if entry.Name() == constants.BaseConsentFileName {
				globalFile = filepath.Join(folderPath, entry.Name())
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

	lock := flock.New(filePath + ".lock")
	lockAquired, err := lock.TryRLock()
	if err != nil {
		return &consent, err
	}
	if !lockAquired {
		return &consent, fmt.Errorf("could not aquire lock on %s", filePath)
	}
	defer lock.Unlock()

	_, err = toml.DecodeFile(filePath, &consent)

	return &consent, err
}

// writeConsentFile writes the given consent file to the given path, replacing it if it already exists
func writeConsentFile(filePath string, consent *consentFile) error {
	lock := flock.New(filePath + ".lock")
	lockAquired, err := lock.TryLock()
	if err != nil {
		return err
	}
	if !lockAquired {
		return fmt.Errorf("could not aquire lock on %s", filePath)
	}
	defer lock.Unlock()

	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := toml.NewEncoder(file).Encode(consent); err != nil {
		return err
	}

	return nil
}
