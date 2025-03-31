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
	"github.com/ubuntu/ubuntu-insights/internal/report"
)

var (
	// ErrReportNotMature is returned when a report is not mature enough to be uploaded.
	ErrReportNotMature = errors.New("report is not mature enough to be uploaded")
	// ErrSendFailure is returned when a report fails to be sent to the server, either due to a network error or a non-200 status code.
	ErrSendFailure = errors.New("report send failed")
)

// Upload uploads the reports corresponding to the source to the configured server.
//
// It will only upload reports that are mature enough, and have not been uploaded before.
// If force is true, maturity and duplicate check will be skipped.
func (um Uploader) Upload(source string, force bool) error {
	slog.Debug("Uploading reports")

	if source == "" {
		return fmt.Errorf("source cannot be an empty string")
	}

	collectedDir := filepath.Join(um.cacheDir, source, constants.LocalFolder)
	uploadedDir := filepath.Join(um.cacheDir, source, constants.UploadedFolder)

	if err := um.makeDirs(collectedDir, uploadedDir); err != nil {
		return err
	}

	consent, err := um.consent.HasConsent(source)
	if err != nil {
		return fmt.Errorf("upload failed to get consent state: %w", err)
	}

	reports, err := report.GetAll(collectedDir)
	if err != nil {
		return fmt.Errorf("failed to get reports: %v", err)
	}

	url, err := um.getURL(source)
	if err != nil {
		return fmt.Errorf("failed to get URL: %v", err)
	}

	mu := &sync.Mutex{}
	var uploadError error
	var wg sync.WaitGroup
	for _, r := range reports {
		wg.Add(1)
		go func(r report.Report) {
			defer wg.Done()
			err := um.upload(r, uploadedDir, url, consent, force)
			if errors.Is(err, ErrReportNotMature) {
				slog.Debug("Skipped report upload, not mature enough", "file", r.Name, "source", source)
			} else if err != nil {
				slog.Warn("Failed to upload report", "file", r.Name, "source", source, "error", err)
				mu.Lock()
				defer mu.Unlock()
				uploadError = errors.Join(uploadError, fmt.Errorf("%s upload failed for report %s: %w", source, r.Name, err))
			}
		}(r)
	}
	wg.Wait()

	if um.dryRun {
		return uploadError
	}

	return errors.Join(report.Cleanup(uploadedDir, um.maxReports), uploadError)
}

// BackoffUpload behaves like Upload, but if there are any send errors, it will retry the upload after a backoff period.
// The backoff period starts at 30 seconds, and doubles with each retry, up to the configured report timeout (default 30 minutes).
// If the report timeout is surpassed, the upload will stop.
func (um Uploader) BackoffUpload(source string, force bool) (err error) {
	slog.Debug("Uploading reports with backoff")

	wait := um.initialRetryPeriod
	for {
		err = um.Upload(source, force)
		if !errors.Is(err, ErrSendFailure) {
			break
		}
		wait *= 2
		if wait > um.maxRetryPeriod {
			slog.Warn("Report timeout reached, stopping upload")
			break
		}
		slog.Warn("Failed to send report, retrying upload after backoff period", "seconds", wait.Seconds(), "error", err)
		time.Sleep(wait)
	}

	return err
}

// upload uploads an individual report to the server. It returns an error if the report is not mature enough to be uploaded, or if the upload fails.
// It also moves the report to the uploaded directory after a successful upload.
func (um Uploader) upload(r report.Report, uploadedDir, url string, consent, force bool) error {
	slog.Debug("Uploading report", "file", r.Name, "consent", consent, "force", force)

	if um.timeProvider.Now().Add(-um.minAge).Before(time.Unix(r.TimeStamp, 0)) && !force {
		return ErrReportNotMature
	}

	// Check for duplicate reports.
	_, err := os.Stat(filepath.Join(uploadedDir, r.Name))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to check if report has already been uploaded: %v", err)
	}
	if err == nil && !force {
		return fmt.Errorf("report has already been uploaded")
	}

	origData, err := r.ReadJSON()
	if err != nil {
		return fmt.Errorf("failed to read report: %v", err)
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
	r, err = r.MarkAsProcessed(uploadedDir, data)
	if err != nil {
		return fmt.Errorf("failed to mark report as processed: %v", err)
	}
	if err := um.send(url, data); err != nil {
		if _, err := r.UndoProcessed(); err != nil {
			return fmt.Errorf("failed to send data: %v, and failed to restore the original report: %v", err, err)
		}
		return fmt.Errorf("failed to send data: %w", err)
	}

	return nil
}

func (um Uploader) getURL(source string) (string, error) {
	u, err := url.Parse(um.baseServerURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse base server URL %s: %v", um.baseServerURL, err)
	}
	u.Path = path.Join(u.Path, source)
	return u.String(), nil
}

func (um Uploader) send(url string, data []byte) error {
	slog.Debug("Sending data to server", "url", url, "data", data)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: um.responseTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return errors.Join(ErrSendFailure, fmt.Errorf("failed to send HTTP request: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Join(ErrSendFailure, fmt.Errorf("unexpected status code: %d", resp.StatusCode))
	}

	return nil
}

// makeDirs creates the directories for the collected and uploaded reports if they don't already exist.
func (um Uploader) makeDirs(collectedDir, uploadedDir string) error {
	if err := os.MkdirAll(collectedDir, 0750); err != nil {
		return fmt.Errorf("failed to create collected directory: %v", err)
	}
	if err := os.MkdirAll(uploadedDir, 0750); err != nil {
		return fmt.Errorf("failed to create uploaded directory: %v", err)
	}
	return nil
}
