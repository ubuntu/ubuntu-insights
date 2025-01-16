// Package reportutils provides utility functions for handling reports.
package reportutils

import (
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ubuntu/ubuntu-insights/internal/constants"
)

// GetPeriodStart returns the start of the period window for a given period in seconds.
func GetPeriodStart(period uint) uint64 {
	utcTime := uint64(time.Now().UTC().Unix())
	return utcTime - (utcTime % uint64(period))
}

// GetReportTime returns a unint64 representation of the report time from the report path.
func GetReportTime(reportPath string) (uint64, error) {
	fileName := filepath.Base(reportPath)
	return strconv.ParseUint(strings.TrimSuffix(fileName, filepath.Ext(fileName)), 10, 64)
}

// GetReportPath returns the path for the most recent report within a period window, returning an empty string if no report is found.
// Not inclusive of the period end (periodStart + period).
func GetReportPath(reportsDir string, time uint64, period uint) (string, error) {
	periodStart := time - (time % uint64(period))
	periodEnd := periodStart + uint64(period)

	// Reports names are utc timestamps. Get the most recent report within the period window.
	var mostRecentReportPath string
	files, err := os.ReadDir(reportsDir)
	if err != nil {
		slog.Error("Failed to read directory", "directory", reportsDir, "error", err)
		return "", err
	}

	for _, reportPath := range files {
		if filepath.Ext(reportPath.Name()) != constants.ReportExtension {
			slog.Info("Skipping non-report file, invalid extension", "file", reportPath.Name())
			continue
		}

		reportTime, err := GetReportTime(reportPath.Name())
		if err != nil {
			slog.Info("Skipping non-report file, invalid file name", "file", reportPath.Name())
			continue
		}

		if reportTime < periodStart {
			continue
		}
		if reportTime >= periodEnd {
			break
		}

		mostRecentReportPath = filepath.Join(reportsDir, reportPath.Name())
	}

	return mostRecentReportPath, nil
}
