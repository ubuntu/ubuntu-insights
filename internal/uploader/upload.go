package uploader

import (
	"bytes"
	"encoding/json"
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

// Upload uploads the reports corresponding to the source to the configured server.
// Does not do duplicate checks.
func (um Manager) Upload(force bool) error {
	slog.Debug("Uploading reports")

	gConsent, err := um.consentM.GetConsentState("")
	if err != nil {
		return fmt.Errorf("upload failed to get global consent state: %v", err)
	}

	sConsent, err := um.consentM.GetConsentState(um.source)
	if err != nil {
		return fmt.Errorf("upload failed to get source consent state: %v", err)
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
	for _, file := range reports {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()
			if err := um.upload(file, url, gConsent && sConsent, force); err != nil {
				slog.Warn("Failed to upload report", "file", file, "source", um.source, "error", err)
			}
		}(file)
	}
	wg.Wait()

	return nil
}

// upload uploads an individual report to the server. It returns an error if the report is not mature enough to be uploaded, or if the upload fails.
// It also moves the report to the uploaded directory after a successful upload.
func (um Manager) upload(fName, url string, consent, force bool) error {
	slog.Debug("Uploading report", "file", fName, "consent", consent, "force", force)

	ts, err := reportutils.GetReportTime(fName)
	if err != nil {
		return fmt.Errorf("failed to parse report time from filename: %v", err)
	}

	if ts > um.timeProvider.NowUnix()-um.minAge && !force {
		return fmt.Errorf("report is not mature enough to be uploaded")
	}

	// Check for duplicate reports.
	fileExists, err := fileutils.FileExists(filepath.Join(um.uploadedDir, fName))
	if err != nil {
		return fmt.Errorf("failed to check if report has already been uploaded: %v", err)
	}
	if fileExists && !force {
		return fmt.Errorf("report has already been uploaded")
	}

	origData, err := um.readPayload(fName)
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
	if err := um.moveReport(filepath.Join(um.uploadedDir, fName), filepath.Join(um.collectedDir, fName), data); err != nil {
		return fmt.Errorf("failed to move report after uploading: %v", err)
	}
	if err := send(url, data); err != nil {
		if moveErr := um.moveReport(filepath.Join(um.collectedDir, fName), filepath.Join(um.uploadedDir, fName), origData); moveErr != nil {
			return fmt.Errorf("failed to send data: %v, and failed to restore the original report: %v", err, moveErr)
		}
		return fmt.Errorf("failed to send data: %v", err)
	}

	return nil
}

func (um Manager) getURL() (string, error) {
	u, err := url.Parse(um.baseServerURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse base server URL %s: %v", um.baseServerURL, err)
	}
	u.Path = path.Join(u.Path, um.source)
	return u.String(), nil
}

func (um Manager) readPayload(file string) ([]byte, error) {
	var jsonData map[string]interface{}

	// Read the report file
	data, err := os.ReadFile(path.Join(um.collectedDir, file))
	if err != nil {
		return nil, fmt.Errorf("failed to read report file: %v", err)
	}

	// Remarshal the JSON data to ensure it is valid
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON data: %v", err)
	}

	data, err = json.Marshal(jsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON data: %v", err)
	}

	return data, nil
}

// moveReport writes the data to the writePath, and removes the matching file from the removePath.
func (um Manager) moveReport(writePath, removePath string, data []byte) error {
	err := fileutils.AtomicWrite(writePath, data)
	if err != nil {
		return fmt.Errorf("failed to write report: %v", err)
	}

	err = os.Remove(removePath)
	if err != nil {
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
