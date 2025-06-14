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
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/google/uuid"
	"github.com/ubuntu/ubuntu-insights/internal/constants"
	"github.com/ubuntu/ubuntu-insights/internal/server/ingest/models"
)

// ErrDatabaseErrors is returned when significant database errors occur during processing.
// It indicates more than a set threshold of report upload attempts have failed due to database issues.
var ErrDatabaseErrors = errors.New("database errors during processing surpassed threshold")

var (
	errNoValidData      = errors.New("report file has no valid data")
	errUnexpectedFields = errors.New("file contains unexpected fields")
	errUploadFailed     = errors.New("failed to upload report to PostgreSQL database")
)

type database interface {
	Upload(ctx context.Context, app string, report *models.TargetModel) error
	UploadLegacy(ctx context.Context, distribution, version string, report *models.LegacyTargetModel) error
	UploadInvalid(ctx context.Context, id, app, rawReport string) error
}

// Processor is responsible for processing reports.
type Processor struct {
	reportsDir string
	db         database
}

// New creates a new Processor instance.
func New(reportsDir string, db database) (*Processor, error) {
	if reportsDir == "" {
		return nil, fmt.Errorf("reportsDir must be set")
	}

	if err := os.MkdirAll(reportsDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create reportsDir: %v", err)
	}

	return &Processor{
		reportsDir: reportsDir,
		db:         db,
	}, nil
}

// Process processes all JSON files in the specified directory, looking within the `reportsDir/app` directory.
// It reads each file, unmarshals the JSON data into a FileData struct,
// and uploads the data to a PostgreSQL database.
// After processing, it removes the file from the filesystem.
//
// It returns an error if a catastrophic failure occurs, or if the number of failed uploads exceeds a threshold.
func (p Processor) Process(ctx context.Context, app string) (err error) {
	const minimumSuccessRate = 0.85

	dir := filepath.Join(p.reportsDir, app)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create directory %q: %v", dir, err)
	}

	files, err := getJSONFiles(dir)
	if err != nil {
		return fmt.Errorf("failed to get JSON files: %v", err)
	}

	var (
		attemptCount = 0
		failureCount = 0
	)
	defer func() {
		// Check if over threshold of uploads failed
		if attemptCount > 0 && float64(failureCount)/float64(attemptCount) > (1-minimumSuccessRate) {
			err = errors.Join(ErrDatabaseErrors, err)
		}
	}()
	legacyApp := isLegacy(app)
	for _, file := range files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		reportID := getReportID(file)

		var procErr error
		if legacyApp {
			distribution, version := parseLegacyApp(app)
			procErr = processAndUpload(
				file,
				validateLegacyReport,
				func(report *models.LegacyTargetModel) error {
					return p.db.UploadLegacy(ctx, distribution, version, report)
				},
			)
		} else {
			procErr = processAndUpload(
				file,
				validateReport,
				func(report *models.TargetModel) error {
					return p.db.Upload(ctx, app, report)
				},
			)
		}

		if procErr == nil || errors.Is(procErr, errUnexpectedFields) || errors.Is(procErr, errUploadFailed) {
			attemptCount++
		}

		if errors.Is(procErr, errUploadFailed) {
			failureCount++
			continue // If upload fails, skip postProcessing
		}

		if procErr != nil {
			uploadAttempted, err := p.uploadInvalid(ctx, file, reportID, app)
			if err != nil {
				slog.Warn("Failed to upload invalid report", "file", file, "err", err)
			}
			if uploadAttempted {
				attemptCount++
				if err != nil {
					failureCount++
				}
			}
		}

		if err := os.Remove(file); err != nil {
			slog.Warn("Failed to remove file after processing", "file", file, "err", err)
		}

		slog.Info("Finished processing file", "file", file)
	}

	return nil
}

