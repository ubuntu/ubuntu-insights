// Package report provides utility functions for handling reports.
package report

// package report

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ubuntu/ubuntu-insights/internal/constants"
	"github.com/ubuntu/ubuntu-insights/internal/fileutils"
)

var (
	// ErrInvalidPeriod is returned when a function requiring a period, received an invalid, period that isn't a non-negative integer.
	ErrInvalidPeriod = errors.New("invalid period, period should be a positive integer")

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
func GetPeriodStart(period int, t time.Time) (int64, error) {
	if period <= 0 {
		return 0, ErrInvalidPeriod
	}
	return t.Unix() - (t.Unix() % int64(period)), nil
}

// GetForPeriod returns the most recent report within a period window for a given directory.
// Not inclusive of the period end (periodStart + period).
//
// For example, given reports 1 and 7, with time 2 and period 7, the function will return the path for report 1.
func GetForPeriod(dir string, t time.Time, period int) (Report, error) {
	periodStart, err := GetPeriodStart(period, t)
	if err != nil {
		return Report{}, err
	}
	periodEnd := periodStart + int64(period)

	// Reports names are utc timestamps. Get the most recent report within the period window.
	var report Report
	err = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("failed to access path: %v", err)
		}

		// Skip subdirectories.
		if d.IsDir() && path != dir {
			return filepath.SkipDir
		}

		r, err := New(path)
		if errors.Is(err, ErrInvalidReportExt) || errors.Is(err, ErrInvalidReportName) {
			slog.Info("Skipping non-report file", "file", d.Name(), "error", err)
			return nil
		} else if err != nil {
			return fmt.Errorf("failed to create report object: %v", err)
		}

		if r.TimeStamp < periodStart {
			return nil
		}
		if r.TimeStamp >= periodEnd {
			return filepath.SkipDir
		}

		report = r
		return nil
	})

	if err != nil {
		return Report{}, err
	}

	return report, nil
}

// GetPerPeriod returns the latest report within each period window for a given directory.
// The key of the map is the start of the period window, and the value is a Report object.
//
// If period is 1, then all reports in the dir are returned.
func GetPerPeriod(dir string, period int) (map[int64]Report, error) {
	if period <= 0 {
		return nil, ErrInvalidPeriod
	}

	reports := make(map[int64]Report)
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("failed to access path: %v", err)
		}

		if d.IsDir() && path != dir {
			return filepath.SkipDir
		}

		r, err := New(path)
		if errors.Is(err, ErrInvalidReportExt) || errors.Is(err, ErrInvalidReportName) {
			slog.Info("Skipping non-report file", "file", d.Name(), "error", err)
			return nil
		} else if err != nil {
			return fmt.Errorf("failed to create report object: %v", err)
		}

		periodStart := r.TimeStamp - (r.TimeStamp % int64(period))
		if existingReport, ok := reports[periodStart]; !ok || existingReport.TimeStamp < r.TimeStamp {
			reports[periodStart] = r
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return reports, nil
}

// GetAll returns all reports in a given directory.
// Reports are expected to have the correct file extension, and have a name which can be parsed by a timestamp.
// Does not traverse subdirectories.
func GetAll(dir string) ([]Report, error) {
	reports := make([]Report, 0)
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("failed to access path: %v", err)
		}

		if d.IsDir() && path != dir {
			return filepath.SkipDir
		}

		r, err := New(path)
		if errors.Is(err, ErrInvalidReportExt) || errors.Is(err, ErrInvalidReportName) {
			slog.Info("Skipping non-report file", "file", d.Name(), "error", err)
			return nil
		} else if err != nil {
			return fmt.Errorf("failed to create report object: %v", err)
		}

		reports = append(reports, r)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return reports, nil
}
