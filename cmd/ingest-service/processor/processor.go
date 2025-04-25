// Package processor provides the functionality to process JSON files.
// It includes functions to validate, read, and process files, as well as upload data to a PostgreSQL database.
package processor

import (
	"fmt"

	"github.com/ubuntu/ubuntu-insights/cmd/ingest-service/config"
)

func getJSONFiles(dir string) ([]string, error) {
	return nil, nil
}

func processSingleFile(file string) error {
	return nil
}

// ProcessFiles processes all JSON files in the specified directory.
// It reads each file, unmarshals the JSON data into a FileData struct,
// and uploads the data to a PostgreSQL database.
// After processing, it removes the file from the filesystem.
func ProcessFiles(cfg *config.ServiceConfig) error {
	files, err := getJSONFiles(cfg.InputDir)
	if err != nil {
		return fmt.Errorf("failed to get JSON files: %w", err)
	}

	for _, file := range files {
		processSingleFile(file)
	}

	return nil
}
