//go:build linux || darwin

package cmdutils_test

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/insights/internal/cmdutils"
)

func TestRunSetsLocaleEnvVars(t *testing.T) {
	// Not parallel because sub-tests use t.Setenv

	tests := map[string]struct {
		presetLang     string
		presetLcAll    string
		presetLanguage string
	}{
		"No locale set": {},
		"French locale": {
			presetLang:     "fr_FR.UTF-8",
			presetLcAll:    "fr_FR.UTF-8",
			presetLanguage: "fr_FR:fr",
		},
		"German locale": {
			presetLang:     "de_DE.UTF-8",
			presetLcAll:    "de_DE.UTF-8",
			presetLanguage: "de_DE:de",
		},
		"Japanese locale": {
			presetLang:     "ja_JP.UTF-8",
			presetLcAll:    "ja_JP.UTF-8",
			presetLanguage: "ja_JP:ja",
		},
		"Mixed locales": {
			presetLang:     "en_GB.UTF-8",
			presetLcAll:    "zh_CN.UTF-8",
			presetLanguage: "ko_KR:ko",
		},
	}

	// Regex to match locale output lines: KEY=value or KEY="value"
	localeLineRegex := regexp.MustCompile(`^([A-Z_]+)=("?)(.*)("?)$`)

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Not parallel - uses t.Setenv

			if tc.presetLang != "" {
				t.Setenv("LANG", tc.presetLang)
			}
			if tc.presetLcAll != "" {
				t.Setenv("LC_ALL", tc.presetLcAll)
			}
			if tc.presetLanguage != "" {
				t.Setenv("LANGUAGE", tc.presetLanguage)
			}

			stdout, stderr, err := cmdutils.Run(context.Background(), "locale")

			require.NoError(t, err, "Run should succeed")
			require.Empty(t, stderr.String(), "stderr should be empty")

			// Parse locale output and verify all values are "C" (the POSIX locale)
			lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
			require.NotEmpty(t, lines, "locale output should not be empty")

			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}

				matches := localeLineRegex.FindStringSubmatch(line)
				require.NotNil(t, matches, "line should match locale format KEY=value: %q", line)

				key := matches[1]
				value := matches[3]
				// Strip surrounding quotes if present
				value = strings.Trim(value, `"`)

				require.Equal(t, "C", value, "locale variable %s should be set to C, got %q", key, value)
			}
		})
	}
}
