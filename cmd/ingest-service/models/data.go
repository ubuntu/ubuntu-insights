// Package models provides the data structures for payloads used in the ingest service.
package models

// FileData represents the structure of the JSON files processed by the ingest service.
type FileData struct {
	AppID         string                 `json:"AppID"`
	Generated     string                 `json:"Generated"`
	SchemaVersion string                 `json:"Schema Version"`
	Common        map[string]interface{} `json:"Common"`
	AppData       map[string]interface{} `json:"AppData"`
}
