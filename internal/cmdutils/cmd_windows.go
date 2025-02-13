// Package cmdutils provides utility functions for running commands.
package cmdutils

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"time"
)

// listEntryRegex matches the key and value (if any) from list formatted output.
// For example: "Status   : OK " matches and has "Status", "OK".
// Or: "DitherType:" matches and has "DitherType", "".
// However: "   : OK" does not match.
var listEntryRegex = regexp.MustCompile(`(?m)^\s*(\S+)\s*:[^\S\n]*(.*?)\s*$`)

var listReplaceRegex = regexp.MustCompile(`\r?\n\s*`)

// listSplitRegex splits on two consecutive newlines, but \r needs special handling.
var listSplitRegex = regexp.MustCompile(`\r?\n\r?\n`)

// RunWMI runs the cmdlet specified by args and only includes fields in the filter.
// if filter is nil then nothing is filtered out.
func RunWMI(args []string, filter map[string]struct{}, log *slog.Logger) (out []map[string]string, err error) {
	defer func() {
		if err == nil && len(out) == 0 {
			err = fmt.Errorf("%v output contained no sections", args)
		}
	}()

	if filter != nil && len(filter) == 0 {
		return nil, fmt.Errorf("empty filter will always produce nothing for cmdlet %v", args)
	}

	stdout, stderr, err := RunWithTimeout(context.Background(), 15*time.Second, args[0], args[1:]...)
	if err != nil {
		return nil, err
	}
	if stderr.Len() > 0 {
		log.Info(fmt.Sprintf("%v output to stderr", args), "stderr", stderr)
	}

	sections := listSplitRegex.Split(stdout.String(), -1)
	out = make([]map[string]string, 0, len(sections))

	for _, section := range sections {
		if section == "" {
			continue
		}

		entries := listEntryRegex.FindAllStringSubmatch(section, -1)
		if len(entries) == 0 {
			log.Warn(fmt.Sprintf("%v output has malformed section", args), "section", section)
			continue
		}

		v := make(map[string]string, len(filter))
		for _, e := range entries {
			if filter != nil {
				if _, ok := filter[e[1]]; !ok {
					continue
				}
			}

			// Get-WmiObject injects newlines and whitespace into values for formatting
			v[e[1]] = listReplaceRegex.ReplaceAllString(e[2], "")
		}

		out = append(out, v)
	}

	return out, nil
}
