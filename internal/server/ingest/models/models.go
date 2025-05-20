// Package models provides the data structures for payloads used in the ingest service.
package models

import (
	"encoding/json"

	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/hardware"
	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/software"
)

// VersionedEnvelope represents the versioned envelope for the ingest service.
// It contains the version of the insights report and the raw JSON data, which will be parsed later.
//
// The `Raw` field is used to hold the rest of the JSON until we know how to parse it.
type VersionedEnvelope struct {
	InsightsVersion string          `json:"insightsVersion"`
	Raw             json.RawMessage `json:"-"`
}

// TargetModel represents the target model for the ingest service.
// It is the current structure used before the data is inserted into the database.
type TargetModel struct {
	InsightsVersion string           `json:"insightsVersion,omitempty"`
	CollectionTime  int64            `json:"collectionTime,omitempty"`
	SystemInfo      TargetSystemInfo `json:"systemInfo,omitzero"`

	OptOut bool `json:"optOut,omitempty"`
}

// TargetSystemInfo represents the target system information for the ingest service.
type TargetSystemInfo struct {
	Hardware      hardware.Info   `json:"hardware,omitzero"`
	Software      software.Info   `json:"software,omitzero"`
	Platform      json.RawMessage `json:"platform,omitempty"`
	SourceMetrics json.RawMessage `json:"sourceMetrics,omitempty"`
}
