package sysinfo

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

func parseJSON(r io.Reader, v any) (any, error) {
	// Read the entire content of the io.Reader first to check for errors even if valid json is first
	buf, err := io.ReadAll(r)
	if err != nil {
		s := fmt.Sprintf("error reading from io.Reader: %v", err)
		return nil, errors.New(s)
	}

	err = json.Unmarshal(buf, v)
	if err != nil {
		s := fmt.Sprintf("couldn't parse JSON: %v", err)
		return nil, errors.New(s)
	}
	return v, nil
}
