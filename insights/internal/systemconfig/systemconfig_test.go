package systemconfig_test

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/common/testutils"
	"github.com/ubuntu/ubuntu-insights/insights/internal/constants"
	"github.com/ubuntu/ubuntu-insights/insights/internal/systemconfig"
)

func TestIsOptedOut(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		configFile string // file from testdata to copy as system-config.toml; empty means no file.

		want    bool
		wantErr bool
	}{
		"No config file returns false":     {},
		"Valid true config returns true":   {configFile: "valid_true-system-config.toml", want: true},
		"Valid false config returns false": {configFile: "valid_false-system-config.toml", want: false},
		"Empty config file returns false":  {configFile: "empty-system-config.toml", want: false},
		"Invalid value config errors":      {configFile: "invalid_value-system-config.toml", wantErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			if tc.configFile != "" {
				err := testutils.CopyFile(t,
					filepath.Join("testdata", tc.configFile),
					filepath.Join(dir, constants.SystemConfigFileName),
				)
				require.NoError(t, err, "Setup: failed to copy system config file")
			}

			m := systemconfig.New(slog.Default(), dir)
			got, err := m.IsOptedOut()
			if tc.wantErr {
				require.Error(t, err, "expected an error but got none")
				return
			}
			require.NoError(t, err, "got an unexpected error")
			require.Equal(t, tc.want, got)
		})
	}
}

func TestIsOptedOut_MissingDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	m := systemconfig.New(slog.Default(), filepath.Join(dir, "nonexistent"))

	got, err := m.IsOptedOut()
	require.NoError(t, err, "missing directory should not return an error")
	require.False(t, got, "missing directory should be treated as not opted out")
}

func TestSetOptOut(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		initialConfigFile string // file from testdata to copy as the initial system-config.toml
		setState          bool
	}{
		"New file, write false":      {},
		"New file, write true":       {setState: true},
		"Overwrite true with false":  {initialConfigFile: "valid_true-system-config.toml", setState: false},
		"Overwrite false with true":  {initialConfigFile: "valid_false-system-config.toml", setState: true},
		"Overwrite true with true":   {initialConfigFile: "valid_true-system-config.toml", setState: true},
		"Overwrite false with false": {initialConfigFile: "valid_false-system-config.toml", setState: false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			if tc.initialConfigFile != "" {
				err := testutils.CopyFile(t,
					filepath.Join("testdata", tc.initialConfigFile),
					filepath.Join(dir, constants.SystemConfigFileName),
				)
				require.NoError(t, err, "Setup: failed to copy initial system config file")
			}

			m := systemconfig.New(slog.Default(), dir)
			err := m.SetOptOut(tc.setState)
			require.NoError(t, err, "got an unexpected error from SetOptOut")

			got, err := m.IsOptedOut()
			require.NoError(t, err, "got an unexpected error from IsOptedOut after SetOptOut")
			require.Equal(t, tc.setState, got, "IsOptedOut should return the state that was set")
		})
	}
}

func TestSetOptOut_CreatesDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	nestedDir := filepath.Join(dir, "new", "nested", "dir")

	m := systemconfig.New(slog.Default(), nestedDir)
	err := m.SetOptOut(true)
	require.NoError(t, err, "SetOptOut should create the directory and file")

	got, err := m.IsOptedOut()
	require.NoError(t, err)
	require.True(t, got)
}

func TestSetOptOut_FilePermissions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	m := systemconfig.New(slog.Default(), dir)

	err := m.SetOptOut(true)
	require.NoError(t, err)

	if runtime.GOOS != "windows" {
		info, err := os.Stat(fmt.Sprintf("%s/%s", dir, constants.SystemConfigFileName))
		require.NoError(t, err, "failed to stat written system config file")
		require.Equal(t, os.FileMode(0644), info.Mode().Perm(), "system config file should be world-readable")
	}
}
