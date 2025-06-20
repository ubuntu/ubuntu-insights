// Package cli provides utility functions for command line interface applications.
package cli

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// InitViperConfig initializes the Viper configuration for a command.
func InitViperConfig(cmdName string, cmd *cobra.Command, vip *viper.Viper) error {
	if v, err := cmd.Flags().GetString("config"); err == nil && v != "" {
		vip.SetConfigFile(v)
	} else {
		vip.SetConfigName(cmdName)
		vip.AddConfigPath(".")

		if runtime.GOOS == "windows" {
			vip.AddConfigPath("C:\\ProgramData\\" + cmdName)
		} else {
			vip.AddConfigPath("/etc/" + cmdName)
			vip.AddConfigPath("/usr/local/etc/" + cmdName)
		}

		if binPath, err := os.Executable(); err != nil {
			slog.Warn("Failed to get current executable path, not adding it as a config dir", "error", err)
		} else {
			vip.AddConfigPath(filepath.Dir(binPath))
		}
	}
	if err := vip.ReadInConfig(); err != nil {
		var e viper.ConfigFileNotFoundError
		if errors.As(err, &e) {
			slog.Info("No configuration file.\nWe will only use the defaults, env variables or flags.", "error", e)
		} else {
			return fmt.Errorf("invalid configuration file: %w", err)
		}
	} else {
		slog.Info("Using configuration file", "file", vip.ConfigFileUsed())
	}

	// Handle environment.
	vip.SetEnvPrefix(cmdName)
	vip.AutomaticEnv()

	// Visit manually env to bind every possibly related environment variable to be able to unmarshal
	// those into a struct.
	// More context on https://github.com/spf13/viper/pull/1429.
	prefix := strings.ToUpper(strings.ReplaceAll(cmdName, "-", "_")) + "_"
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, prefix) {
			continue
		}

		s := strings.Split(e, "=")
		k := strings.ReplaceAll(strings.TrimPrefix(s[0], prefix), "_", ".")
		if err := vip.BindEnv(k, s[0]); err != nil {
			return fmt.Errorf("could not bind environment variable: %w", err)
		}
	}

	return nil
}

// InstallConfigFlag adds a config flag to the command.
func InstallConfigFlag(cmd *cobra.Command) *string {
	return cmd.PersistentFlags().String("config", "", "use a specific configuration file")
}
