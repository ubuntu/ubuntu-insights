package constants_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/constants"
)

//nolint:dupl //Tests for GetDefaultConfigPath is very similar to GetDefaultCachePath.
func Test_GetDefaultConfigPath(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		want string
		mock func() (string, error)
	}{
		"os.UserConfigDir success": {
			want: filepath.Join("abc", "def", constants.DefaultAppFolder),
			mock: func() (string, error) {
				return "abc/def", nil
			},
		},
		"os.UserConfigDir error": {
			want: constants.DefaultAppFolder,
			mock: func() (string, error) {
				return "", fmt.Errorf("os.UserCacheDir error")
			},
		},
		"os.UserConfigDir error 2": {
			want: constants.DefaultAppFolder,
			mock: func() (string, error) {
				return filepath.Join("abc", "def"), fmt.Errorf("os.UserCacheDir error")
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			opts := []constants.Option{constants.WithBaseDir(tt.mock)}
			require.Equal(t, tt.want, constants.GetDefaultConfigPath(opts...))
		})
	}
}

//nolint:dupl //Tests for GetDefaultConfigPath is very similar to GetDefaultCachePath.
func Test_GetDefaultCachePath(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		want string
		mock func() (string, error)
	}{
		"os.UserCacheDir success": {
			want: filepath.Join("abc", "def", constants.DefaultAppFolder),
			mock: func() (string, error) {
				return filepath.Join("abc", "def"), nil
			},
		},
		"os.UserCacheDir error": {
			want: constants.DefaultAppFolder,
			mock: func() (string, error) {
				return "", fmt.Errorf("os.UserCacheDir error")
			},
		},
		"os.UserCacheDir error with return": {
			want: constants.DefaultAppFolder,
			mock: func() (string, error) {
				return "return", fmt.Errorf("os.UserCacheDir error")
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			opts := []constants.Option{constants.WithBaseDir(tt.mock)}
			require.Equal(t, tt.want, constants.GetDefaultCachePath(opts...))
		})
	}
}
