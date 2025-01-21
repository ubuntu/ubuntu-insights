package uploader

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"os"
	"path"
	"sync"
	"time"

	"github.com/ubuntu/ubuntu-insights/internal/constants"
	"github.com/ubuntu/ubuntu-insights/internal/fileutils"
	"github.com/ubuntu/ubuntu-insights/internal/reportutils"
)

// Upload uploads the reports corresponding to the source to the configured server.
// Does not do duplicate checks.
func (um Manager) Upload() error {
	slog.Debug("Uploading reports")

	gConsent, err := um.consentManager.GetConsentState("")
	if err != nil {
		return fmt.Errorf("upload failed to get global consent state: %v", err)
	}

	sConsent, err := um.consentManager.GetConsentState(um.source)
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
			if err := um.upload(file, url, gConsent && sConsent); err != nil {
				slog.Warn("Failed to upload report", "file", file, "source", um.source, "error", err)
			}
		}(file)
	}
	wg.Wait()

	return nil
}

func (um Manager) upload(file, url string, consent bool) error {
	slog.Debug("Uploading report", "file", file, "consent", consent)

	ts, err := reportutils.GetReportTime(file)
	if err != nil {
		return fmt.Errorf("failed to parse report time from filename: %v", err)
	}

	// Report maturity check
	if um.minAge > math.MaxInt64 {
		return fmt.Errorf("min age is too large: %d", um.minAge)
	}
	if ts+int64(um.minAge) > um.timeProvider.NowUnix() {
		slog.Debug("Skipping report due to min age", "timestamp", file, "minAge", um.minAge)
		return ErrReportNotMature
	}

	payload, err := um.getPayload(file, consent)
	if err != nil {
		return fmt.Errorf("failed to get payload: %v", err)
	}
	slog.Debug("Uploading", "payload", payload)

	if !um.dryRun {
		if err := send(url, payload); err != nil {
			return fmt.Errorf("failed to send data: %v", err)
		}

		if err := um.moveReport(file, payload); err != nil {
			return fmt.Errorf("failed to move report after uploading: %v", err)
		}
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

func (um Manager) getPayload(file string, consent bool) ([]byte, error) {
	path := path.Join(um.collectedDir, file)
	var jsonData map[string]interface{}

	data, err := json.Marshal(constants.OptOutJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON data")
	}
	if consent {
		// Read the report file
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read report file: %v", err)
		}

		// Remashal the JSON data to ensure it is valid
		if err := json.Unmarshal(data, &jsonData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON data: %v", err)
		}

		// Marshal the JSON data back to bytes
		data, err = json.Marshal(jsonData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal JSON data: %v", err)
		}

		return data, nil
	}

	return data, nil
}

// moveReport writes the uploaded report to the uploaded directory, and removes it from the collected directory.
func (um Manager) moveReport(file string, data []byte) error {
	err := fileutils.AtomicWrite(path.Join(um.uploadedDir, file), data)
	if err != nil {
		return fmt.Errorf("failed to write report to uploaded directory: %v", err)
	}

	err = os.Remove(path.Join(um.collectedDir, file))
	if err != nil {
		return fmt.Errorf("failed to remove report from collected directory: %v", err)
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
		return fmt.Errorf("failed to send data: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status code %d", resp.StatusCode)
	}

	return nil
}
