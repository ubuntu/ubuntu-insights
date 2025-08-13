// Package report provides utility functions for handling reports.
package report

// package report

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/ubuntu/ubuntu-insights/common/fileutils"
	"github.com/ubuntu/ubuntu-insights/insights/internal/constants"
)

var (
	// ErrInvalidReportExt is returned when a report file has an invalid extension.
	ErrInvalidReportExt = errors.New("invalid report file extension")

	// ErrInvalidReportName is returned when a report file has an invalid name that can't be parsed.
	ErrInvalidReportName = errors.New("invalid report file name")
)

// Report represents a report file.
type Report struct {
	Path      string // Path is the path to the report file.
	Name      string // Name is the name of the report file, including extension.
	TimeStamp int64  // TimeStamp is the timestamp of the report.

	reportStash reportStash
}

// reportStash is a helper struct to store a report and its data for movement.
type reportStash struct {
	Path string
	Data []byte
}

// New creates a new Report object from a path.
// It does not write to the file system, or validate the path.
func New(path string) (Report, error) {
	if filepath.Ext(path) != constants.ReportExt {
		return Report{}, ErrInvalidReportExt
	}

	rTime, err := getReportTime(filepath.Base(path))
	if err != nil {
		return Report{}, err
	}

	return Report{Path: path, Name: filepath.Base(path), TimeStamp: rTime}, nil
}

// ReadJSON reads the JSON data from the report file.
func (r Report) ReadJSON() ([]byte, error) {
	// Read the report file
	data, err := os.ReadFile(r.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read report file: %v", err)
	}

	if !json.Valid(data) {
		return nil, fmt.Errorf("invalid JSON data in report file")
	}

	return data, nil
}

// MarkAsProcessed moves the report to a destination directory, and writes the data to the report.
// The original report is removed.
//
// The new report is returned, and the original data is stashed for use with UndoProcessed.
// Note that calling MarkAsProcessed multiple times on the same report will overwrite the stashed data.
func (r Report) MarkAsProcessed(dest string, data []byte) (Report, error) {
	origData, err := r.ReadJSON()
	if err != nil {
		return Report{}, fmt.Errorf("failed to read original report: %v", err)
	}

	newReport := Report{Path: filepath.Join(dest, r.Name), Name: r.Name, TimeStamp: r.TimeStamp,
		reportStash: reportStash{Path: r.Path, Data: origData}}

	if err := fileutils.AtomicWrite(newReport.Path, data); err != nil {
		return Report{}, fmt.Errorf("failed to write report: %v", err)
	}

	if err := os.Remove(r.Path); err != nil {
		return Report{}, fmt.Errorf("failed to remove report: %v", err)
	}

	return newReport, nil
}

// UndoProcessed moves the report back to the original directory, and writes the original data to the report.
// The new report is returned, and the original data is removed.
func (r Report) UndoProcessed() (Report, error) {
	if r.reportStash.Path == "" {
		return Report{}, errors.New("no stashed data to restore")
	}

	if err := fileutils.AtomicWrite(r.reportStash.Path, r.reportStash.Data); err != nil {
		return Report{}, fmt.Errorf("failed to write report: %v", err)
	}

	if err := os.Remove(r.Path); err != nil {
		return Report{}, fmt.Errorf("failed to remove report: %v", err)
	}

	newReport := Report{Path: r.reportStash.Path, Name: r.Name, TimeStamp: r.TimeStamp}
	return newReport, nil
}

// getReportTime returns a int64 representation of the report time from the report path.
func getReportTime(path string) (int64, error) {
	fileName := filepath.Base(path)
	i, err := strconv.ParseInt(strings.TrimSuffix(fileName, filepath.Ext(fileName)), 10, 64)
	if err != nil {
		return i, fmt.Errorf("%w: %v", ErrInvalidReportName, err)
	}
	return i, nil
}

