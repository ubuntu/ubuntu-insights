package sysinfo

import (
	"encoding/json"
	"fmt"
	"io"
)

// parseJSON unmarshals the data in r into v.
func parseJSON(r io.Reader, v any) error {
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
