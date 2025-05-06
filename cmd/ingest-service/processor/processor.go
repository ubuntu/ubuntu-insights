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
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/ubuntu/ubuntu-insights/cmd/ingest-service/config"
	"github.com/ubuntu/ubuntu-insights/cmd/ingest-service/models"
	"github.com/ubuntu/ubuntu-insights/cmd/ingest-service/storage"
)

var semverRegex = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
var workerCount = runtime.NumCPU()

func validateGeneratedTime(generated string) error {
	unixSec, err := strconv.ParseInt(generated, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid unix timestamp: %w", err)
	}

	parsedTime := time.Unix(unixSec, 0).UTC()
	now := time.Now().UTC()
	if parsedTime.After(now) {
		return fmt.Errorf("timestamp is in the future")
	}

	inceptionDate := time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC)
	if parsedTime.Before(inceptionDate) {
		return fmt.Errorf("timestamp %q is before inception of ubuntu-insights v2 %q", parsedTime.Format(time.RFC3339), inceptionDate.Format(time.RFC3339))
	}

	return nil
}

func validateFile(data *models.RawFileData, path string) error {
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

func transformFileData(data *models.RawFileData) (*models.DBFileData, error) {
	var dbFileData models.DBFileData
	dbFileData.AppID = data.AppID

	// Convert unix timestamp to time.Time
	unixSec, err := strconv.ParseInt(data.Generated, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid unix timestamp: %w", err)
	}
	dbFileData.Generated = time.Unix(unixSec, 0).UTC()

	dbFileData.SchemaVersion = data.SchemaVersion
	dbFileData.Common = data.Common
	dbFileData.AppData = data.AppData

	return &dbFileData, nil
}

func processFile(file string) (*models.DBFileData, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var fileData models.RawFileData
	if err = json.Unmarshal(data, &fileData); err != nil {
		return nil, err
	}

	if err = validateFile(&fileData, file); err != nil {
		return nil, err
	}

	var dbFileData *models.DBFileData
	dbFileData, err = transformFileData(&fileData)

	if err != nil {
		return nil, err
	}

	return dbFileData, nil
}

// ProcessFiles processes all JSON files in the specified directory.
// It reads each file, unmarshals the JSON data into a FileData struct,
// and uploads the data to a PostgreSQL database.
// After processing, it removes the file from the filesystem.
func ProcessFiles(ctx context.Context, cfg *config.ServiceConfig, uploader storage.Uploader) error {
	files, err := getJSONFiles(cfg.InputDir)
	if err != nil {
		return fmt.Errorf("failed to get JSON files: %w", err)
	}

	fileCh := make(chan string)
	var wg sync.WaitGroup
	errCh := make(chan error, workerCount)

	slog.Info("Processing files", "count", len(files))
	slog.Info("Directory", "dir", cfg.InputDir)

	// Start workers
	for range workerCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range fileCh {
				select {
				case <-ctx.Done():
					return // stop immediately if context cancelled
				default:
					// continue
				}

				slog.Info("Processing file", "file", file)

				fileData, err := processFile(file)
				if err == nil {
					if err = uploader.Upload(ctx, storage.Get(), fileData); err == nil {
						slog.Info("Successfully processed and uploaded file", "file", file)
					} else {
						if errors.Is(err, context.Canceled) {
							return // normal shutdown
						}
						slog.Warn("Failed to upload file to PostgreSQL", "file", file, "err", err)
						errCh <- err
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
		}()
	}

	// Send files to workers
	go func() {
		defer close(fileCh)
		for _, file := range files {
			select {
			case <-ctx.Done():
				return // stop sending if context cancelled
			case fileCh <- file:
				// file sent
			}
		}
	}()

	// Wait for workers to finish
	wg.Wait()
	close(errCh)

	// Check if any errors happened
	var finalErr error
	for err := range errCh {
		if finalErr == nil {
			finalErr = err
		} else {
			finalErr = fmt.Errorf("%v; %w", finalErr, err)
		}
	}

	return finalErr
}
