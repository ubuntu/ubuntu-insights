package commands

import (
	"bytes"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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
