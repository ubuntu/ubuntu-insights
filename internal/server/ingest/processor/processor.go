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

	"github.com/ubuntu/ubuntu-insights/internal/server/ingest/models"
)

var semverRegex = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

type database interface {
	Upload(ctx context.Context, app string, data *models.TargetModel) error
}

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

func validateFile(data *models.TargetModel, path string) error {

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
func ProcessFiles(ctx context.Context, dir string, db database) error {
	app := filepath.Base(dir)
	files, err := getJSONFiles(dir)
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
