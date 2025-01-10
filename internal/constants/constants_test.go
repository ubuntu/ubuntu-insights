package constants_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/constants"
)

func Test_GetUserConfigDir(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		want string
		mock func() (string, error)
	}{
		"os.UserConfigDir success": {
			want: "abc/def" + string(os.PathSeparator) + constants.DefaultAppFolder,
			mock: func() (string, error) {
				return "abc/def", nil
			},
		},
		"os.UserConfigDir error": {
			want: string(os.PathSeparator) + constants.DefaultAppFolder,
			mock: func() (string, error) {
				return "", fmt.Errorf("error")
			},
		},
		"os.UserConfigDir error 2": {
			want: string(os.PathSeparator) + constants.DefaultAppFolder,
			mock: func() (string, error) {
				return "abc", fmt.Errorf("error")
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

func Test_userCacheDir(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		want string
		mock func() (string, error)
	}{
		"os.UserCacheDir success": {
			want: "def/abc" + string(os.PathSeparator) + constants.DefaultAppFolder,
			mock: func() (string, error) {
				return "def/abc", nil
			},
		},
		"os.UserCacheDir error": {
			want: string(os.PathSeparator) + constants.DefaultAppFolder,
			mock: func() (string, error) {
				return "", fmt.Errorf("error")
			},
		},
		"os.UserCacheDir error 2": {
			want: string(os.PathSeparator) + constants.DefaultAppFolder,
			mock: func() (string, error) {
				return "abc", fmt.Errorf("error")
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
