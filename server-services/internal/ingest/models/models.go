// Package models provides the data structures for payloads used in the ingest service.
package models

import (
	"encoding/json"
)

// TargetModels is an interface that represents the root target models for the ingest service.
type TargetModels interface {
	TargetModel | LegacyTargetModel
}

// TargetModel represents the target model for the ingest service.
// It is the current structure used before the data is inserted into the database.
type TargetModel struct {
	InsightsVersion string           `json:"insightsVersion,omitempty"`
	CollectionTime  int64            `json:"collectionTime,omitempty"`
	SystemInfo      TargetSystemInfo `json:"systemInfo,omitzero"`
	SourceMetrics   json.RawMessage  `json:"sourceMetrics,omitempty"`

	OptOut bool `json:"OptOut,omitempty"`

	Extras map[string]any `json:",omitzero" mapstructure:",remain"` // This field is used to hold any extra data that doesn't fit into the other fields.
}

// TargetSystemInfo represents the target system information for the ingest service.
type TargetSystemInfo struct {
	Hardware json.RawMessage `json:"hardware,omitempty"`
	Software json.RawMessage `json:"software,omitempty"`
	Platform json.RawMessage `json:"platform,omitempty"`

	Extras map[string]any `json:",omitzero" mapstructure:",remain"` // This field is used to hold any extra data that doesn't fit into the other fields.
}

// LegacyTargetModel represents the legacy ubuntu report target model for the ingest service.
type LegacyTargetModel struct {
	OptOut bool `json:"OptOut,omitempty"`

	Fields map[string]any `json:",omitzero" mapstructure:",remain"` // This field holds all other fields that are not explicitly defined in the struct.
}
