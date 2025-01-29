// Package reportutils provides utility functions for handling reports.
package reportutils

// package report

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ubuntu/ubuntu-insights/internal/constants"
)

// ErrInvalidPeriod is returned when a function requiring a period, received an invalid, period that isn't a non-negative integer.
var ErrInvalidPeriod = errors.New("invalid period, period should be a positive integer")

// type Report struct{}

// GetPeriodStart returns the start of the period window for a given period in seconds.
func GetPeriodStart(period int) (int64, error) {
	if period <= 0 {
		return 0, ErrInvalidPeriod
	}
	utcTime := time.Now().UTC().Unix()
	return utcTime - (utcTime % int64(period)), nil
}

// GetReportTime returns a int64 representation of the report time from the report path.
func GetReportTime(path string) (int64, error) {
	fileName := filepath.Base(path)
	return strconv.ParseInt(strings.TrimSuffix(fileName, filepath.Ext(fileName)), 10, 64)
}

// Path is a method on Report

// GetReportPath returns the path for the most recent report within a period window, returning an empty string if no report is found.
// Not inclusive of the period end (periodStart + period).
//
// For example, given reports 1 and 7, with time 2 and period 7, the function will return the path for report 1.
func GetReportPath(dir string, time int64, period int) (string, error) {
	if period <= 0 {
		return "", ErrInvalidPeriod
	}

	periodStart := time - (time % int64(period))
	periodEnd := periodStart + int64(period)

	// Reports names are utc timestamps. Get the most recent report within the period window.
	var mostRecentReportPath string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			slog.Error("Failed to access path", "path", path, "error", err)
			return err
		}

		// Skip subdirectories.
		if d.IsDir() && path != dir {
			return filepath.SkipDir
		}

		if filepath.Ext(d.Name()) != constants.ReportExt {
			slog.Info("Skipping non-report file, invalid extension", "file", d.Name())
			return nil
		}

		reportTime, err := GetReportTime(d.Name())
		if err != nil {
			slog.Info("Skipping non-report file, invalid file name", "file", d.Name())
			return nil
		}

		if reportTime < periodStart {
			return nil
		}
		if reportTime >= periodEnd {
			return filepath.SkipDir
		}

		mostRecentReportPath = path
		return nil
	})

	if err != nil {
		return "", err
	}

	return mostRecentReportPath, nil
}

// -> map[int64]Report

// GetReports returns the paths for the latest report within each period window for a given directory.
// The key of the map is the start of the period window, and the report timestamp.
//
// If period is 1, then all reports in the dir are returned.
func GetReports(dir string, period int) (map[int64]int64, error) {
	if period <= 0 {
		return nil, ErrInvalidPeriod
	}

	reports := make(map[int64]int64)
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			slog.Error("Failed to access path", "path", path, "error", err)
			return err
		}

		if d.IsDir() && path != dir {
			return filepath.SkipDir
		}

		if filepath.Ext(d.Name()) != constants.ReportExt {
			slog.Info("Skipping non-report file, invalid extension", "file", d.Name())
			return nil
		}

		reportTime, err := GetReportTime(d.Name())
		if err != nil {
			slog.Info("Skipping non-report file, invalid file name", "file", d.Name())
			return nil
		}

		periodStart := reportTime - (reportTime % int64(period))
		if existingReport, ok := reports[periodStart]; !ok || existingReport < reportTime {
			reports[periodStart] = reportTime
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return reports, nil
}

// reports.GetAll(dir string) ([]Report, error)

// GetAllReports returns the filename for all reports within a given directory, which match the expected pattern.
// Does not traverse subdirectories.
func GetAllReports(dir string) ([]string, error) {
	reports := make([]string, 0)
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			slog.Error("Failed to access path", "path", path, "error", err)
			return err
		}

		if d.IsDir() && path != dir {
			return filepath.SkipDir
		}

		if filepath.Ext(d.Name()) != constants.ReportExt {
			slog.Info("Skipping non-report file, invalid extension", "file", d.Name())
			return nil
		}

		if _, err := GetReportTime(d.Name()); err != nil {
			slog.Info("Skipping non-report file, invalid file name", "file", d.Name())
			return nil
		}

		reports = append(reports, d.Name())
		return nil
	})

	if err != nil {
		return nil, err
	}

	return reports, nil
}
