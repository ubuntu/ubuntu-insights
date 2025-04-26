// Package processor provides the functionality to process JSON files.
// It includes functions to validate, read, and process files, as well as upload data to a PostgreSQL database.
package processor

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/ubuntu/ubuntu-insights/cmd/ingest-service/config"
	"github.com/ubuntu/ubuntu-insights/cmd/ingest-service/models"
	"github.com/ubuntu/ubuntu-insights/cmd/ingest-service/storage"
)


func validateFile(data *models.FileData, path string) error {
	// Validate AppID
	if data.AppID == "" {
		return fmt.Errorf("AppID is required")
	}
	parentDir := filepath.Base(filepath.Dir(path))
	if data.AppID != parentDir {
		return fmt.Errorf("AppID %q does not match target app %q", data.AppID, parentDir)
	}

	return nil
}

func getJSONFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".json" {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}

func processFile(file string) (*models.FileData, error) {
	defer func() {
		if err := os.Remove(file); err != nil {
			slog.Warn("Failed to remove file after processing", "file", file, "err", err)
		} else {
			slog.Info("Removed file", "file", file)
		}
	}()

	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var fileData models.FileData
	if err = json.Unmarshal(data, &fileData); err != nil {
		return nil, err
	}

	return &fileData, nil
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
		fileData, err := processFile(file)
		if err != nil {
			slog.Warn("Failed to process file", "file", file, "err", err)
			continue
		}

		if err = storage.UploadToPostgres(fileData); err != nil {
			slog.Warn("Failed to upload file to PostgreSQL", "file", file, "err", err)
			continue
		}

		slog.Info("Successfully processed and uploaded file", "file", file)
	}

	return nil
}
