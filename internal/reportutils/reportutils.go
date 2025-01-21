// Package reportutils provides utility functions for handling reports.
package reportutils

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
	files, err := os.ReadDir(dir)
	if err != nil {
		slog.Error("Failed to read directory", "directory", dir, "error", err)
		return "", err
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != constants.ReportExt {
			slog.Info("Skipping non-report file, invalid extension", "file", file.Name())
			continue
		}

		reportTime, err := GetReportTime(file.Name())
		if err != nil {
			slog.Info("Skipping non-report file, invalid file name", "file", file.Name())
			continue
		}

		if reportTime < periodStart {
			continue
		}
		if reportTime >= periodEnd {
			break
		}

		mostRecentReportPath = filepath.Join(dir, file.Name())
	}

	return mostRecentReportPath, nil
}

// GetReports returns the paths for the latest report within each period window for a given directory.
// The key of the map is the start of the period window, and the report timestamp.
func GetReports(dir string, period int) (map[int64]int64, error) {
	if period <= 0 {
		return nil, ErrInvalidPeriod
	}

	// Get the most recent report within each period window.
	files, err := os.ReadDir(dir)
	if err != nil {
		slog.Error("Failed to read directory", "directory", dir, "error", err)
		return nil, err
	}

	// Map to store the most recent report within each period window.
	reports := make(map[int64]int64)
	for _, file := range files {
		if filepath.Ext(file.Name()) != constants.ReportExt {
			slog.Info("Skipping non-report file, invalid extension", "file", file.Name())
			continue
		}

		reportTime, err := GetReportTime(file.Name())
		if err != nil {
			slog.Info("Skipping non-report file, invalid file name", "file", file.Name())
			continue
		}

		periodStart := reportTime - (reportTime % int64(period))
		if existingReport, ok := reports[periodStart]; !ok || existingReport < reportTime {
			reports[periodStart] = reportTime
		}
	}

	return reports, nil
}
