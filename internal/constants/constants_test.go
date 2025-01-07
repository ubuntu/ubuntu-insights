package constants

import (
	"fmt"
	"testing"
)

func Test_userConfigDir(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		want string
		mock func() (string, error)
	}{
		{
			name: "os.UserConfigDir success",
			want: "abc/def",
			mock: func() (string, error) {
				return "abc/def", nil
			},
		},
		{
			name: "os.UserConfigDir error",
			want: "",
			mock: func() (string, error) {
				return "", fmt.Errorf("error")
			},
		},
		{
			name: "os.UserConfigDir error 2",
			want: "",
			mock: func() (string, error) {
				return "abc", fmt.Errorf("error")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := userConfigDir(tt.mock); got != tt.want {
				t.Errorf("userConfigDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_userCacheDir(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		want string
		mock func() (string, error)
	}{
		{
			name: "os.UserCacheDir success",
			want: "def/abc",
			mock: func() (string, error) {
				return "def/abc", nil
			},
		},
		{
			name: "os.UserCacheDir error",
			want: "",
			mock: func() (string, error) {
				return "", fmt.Errorf("error")
			},
		},
		{
			name: "os.UserCacheDir error 2",
			want: "",
			mock: func() (string, error) {
				return "abc", fmt.Errorf("error")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := userCacheDir(tt.mock); got != tt.want {
				t.Errorf("userCacheDir() = %v, want %v", got, tt.want)
			}
		})
	}
}
