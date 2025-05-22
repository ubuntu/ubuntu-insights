// Package processor provides the functionality to process JSON files.
// It includes functions to validate, read, and process files, as well as upload data to a PostgreSQL database.
package processor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"regexp"

	"github.com/go-viper/mapstructure/v2"
	"github.com/ubuntu/ubuntu-insights/internal/server/ingest/models"
)

var debianVersionRegex = regexp.MustCompile(`^(?:(\d+):)?([a-zA-Z0-9.+~-]+)(?:-([a-zA-Z0-9.+~]+))?$`)

var (
	errInvalidJSON = errors.New("json file is invalid and could not be parsed")
	errNoValidData = errors.New("report file has no valid data")
)

type database interface {
	Upload(ctx context.Context, app string, data *models.TargetModel) error
}

type validateFileError struct {
	Data *models.TargetModel
	Err  error
}

func (e *validateFileError) Error() string                 { return e.Err.Error() }
func (e *validateFileError) Unwrap() error                 { return e.Err }
func (e *validateFileError) FileData() *models.TargetModel { return e.Data }

func validateFile(data *models.TargetModel, path string) error {
	if data.OptOut {
		// Ensure everything else is empty
		if !reflect.DeepEqual(data, &models.TargetModel{OptOut: true}) {
			return fmt.Errorf("opt-out file %q contains unexpected data", path)
		}
		return nil
	}

	// Check if everything we expect (other than extras) is empty
	if data.InsightsVersion == "" &&
		data.CollectionTime == 0 &&
		reflect.DeepEqual(data.SystemInfo, models.TargetSystemInfo{}) &&
		data.SourceMetrics == nil {
		return errNoValidData
	}

	// Check version
	if data.InsightsVersion == "" {
		return fmt.Errorf("missing InsightsVersion in file %q", path)
	}

	if !debianVersionRegex.MatchString(data.InsightsVersion) {
		return fmt.Errorf("invalid version format %q in file %q", data.InsightsVersion, path)
	}

	// Check no extra data
	if data.Extras != nil {
		return fmt.Errorf("unexpected Extras field in file %q", path)
	}

	if data.SystemInfo.Extras != nil {
		return fmt.Errorf("unexpected SystemInfo.Extras field in file %q", path)
	}

	return nil
}

func getJSONFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.Type().IsRegular() && filepath.Ext(path) == ".json" {
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

	var jsonData map[string]any
	if err = json.Unmarshal(data, &jsonData); err != nil {
		return nil, errors.Join(errInvalidJSON, err)
	}

	config := &mapstructure.DecoderConfig{
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			// This hook converts any map[string]interface{} or []interface{} to json.RawMessage
			func(from reflect.Type, to reflect.Type, data any) (any, error) {
				if to != reflect.TypeOf(json.RawMessage{}) {
					return data, nil
				}

				// Marshal the data back to JSON bytes
				jsonBytes, err := json.Marshal(data)
				if err != nil {
					return nil, err
				}

				return json.RawMessage(jsonBytes), nil
			},
		),
		WeaklyTypedInput: true,
		Result:           &models.TargetModel{},
	}

	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return nil, errors.Join(errInvalidJSON, err)
	}

	if err = decoder.Decode(jsonData); err != nil {
		return nil, errors.Join(errInvalidJSON, err)
	}

	// Replace the unchecked type assertion with a checked one
	fileData, ok := config.Result.(*models.TargetModel)
	if !ok {
		return nil, errors.Join(errInvalidJSON, errors.New("failed to convert result to TargetModel"))
	}

	if err := validateFile(fileData, file); err != nil {
		return nil, &validateFileError{Data: fileData, Err: err}
	}

	return fileData, nil
}

// postProcess is a helper function to handle post-processing of processed files.
//
// Files which are successfully processed and uploaded to the database without any validtion errors
// are removed from the filesystem.
// Files which are invalid or have validation errors, even if they were partially added to the database,
// are moved to a separate invalid directory for further inspection.
//
// The function expects invalidDir to be a valid directory path where invalid files will be moved.
func postProcess(file string, err error, invalidDir string) {
	if err == nil {
		if err := os.Remove(file); err != nil {
			slog.Warn("Failed to remove file after processing", "file", file, "err", err)
		}
		return
	}

	newPath := filepath.Join(invalidDir, filepath.Base(file))
	if err := os.Rename(file, newPath); err != nil {
		slog.Warn("Failed to move invalid file", "file", file, "newPath", newPath, "err", err)

		if err := os.Remove(file); err != nil {
			slog.Warn("Failed remove unmovable invalid file", "file", file, "err", err)
		}
		return
	}
	slog.Debug("Moved invalid file to invalid directory", "file", file, "newPath", newPath)
}

// ProcessFiles processes all JSON files in the specified directory.
// It reads each file, unmarshals the JSON data into a FileData struct,
// and uploads the data to a PostgreSQL database.
// After processing, it removes the file from the filesystem.
//
// It returns an error if a catastrophic failure occurs, excluding database errors.
func ProcessFiles(ctx context.Context, dir string, db database, invalidDir string) error {
	app := filepath.Base(dir)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create directory %q: %v", dir, err)
	}

	if err := os.MkdirAll(invalidDir, 0750); err != nil {
		return fmt.Errorf("failed to create invalid directory %q: %v", invalidDir, err)
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
		var vfe *validateFileError
		switch {
		case errors.Is(err, errNoValidData):
			slog.Warn("File has no valid data", "file", file, "err", err)
		case errors.As(err, &vfe):
			slog.Warn("Failed to fully process file", "file", file, "err", err)
			fileData = vfe.FileData()
			fallthrough // Continue with upload for data we were able to process
		case err == nil:
			if uErr := db.Upload(ctx, app, fileData); uErr != nil {
				if errors.Is(uErr, context.Canceled) {
					return err // normal shutdown
				}
				slog.Warn("Failed to upload file to PostgreSQL", "file", file, "err", uErr)
				continue // Skip file removal if upload fails
			}
			slog.Info("Successfully processed and uploaded file", "file", file)
		case errors.Is(err, errInvalidJSON):
			fallthrough
		default:
			slog.Warn("Failed to process file", "file", file, "err", err)
		}

		postProcess(file, err, invalidDir)
		slog.Info("Finished processing file", "file", file)
	}

	return nil
}
