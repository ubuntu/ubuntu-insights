// Package processor provides the functionality to process JSON files.
// It includes functions to validate, read, and process files, as well as upload data to a PostgreSQL database.
package processor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/ubuntu/ubuntu-insights/internal/server/ingest/config"
	storage "github.com/ubuntu/ubuntu-insights/internal/server/ingest/database"
	"github.com/ubuntu/ubuntu-insights/internal/server/ingest/models"
)

var semverRegex = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

func validateGeneratedTime(generated string) error {
	parsedTime, err := time.Parse(time.RFC3339, generated)
	if err != nil {
		return fmt.Errorf("invalid time format: %w", err)
	}

	now := time.Now()
	if parsedTime.After(now) {
		return fmt.Errorf("timestamp is in the future")
	}

	inceptionDate := time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC)
	if parsedTime.Before(inceptionDate) {
		return fmt.Errorf("timestamp %q is before inception of ubuntu-insights v2 %q", generated, inceptionDate.Format(time.RFC3339))
	}

	return nil
}

func validateFile(data *models.FileData, path string) error {
	// Validate AppID
	if data.AppID == "" {
		return fmt.Errorf("AppID is required")
	}
	parentDir := filepath.Base(filepath.Dir(path))
	if data.AppID != parentDir {
		return fmt.Errorf("AppID %q does not match target app %q", data.AppID, parentDir)
	}

	// Validate Generated timestamp
	if data.Generated == "" {
		return fmt.Errorf("timestamp is required")
	}
	if err := validateGeneratedTime(data.Generated); err != nil {
		return fmt.Errorf("timestamp is invalid: %w", err)
	}

	// Validate SchemaVersion
	if !semverRegex.MatchString(data.SchemaVersion) {
		return fmt.Errorf("invalid schema version %q", data.SchemaVersion)
	}

	if len(data.Common) == 0 {
		return fmt.Errorf("empty payload")
	}
	if len(data.AppData) == 0 {
		return fmt.Errorf("empty payload")
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
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var fileData models.FileData
	if err = json.Unmarshal(data, &fileData); err != nil {
		return nil, err
	}

	if err := validateFile(&fileData, file); err != nil {
		return nil, err
	}

	return &fileData, nil
}

// ProcessFiles processes all JSON files in the specified directory.
// It reads each file, unmarshals the JSON data into a FileData struct,
// and uploads the data to a PostgreSQL database.
// After processing, it removes the file from the filesystem.
func ProcessFiles(ctx context.Context, cfg *config.ServiceConfig) error {
	files, err := getJSONFiles(cfg.InputDir)
	if err != nil {
		return fmt.Errorf("failed to get JSON files: %w", err)
	}

	for _, file := range files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		fileData, err := processFile(file)
		if err == nil {
			if err = storage.UploadToPostgres(ctx, fileData); err == nil {
				slog.Info("Successfully processed and uploaded file", "file", file)
			} else {
				if errors.Is(err, context.Canceled) {
					return err // normal shutdown
				}
				slog.Warn("Failed to upload file to PostgreSQL", "file", file, "err", err)
				continue // Skip file removal if upload fails
			}
		} else {
			slog.Warn("Failed to process file", "file", file, "err", err)
		}

		if err := os.Remove(file); err != nil {
			slog.Warn("Failed to remove file after processing", "file", file, "err", err)
			continue
		}

		slog.Info("Removed file after processing", "file", file)
	}

	return nil
}
