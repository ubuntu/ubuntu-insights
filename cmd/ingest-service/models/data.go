// Package models provides the data structures for payloads used in the ingest service.
package models

import "time"

// FileData represents the structure of the JSON files processed by the ingest service.
type RawFileData struct {
	AppID         string                 `json:"AppID"`
	Generated     string                 `json:"Generated"`
	SchemaVersion string                 `json:"Schema Version"`
	Common        map[string]interface{} `json:"Common"`
	AppData       map[string]interface{} `json:"AppData"`
}

// DBFileData represents the structure of the data to be stored in the database.
type DBFileData struct {
	AppID         string
	Generated     time.Time
	SchemaVersion string
	Common        map[string]interface{}
	AppData       map[string]interface{}
}
