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
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/ubuntu/ubuntu-insights/internal/constants"
	"github.com/ubuntu/ubuntu-insights/internal/server/ingest/models"
)

var (
	errInvalidJSON      = errors.New("json file is invalid and could not be parsed")
	errInvalidModel     = errors.New("file data does not match expected model structure")
	errNoValidData      = errors.New("report file has no valid data")
	errUnexpectedFields = errors.New("file contains unexpected fields")
)

type database interface {
	Upload(ctx context.Context, app string, data *models.TargetModel) error
}

// Processor is responsible for processing reports.
type Processor struct {
	baseDir    string
	invalidDir string
	db         database
}

// New creates a new Processor instance.
func New(baseDir, invalidDir string, db database) *Processor {
	return &Processor{
		baseDir:    baseDir,
		invalidDir: invalidDir,
		db:         db,
	}
}

// Process processes all JSON files in the specified directory, looking within the `baseDir/app` directory.
// It reads each file, unmarshals the JSON data into a FileData struct,
// and uploads the data to a PostgreSQL database.
// After processing, it removes the file from the filesystem.
//
// It returns an error if a catastrophic failure occurs, excluding database errors.
func (p Processor) Process(ctx context.Context, app string) error {
	dir := filepath.Join(p.baseDir, app)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create directory %q: %v", dir, err)
	}

	if err := os.MkdirAll(p.invalidDir, 0750); err != nil {
		return fmt.Errorf("failed to create invalid directory %q: %v", p.invalidDir, err)
	}

	files, err := getJSONFiles(dir)
	if err != nil {
		return fmt.Errorf("failed to get JSON files: %v", err)
	}

	isLegacy := isLegacy(app)
	for _, file := range files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if isLegacy {
			// For legacy reports, for now don't attempt to upload the report TODO: database handling for legacy reports
			postProcess(file, errors.Join(fmt.Errorf("legacy report, skipping upload")), p.invalidDir)
			slog.Info("Finished processing legacy file", "file", file)
			continue
		}

		pResult, err := processFile(file)
		if err != nil {
			slog.Warn("Failed to process file", "file", file, "err", err)
			postProcess(file, err, p.invalidDir)
			continue // Skip to the next file if processing fails
		}

		switch {
		case errors.Is(pResult.errors, errUnexpectedFields):
			slog.Warn("Failed to fully process file", "file", file, "err", pResult.errors)
			fallthrough // Continue with upload for data we were able to process
		case pResult.errors == nil:
			if uErr := p.db.Upload(ctx, app, pResult.report); uErr != nil {
				if errors.Is(uErr, context.Canceled) {
					return err // normal shutdown
				}
				slog.Warn("Failed to upload file to PostgreSQL", "file", file, "err", uErr)
				continue // Skip file removal if upload fails
			}
			slog.Info("Successfully processed and uploaded file", "file", file)
		case pResult.errors != nil:
			slog.Warn("File processed with errors, skipping upload", "file", file, "err", pResult.errors)
		}

		postProcess(file, pResult.errors, p.invalidDir)
		slog.Info("Finished processing file", "file", file)
	}

	return nil
}

func validateReport(data *models.TargetModel) (err error) {
	if data.OptOut {
		// Ensure everything else is empty
		if !reflect.DeepEqual(data, &models.TargetModel{OptOut: true}) {
			return errors.Join(errUnexpectedFields, fmt.Errorf("opt-out file contains unexpected data"))
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

	// Check no extra data
	if data.Extras != nil {
		return errors.Join(errUnexpectedFields, fmt.Errorf("unexpected Extras field"))
	}

	if data.SystemInfo.Extras != nil {
		return errors.Join(errUnexpectedFields, fmt.Errorf("unexpected SystemInfo.Extras field"))
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

type processResult struct {
	jsonData map[string]any
	report   *models.TargetModel
	errors   error
}

func processFile(file string) (*processResult, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var jsonData map[string]any
	if err = json.Unmarshal(data, &jsonData); err != nil {
		return &processResult{errors: errors.Join(errInvalidJSON, err)}, nil
	}
	result := &processResult{jsonData: jsonData}

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
		return nil, fmt.Errorf("failed to create decoder: %v", err)
	}

	if err = decoder.Decode(jsonData); err != nil {
		result.errors = errors.Join(errInvalidJSON, err)
		return result, nil
	}

	// Replace the unchecked type assertion with a checked one
	report, ok := config.Result.(*models.TargetModel)
	if !ok {
		result.errors = errInvalidModel
		return result, nil
	}
	result.report = report

	result.errors = validateReport(report)
	return result, nil
}

func isLegacy(app string) bool {
	app = filepath.ToSlash(app)
	parts := strings.SplitN(app, "/", 2)
	return parts[0] == constants.LegacyReportTag
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
