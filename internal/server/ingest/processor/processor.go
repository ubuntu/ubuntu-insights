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
	"reflect"
	"regexp"

	"github.com/ubuntu/ubuntu-insights/internal/server/ingest/models"
)

var debianVersionRegex = regexp.MustCompile(`^(?:(\d+):)?([a-zA-Z0-9.+~-]+)(?:-([a-zA-Z0-9.+~]+))?$`)

type database interface {
	Upload(ctx context.Context, app string, data *models.TargetModel) error
}

func validateFile(data *models.TargetModel, path string) error {
	if data.OptOut {
		// Ensure everything else is empty
		if !reflect.DeepEqual(data, &models.TargetModel{OptOut: true}) {
			return fmt.Errorf("opt-out file %q contains unexpected data", path)
		}
		return nil
	}

	// Check version
	if !debianVersionRegex.MatchString(data.InsightsVersion) {
		return fmt.Errorf("invalid version format %q in file %q", data.InsightsVersion, path)
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

func processFile(file string) (*models.TargetModel, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var fileData models.TargetModel
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
//
// It returns an error if a catastrophic failure occurs, excluding database errors.
func ProcessFiles(ctx context.Context, dir string, db database) error {
	app := filepath.Base(dir)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create directory %q: %v", dir, err)
	}

	files, err := getJSONFiles(dir)
	if err != nil {
		return fmt.Errorf("failed to get JSON files: %v", err)
	}

	for _, file := range files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		fileData, err := processFile(file)
		if err == nil {
			if err = db.Upload(ctx, app, fileData); err == nil {
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
