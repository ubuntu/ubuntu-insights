package models

import "encoding/json"

// V1Model represents the version 1 model for the ingest service.
type V1Model struct {
	InsightsVersion string `json:"insightsVersion"`
	SystemInfo      struct {
		Hardware struct {
			Product struct {
				Family string `json:"family"`
				Name   string `json:"name"`
				Vendor string `json:"vendor"`
			} `json:"product,omitzero"`

			CPU struct {
				Name    string `json:"name"`
				Vendor  string `json:"vendor"`
				Arch    string `json:"architecture"`
				Cpus    uint64 `json:"cpus"`
				Sockets uint64 `json:"sockets"`
				Cores   uint64 `json:"coresPerSocket"`
				Threads uint64 `json:"threadsPerCore"`
			} `json:"cpu,omitzero"`

			GPUs []struct {
				Name   string `json:"name,omitempty"`
				Device string `json:"device,omitempty"`
				Vendor string `json:"vendor"`
				Driver string `json:"driver"`
			} `json:"gpus,omitempty"`

			Mem struct {
				Total int `json:"size"`
			} `json:"memory,omitzero"`

			Blks []v1Disk `json:"disks,omitempty"`

			Screens []struct {
				PhysicalResolution string `json:"physicalResolution,omitempty"`
				Size               string `json:"size,omitempty"`
				Resolution         string `json:"resolution,omitempty"`
				RefreshRate        string `json:"refreshRate,omitempty"`
			} `json:"screens,omitempty"`
		} `json:"hardware"`

		Software struct {
			OS struct {
				Family  string `json:"family"`
				Distro  string `json:"distribution"`
				Version string `json:"version"`
				Edition string `json:"edition,omitempty"`
			} `json:"os,omitzero"`
			Timezone string `json:"timezone,omitempty"`
			Lang     string `json:"language,omitempty"`
			Bios     struct {
				Vendor  string `json:"vendor"`
				Version string `json:"version"`
			} `json:"bios,omitzero"`
		} `json:"software"`

		Platform      json.RawMessage `json:"platform"`
		SourceMetrics json.RawMessage `json:"sourceMetrics"`
	}
}

type v1Disk struct {
	Size uint64 `json:"size"`
	Type string `json:"type,omitempty"`

	Children []v1Disk `json:"children,omitempty"`
}
