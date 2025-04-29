// Package daemon provides the web service daemon for Ubuntu Insights.
package daemon

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"runtime"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ubuntu/ubuntu-insights/internal/cli"
	"github.com/ubuntu/ubuntu-insights/internal/constants"
	"github.com/ubuntu/ubuntu-insights/internal/server/shared/config"
	"github.com/ubuntu/ubuntu-insights/internal/server/webservice"
)

// App represents the application.
type App struct {
	cmd    *cobra.Command
	viper  *viper.Viper
	config appConfig

	daemon *webservice.Server

	ready chan struct{}
}

// appConfig holds the configuration for the application.
type appConfig struct {
	Verbosity int
	Daemon    webservice.StaticConfig
}

// New creates a new App instance with default values.
func New() (*App, error) {
	a := App{ready: make(chan struct{})}

	a.cmd = &cobra.Command{
		Use:           constants.WebServiceCmdName,
		Short:         "Ubuntu Insights web service",
		Long:          "Ubuntu Insights web service used for accepting HTTP requests with insights reports from clients.",
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Command parsing has been successful. Returns to not print usage anymore.
			a.cmd.SilenceUsage = true
			cli.SetVerbosity(a.config.Verbosity) // Set verbosity before loading config
			if err := cli.InitViperConfig(constants.WebServiceCmdName, a.cmd, a.viper); err != nil {
				return err
			}
			if err := a.viper.Unmarshal(&a.config); err != nil {
				return fmt.Errorf("unable to strictly decode configuration into struct: %w", err)
			}
			slog.Info("got app config", "config", a.config)

			cli.SetVerbosity(a.config.Verbosity)
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			a.cmd.SilenceUsage = true

			return a.run()
		},
	}
	a.viper = viper.New()

	if err := installRootCmd(&a); err != nil {
		return nil, err
	}
	cli.InstallConfigFlag(a.cmd)

	if err := a.viper.BindPFlags(a.cmd.PersistentFlags()); err != nil {
		return nil, err
	}

	a.installVersion()

	return &a, nil
}

func installRootCmd(app *App) error {
	cmd := app.cmd

	defaultConf := webservice.StaticConfig{
		ConfigPath:     "config.json",
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   10 * time.Second,
		RequestTimeout: 3 * time.Second,
		MaxHeaderBytes: 1 << 13, // 8 KB
		MaxUploadBytes: 1 << 17, // 128 KB

		RateLimitPS: 0.1,
		BurstLimit:  3,

		ListenPort: 8080,
	}

	cmd.PersistentFlags().CountVarP(&app.config.Verbosity, "verbose", "v", "issue INFO (-v), DEBUG (-vv)")

	// Daemon flags
	cmd.PersistentFlags().StringVar(&app.config.Daemon.ConfigPath, "daemon-config", defaultConf.ConfigPath, "Path to the configuration file")
	cmd.PersistentFlags().DurationVar(&app.config.Daemon.ReadTimeout, "read-timeout", defaultConf.ReadTimeout, "Read timeout for HTTP server")
	cmd.PersistentFlags().DurationVar(&app.config.Daemon.WriteTimeout, "write-timeout", defaultConf.WriteTimeout, "Write timeout for HTTP server")
	cmd.PersistentFlags().DurationVar(&app.config.Daemon.RequestTimeout, "request-timeout", defaultConf.RequestTimeout, "Request timeout for HTTP server")
	cmd.PersistentFlags().IntVar(&app.config.Daemon.MaxHeaderBytes, "max-header-bytes", defaultConf.MaxHeaderBytes, "Maximum header bytes for HTTP server")
	cmd.PersistentFlags().IntVar(&app.config.Daemon.MaxUploadBytes, "max-upload-bytes", defaultConf.MaxUploadBytes, "Maximum upload bytes for HTTP server")

	cmd.PersistentFlags().Float64Var(&app.config.Daemon.RateLimitPS, "rate-limit-ps", defaultConf.RateLimitPS, "Rate limit in packets per second")
	cmd.PersistentFlags().IntVar(&app.config.Daemon.BurstLimit, "burst-limit", defaultConf.BurstLimit, "Burst limit for rate limiting")

	cmd.PersistentFlags().StringVar(&app.config.Daemon.ListenHost, "listen-host", defaultConf.ListenHost, "Host to listen on")
	cmd.PersistentFlags().IntVar(&app.config.Daemon.ListenPort, "listen-port", defaultConf.ListenPort, "Port to listen on")

	err := cmd.MarkPersistentFlagFilename("daemon-config")
	if err != nil {
		return fmt.Errorf("failed to mark daemon-config flag as filename: %w", err)
	}

	return nil
}

// Run executes the command and associated process, returning an error if any.
func (a App) Run() error {
	return a.cmd.Execute()
}

// UsageError returns if the error is a command parsing or runtime one.
func (a App) UsageError() bool {
	return !a.cmd.SilenceUsage
}

// Hup prints all goroutine stack traces and return false to signal you shouldn't quit.
func (a App) Hup() (shouldQuit bool) {
	buf := make([]byte, 1<<16)
	runtime.Stack(buf, true)
	fmt.Printf("%s", buf)
	return false
}

// Quit gracefully shuts down the daemon.
func (a *App) Quit() {
	a.WaitReady()
	if a.daemon != nil {
		a.daemon.Quit(false)
	}
}

// WaitReady waits for the daemon to be ready.
func (a *App) WaitReady() {
	<-a.ready
}

// RootCmd returns the root command.
func (a App) RootCmd() cobra.Command {
	return *a.cmd
}

func (a *App) run() (err error) {
	a.config.Daemon.ConfigPath, err = filepath.Abs(a.config.Daemon.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for config file: %v", err)
	}
	dConf := a.config.Daemon
	cm := config.New(dConf.ConfigPath)
	a.daemon, err = dConf.New(context.Background(), cm)
	close(a.ready)
	if err != nil {
		return fmt.Errorf("failed to create server: %v", err)
	}

	return a.daemon.Run()
}