// processAndUpload processes a file, validates the report, and uploads it to the database.
//
// If upload fails, it returns errUploadFailed.
// If any error other than errUnexpectedFields or errUploadFailed is returned, upload was not attempted.
func processAndUpload[T models.TargetModels](
	file string,
	validate func(*T) error,
	upload func(*T) error,
) error {
	report, err := processFile[T](file)
	if err != nil {
		slog.Warn("Failed to process file", "file", file, "err", err)
		return err
	}
	validationErr := validate(report)
	switch {
	case errors.Is(validationErr, errUnexpectedFields):
		slog.Warn("Failed to fully process file", "file", file, "err", validationErr)
		fallthrough
	case validationErr == nil:
		if err := upload(report); err != nil {
			slog.Warn("Failed to upload file to PostgreSQL", "file", file, "err", err)
			return errors.Join(errUploadFailed, err)
		}
		slog.Info("Successfully processed and uploaded file", "file", file)
		return validationErr
	default:
		slog.Warn("File processed with errors, skipping upload", "file", file, "err", validationErr)
		return validationErr
	}
}

func validateReport(data *models.TargetModel) (err error) {
	if data.OptOut {
		// Even if other fields are present, treat this as a valid file and discard it fully later.
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

func validateLegacyReport(data *models.LegacyTargetModel) error {
	if data.OptOut {
		// Even if other fields are present, treat this as a valid file and discard it fully later.
		return nil
	}

	// Check if everything is empty
	if data.Fields == nil {
		return errNoValidData
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

// getReportID extracts the report ID from the file path.
// If the file name does not contain a valid UUID, it logs a warning and generates a new UUID.
func getReportID(file string) string {
	reportID := filepath.Base(file)
	reportID = strings.TrimSuffix(reportID, filepath.Ext(reportID))

	if err := uuid.Validate(reportID); err != nil {
		reportID = uuid.NewString()
		slog.Warn("Report has invalid UUID, generating a new one", "file", file, "UUID", reportID, "err", err)
	}

	return reportID
}

// processFile reads a JSON file, unmarshals it into the specified target model type.
// It returns the target model or an error if the file is invalid or does not match the expected structure.
func processFile[T models.TargetModels](file string) (*T, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var jsonData map[string]any
	if err = json.Unmarshal(data, &jsonData); err != nil {
		return nil, errors.Join(errors.New("json file is invalid and could not be parsed"), err)
	}

	report := new(T)
	config := getDecoderConfig(report)
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create decoder: %v", err)
	}

	if err = decoder.Decode(jsonData); err != nil {
		return nil, errors.Join(errors.New("file data does not match expected model structure"), err)
	}

	return report, nil
}

func isLegacy(app string) bool {
	app = filepath.ToSlash(app)
	parts := strings.SplitN(app, "/", 2)
	return parts[0] == constants.LegacyReportTag
}

var legacyPathRE = regexp.MustCompile("^" + regexp.QuoteMeta(constants.LegacyReportTag) + `/([^/]+)/desktop/([^/]+)$`)

// parseLegacyApp parses the legacy app string to extract the distribution and version.
// Returns empty strings if the format is invalid.
func parseLegacyApp(app string) (distribution, version string) {
	app = filepath.ToSlash(app)
	matches := legacyPathRE.FindStringSubmatch(app)
	if len(matches) != 3 {
		return "", ""
	}
	return matches[1], matches[2]
}

// uploadInvalid reads the invalid file and uploads its content to the database as a string.
// It skips empty files or files that contain only whitespace, returning nil in those cases.
//
// If an upload was attempted, even if it failed, it returns true. Otherwise, it returns false.
func (p Processor) uploadInvalid(ctx context.Context, file, id, app string) (bool, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return false, fmt.Errorf("failed to re-read invalid file %q: %v", file, err)
	}

	if len(data) == 0 || strings.TrimSpace(string(data)) == "" {
		slog.Info("Skipping upload of empty invalid file", "file", file)
		return false, nil // Skip empty files
	}

	var jsonFile = make(map[string]any)
	if err := json.Unmarshal(data, &jsonFile); err == nil {
		if len(jsonFile) == 0 {
			slog.Info("Skipping upload of empty JSON file", "file", file)
			return false, nil // Skip empty JSON files
		}
	}

	if err := p.db.UploadInvalid(ctx, id, app, string(data)); err != nil {
		return true, errors.Join(errUploadFailed, err)
	}
	return true, nil
}

func getDecoderConfig(target any) *mapstructure.DecoderConfig {
	return &mapstructure.DecoderConfig{
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
		Result:           target,
	}
}
