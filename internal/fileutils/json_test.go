package fileutils_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/fileutils"
	"github.com/ubuntu/ubuntu-insights/internal/testutils"
)

func TestUnmarshalJSON(t *testing.T) {
	t.Parallel()

	type st struct {
		Str string
		I   int
	}

	getData := func(t *testing.T, input []st) []byte {
		t.Helper()

		var b []byte
		var err error
		if len(input) == 1 {
			b, err = json.Marshal(input[0])
		} else if input == nil {
			b = []byte("")
		} else {
			b, err = json.Marshal(input)
		}

		require.NoError(t, err, "Setup: failed to marshal test data")
		return b
	}

	tests := map[string]struct {
		input []byte

		wantErr bool
	}{
		"empty list": {
			input: getData(t, []st{}),
		},
		"single object": {
			input: getData(t, []st{{Str: "test", I: 1}}),
		},
		"multiple objects": {
			input: getData(t, []st{{Str: "test"}, {Str: "test2", I: 2}}),
		},

		// Error cases
		"Nil": {
			input:   getData(t, nil),
			wantErr: true,
		},
		"Junk data": {
			input: func() []byte {
				b, err := json.Marshal("some junk data")
				require.NoError(t, err, "Setup: failed to marshal junk data")
				return b
			}(),
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := fileutils.UnmarshalJSON[st](tc.input)
			if tc.wantErr {
				require.Error(t, err, "expected error but got none")
				return
			}
			require.NoError(t, err)

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			assert.Equal(t, want, got, "unmarshalled data should match golden file")
		})
	}
}
