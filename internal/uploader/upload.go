package uploader

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/ubuntu/ubuntu-insights/internal/constants"
	"github.com/ubuntu/ubuntu-insights/internal/fileutils"
	"github.com/ubuntu/ubuntu-insights/internal/reportutils"
)

var (
	// ErrReportNotMature is returned when a report is not mature enough to be uploaded.
	ErrReportNotMature = errors.New("report is not mature enough to be uploaded")
)

// Upload uploads the reports corresponding to the source to the configured server.
// Does not do duplicate checks.
func (um Uploader) Upload(force bool) error {
	slog.Debug("Uploading reports")

	consent, err := um.consentM.HasConsent(um.source)
	if err != nil {
		return fmt.Errorf("upload failed to get consent state: %v", err)
	}

	reports, err := reportutils.GetAllReports(um.collectedDir)
	if err != nil {
		return fmt.Errorf("failed to get reports: %v", err)
	}

	url, err := um.getURL()
	if err != nil {
		return fmt.Errorf("failed to get URL: %v", err)
	}

	var wg sync.WaitGroup
	for _, name := range reports {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			err := um.upload(name, url, consent, force)
			if errors.Is(err, ErrReportNotMature) {
				slog.Debug("Skipped report upload, not mature enough", "file", name, "source", um.source)
			} else if err != nil {
				slog.Warn("Failed to upload report", "file", name, "source", um.source, "error", err)
			}
		}(name)
	}
	wg.Wait()

	return nil
}

// upload uploads an individual report to the server. It returns an error if the report is not mature enough to be uploaded, or if the upload fails.
// It also moves the report to the uploaded directory after a successful upload.
func (um Uploader) upload(name, url string, consent, force bool) error {
	slog.Debug("Uploading report", "file", name, "consent", consent, "force", force)

	// TODO… pass the Report object directly.
	ts, err := reportutils.GetReportTime(name)
	if err != nil {
		return fmt.Errorf("failed to parse report time from filename: %v", err)
	}

	if um.timeProvider.Now().Add(time.Duration(-um.minAge)*time.Second).Before(time.Unix(ts, 0)) && !force {
		return ErrReportNotMature
	}

	// Check for duplicate reports.
	_, err = os.Stat(filepath.Join(um.uploadedDir, name))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to check if report has already been uploaded: %v", err)
	}
	if err == nil && !force {
		// TODO: What to do with the original file? Should we clean it up?
		// Should we move it elsewhere for investigation in a "tmp" and clean it afterwards?
		return fmt.Errorf("report has already been uploaded")
	}

	origData, err := um.readJSON(name)
	if err != nil {
		return fmt.Errorf("failed to get payload: %v", err)
	}
	data := origData
	if !consent {
		data, err = json.Marshal(constants.OptOutJSON)
		if err != nil {
			return fmt.Errorf("failed to marshal opt-out JSON data: %v", err)
		}
	}
	slog.Debug("Uploading", "payload", data)

	if um.dryRun {
		slog.Debug("Dry run, skipping upload")
		return nil
	}

	// Move report first to avoid the situation where the report is sent, but not marked as sent.
	// TODO: maybe a method on Reports ?
	if err := um.moveReport(filepath.Join(um.uploadedDir, name), filepath.Join(um.collectedDir, name), data); err != nil {
		return fmt.Errorf("failed to move report after uploading: %v", err)
	}
	if err := send(url, data); err != nil {
		if moveErr := um.moveReport(filepath.Join(um.collectedDir, name), filepath.Join(um.uploadedDir, name), origData); moveErr != nil {
			return fmt.Errorf("failed to send data: %v, and failed to restore the original report: %v", err, moveErr)
		}
		return fmt.Errorf("failed to send data: %v", err)
	}

	return nil
}

func (um Uploader) getURL() (string, error) {
	u, err := url.Parse(um.baseServerURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse base server URL %s: %v", um.baseServerURL, err)
	}
	u.Path = path.Join(u.Path, um.source)
	return u.String(), nil
}

// readJSON reads the JSON data from the report file.
func (um Uploader) readJSON(file string) ([]byte, error) {
	// Read the report file
	data, err := os.ReadFile(path.Join(um.collectedDir, file))
	if err != nil {
		return nil, fmt.Errorf("failed to read report file: %v", err)
	}

	if !json.Valid(data) {
		return nil, fmt.Errorf("invalid JSON data in report file")
	}

	return data, nil
}

func (um Uploader) moveReport(writePath, removePath string, data []byte) error {
	// (Report).MarkAsProcessed(data)
	// (Report).UndoProcessed
	// moveReport writes the data to the writePath, and removes the matching file from the removePath.
	// dest, src, data

	if err := fileutils.AtomicWrite(writePath, data); err != nil {
		return fmt.Errorf("failed to write report: %v", err)
	}

	if err := os.Remove(removePath); err != nil {
		return fmt.Errorf("failed to remove report: %v", err)
	}

	return nil
}

func send(url string, data []byte) error {
	slog.Debug("Sending data to server", "url", url, "data", data)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: time.Second * 10}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status code %d", resp.StatusCode)
	}

	return nil
}
