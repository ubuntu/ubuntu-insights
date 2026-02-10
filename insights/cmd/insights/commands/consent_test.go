package commands_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/common/testutils"
)

func TestConsent(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		args []string

		platformConsent consentFixture

		wantErr      bool
		wantUsageErr bool
	}{
		// Get
		"Get platform true":    {args: []string{"consent"}, platformConsent: fixtureTrue},
		"Get platform false":   {args: []string{"consent"}, platformConsent: fixtureFalse},
		"Get source true":      {args: []string{"consent", "True"}},
		"Get source false":     {args: []string{"consent", "False"}},
		"Get multiple sources": {args: []string{"consent", "True", "False"}},
		"Get platform empty":   {args: []string{"consent"}, platformConsent: fixtureEmpty},
		"Get platform bad key": {args: []string{"consent"}, platformConsent: fixtureBadKey},

		// Get Errors
		"Get Multiple Sources errors when source is missing ": {args: []string{"consent", "True", "Unknown"}, wantErr: true},
		"Get Multiple Sources errors when source file bad":    {args: []string{"consent", "True", "Bad-File", "False"}, wantErr: true},

		"Get errors when platform missing":   {args: []string{"consent"}, wantErr: true},
		"Get errors when platform bad file":  {args: []string{"consent"}, platformConsent: fixtureBadFile, wantErr: true},
		"Get errors when platform bad ext":   {args: []string{"consent"}, platformConsent: fixtureBadExt, wantErr: true},
		"Get errors when platform bad value": {args: []string{"consent"}, platformConsent: fixtureBadValue, wantErr: true},

		"Get errors when source missing": {args: []string{"consent", "unknown"}, wantErr: true},

		// Set
		"Set platform to new value":   {args: []string{"consent", "--state=false"}, platformConsent: fixtureTrue},
		"Set platform to same value":  {args: []string{"consent", "--state=true"}, platformConsent: fixtureTrue},
		"Set source to new value":     {args: []string{"consent", "False", "--state=true"}},
		"Set source to same value":    {args: []string{"consent", "True", "--state=true"}},
		"Set multiple sources:":       {args: []string{"consent", "True", "False", "-s=false"}},
		"Set new source":              {args: []string{"consent", "Unknown", "--state=true"}},
		"Set existing and new source": {args: []string{"consent", "True", "Unknown", "-s=true"}},
		"Set existing and bad source": {args: []string{"consent", "True", "Bad-File", "False", "-s=true"}},

		"Set shorthand True":                 {args: []string{"consent", "-s=true"}},
		"Does not error with the quiet flag": {args: []string{"consent", "--state=false", "--quiet"}},

		// Usage Errors
		"Usage errors when passing bad flag":                    {args: []string{"consent", "-unknown"}, wantUsageErr: true, wantErr: true},
		"Usage errors when unparsable state is passed":          {args: []string{"consent", "-s=bad"}, wantUsageErr: true, wantErr: true},
		"Usage errors when verbose and quiet are used together": {args: []string{"consent", "--verbose", "--quiet"}, wantErr: true, wantUsageErr: true},
		"Usage errors propagate with the quiet flag":            {args: []string{"consent", "-s=bad", "--quiet"}, wantErr: true, wantUsageErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			app, configDir := newAppForTests(t, tc.args, tc.platformConsent)

			err := app.Run()
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tc.wantUsageErr {
				assert.True(t, app.UsageError())
			} else {
				assert.False(t, app.UsageError())
			}

			got, err := testutils.GetDirContents(t, configDir, 2)
			require.NoError(t, err)

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			require.Equal(t, want, got, "Unexpected consent files state")
		},
		)
	}
}
