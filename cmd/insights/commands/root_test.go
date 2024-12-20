package commands

import (
	"bytes"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestSetVerbosity(t *testing.T) {
	testCases := []struct {
		name    string
		pattern []bool
	}{
		{
			name:    "true",
			pattern: []bool{true},
		},
		{
			name:    "false",
			pattern: []bool{false},
		},
		{
			name:    "true false",
			pattern: []bool{true, false},
		},
		{
			name:    "false true",
			pattern: []bool{false, true},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			log.Logger = log.Output(zerolog.ConsoleWriter{Out: &buf})

			for _, p := range tc.pattern {
				buf = bytes.Buffer{}
				verbose = p
				setVerbosity()

				log.Debug().Msg(tc.name + " debug message")
				if p {
					assert.Equal(t, zerolog.DebugLevel, zerolog.GlobalLevel())
					assert.Contains(t, buf.String(), tc.name+" debug message")
				} else {
					assert.Equal(t, zerolog.InfoLevel, zerolog.GlobalLevel())
					assert.NotContains(t, buf.String(), tc.name+" debug message")
				}
			}
		})
	}
}

func TestRootFlags(t *testing.T) {
	testCases := []struct {
		name     string
		short    string
		required bool
		dirname  bool
	}{
		{
			name:     "verbose",
			short:    "v",
		},
		{
			name:    "consent-dir",
			dirname: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			flag := rootCmd.PersistentFlags().Lookup(tc.name)
			assert.NotNil(t, flag)
			assert.Equal(t, tc.short, flag.Shorthand)

			if tc.required {
				assert.Equal(t, "true", flag.Annotations[cobra.BashCompOneRequiredFlag][0])
			} else {
				assert.Nil(t, flag.Annotations[cobra.BashCompOneRequiredFlag])
			}

			if tc.dirname {
				assert.Equal(t, []string{}, flag.Annotations[cobra.BashCompSubdirsInDir])
			} else {
				assert.Nil(t, flag.Annotations[cobra.BashCompSubdirsInDir])
			}
		})
	}
}
