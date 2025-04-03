// Package fileutils provides utility functions for handling files.
package fileutils

import (
	"encoding/json"
	"fmt"
	"io"
)

// ParseJSON unmarshals the data in r into v.
func ParseJSON(r io.Reader, v any) error {
	// Read the entire content of the io.Reader first to check for errors even if valid json is first.
	buf, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("error reading from io.Reader: %v", err)
	}

	err = json.Unmarshal(buf, v)
	if err != nil {
		return fmt.Errorf("couldn't parse JSON: %v", err)
	}
	return nil
}

// UnmarshalJSON unmarshals the data in b into a list of v.
func UnmarshalJSON[T any](data []byte) ([]T, error) {
	var v []T
	if err := json.Unmarshal(data, &v); err == nil {
		return v, nil
	}

	// Fallback to single object
	var single T
	if err := json.Unmarshal(data, &single); err != nil {
		return nil, fmt.Errorf("couldn't parse JSON: %v", err)
	}
	return []T{single}, nil
}