// GetPeriodStart returns the start of the period window for a given period in seconds.
// If period is 0, it returns the current time as a Unix timestamp.
// In cases of underflow, it returns the minimum int64 value.
func GetPeriodStart(t int64, period uint32) int64 {
	if period == 0 {
		return t // If period is 0, return the current time
	}

	if t < math.MinInt64+int64(period) {
		return math.MinInt64 // Pin to minimum int64 in case of underflow
	}

	return t - int64(period)
}

// DuplicateExists checks if there are any reports within the given period window.
func DuplicateExists(l *slog.Logger, dir string, t int64, period uint32) (bool, error) {
	periodStart := GetPeriodStart(t, period)

	reports, err := walkReports(l, dir, periodStart, t, func(reports []Report) bool {
		return len(reports) > 0
	})
	if err != nil {
		return false, err
	}

	return len(reports) > 0, nil
}

// GetAll returns all reports in a given directory.
// Reports are expected to have the correct file extension, and have a name which can be parsed by a timestamp.
// Does not traverse subdirectories.
func GetAll(l *slog.Logger, dir string) ([]Report, error) {
	return walkReports(l, dir, math.MinInt64, math.MaxInt64, func(reports []Report) bool {
		return false // No early exit condition, we want all reports.
	})
}

// ClearPeriod removes all reports in a given dir, within the period window [t-period, t].
// If a file failed to be removed, an error is logged but the function continues.
func ClearPeriod(l *slog.Logger, dir string, t int64, period uint32) error {
	periodStart := GetPeriodStart(t, period)
	maxReports := int64(period) + 1
	reports, err := walkReports(l, dir, periodStart, t, func(reports []Report) bool {
		// Early exit condition: stop if we have enough reports.
		return int64(len(reports)) >= maxReports
	})
	if err != nil {
		return err
	}

	for _, r := range reports {
		if err := os.Remove(r.Path); err != nil {
			l.Error("failed to remove report", "path", r.Path, "error", err)
		}
	}

	return nil
}

// Cleanup removes reports in a directory, keeping the most recent maxReports reports.
// If a file does not appear to be a report, it is skipped and ignored.
// If a file is unable to be removed, then it will be logged, but the function will continue.
func Cleanup(l *slog.Logger, dir string, maxReports uint32) error {
	if maxReports > math.MaxInt32 {
		return fmt.Errorf("maxReports is too large and would result in an overflow: %d", maxReports)
	}

	mReports := int32(maxReports)

	reports, err := GetAll(l, dir)
	if err != nil {
		return err
	}

	if len(reports) <= int(mReports) {
		l.Debug("no reports to cleanup", "maxReports", mReports, "numReports", len(reports))
		return nil
	}

	// Sort the reports by timestamp, and keep the most recent maxReports.
	slices.SortStableFunc(reports, func(i, j Report) int {
		return int(i.TimeStamp - j.TimeStamp)
	})

	// Remove the oldest reports, keeping the most recent maxReports.
	for _, report := range reports[:len(reports)-int(mReports)] {
		if err := os.Remove(report.Path); err != nil {
			l.Error("failed to remove report", "path", report.Path, "error", err)
		}
	}

	return nil
}

// walkReports walks the report directory and collects reports within the given time period.
// When a new report is found and appended, earlyExitFn is called to determine if we should stop walking.
func walkReports(l *slog.Logger, dir string, periodStart, periodEnd int64, earlyExitFn func([]Report) bool) ([]Report, error) {
	var reports []Report
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("failed to access path: %v", err)
		}

		if d.IsDir() {
			if path != dir {
				return filepath.SkipDir // Skip subdirectories.
			}
			return nil // Continue walking the directory.
		}

		r, err := New(path)
		if err != nil {
			l.Debug("Skipping non-report file", "file", d.Name(), "error", err)
			return nil
		}

		if r.TimeStamp < periodStart || r.TimeStamp > periodEnd {
			return nil
		}

		reports = append(reports, r)
		if earlyExitFn(reports) {
			return filepath.SkipAll // Stop walking early if early exit condition is met.
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return reports, nil
}
