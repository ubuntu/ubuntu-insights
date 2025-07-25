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
	"github.com/ubuntu/ubuntu-insights/common/cli"
	"github.com/ubuntu/ubuntu-insights/server/internal/common/config"
	"github.com/ubuntu/ubuntu-insights/server/internal/common/constants"
	"github.com/ubuntu/ubuntu-insights/server/internal/webservice"
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
	a.cmd.CompletionOptions.HiddenDefaultCmd = true

	installRootCmd(&a)
	cli.InstallConfigFlag(a.cmd)

	if err := a.viper.BindPFlags(a.cmd.PersistentFlags()); err != nil {
		return nil, err
	}

	a.installVersion()

	return &a, nil
}

func installRootCmd(app *App) {
	cmd := app.cmd

	defaultConf := webservice.StaticConfig{
		ConfigPath: "",
		ReportsDir: constants.DefaultServiceReportsDir,

		ReadTimeout:    5 * time.Second,
		WriteTimeout:   10 * time.Second,
		RequestTimeout: 3 * time.Second,
		MaxHeaderBytes: 1 << 13, // 8 KB
		MaxUploadBytes: 1 << 17, // 128 KB

		ListenPort:  8080,
		MetricsPort: 2112,
	}

	cmd.PersistentFlags().CountVarP(&app.config.Verbosity, "verbose", "v", "issue INFO (-v), DEBUG (-vv)")

	// Daemon flags
	cmd.Flags().StringVar(&app.config.Daemon.ConfigPath, "daemon-config", defaultConf.ConfigPath, "path to the configuration file")
	cmd.Flags().StringVar(&app.config.Daemon.ReportsDir, "reports-dir", defaultConf.ReportsDir, "directory to store reports in")

	cmd.Flags().DurationVar(&app.config.Daemon.ReadTimeout, "read-timeout", defaultConf.ReadTimeout, "read timeout for HTTP server")
	cmd.Flags().DurationVar(&app.config.Daemon.WriteTimeout, "write-timeout", defaultConf.WriteTimeout, "write timeout for HTTP server")
	cmd.Flags().DurationVar(&app.config.Daemon.RequestTimeout, "request-timeout", defaultConf.RequestTimeout, "request timeout for HTTP server")
	cmd.Flags().IntVar(&app.config.Daemon.MaxHeaderBytes, "max-header-bytes", defaultConf.MaxHeaderBytes, "maximum header bytes for HTTP server")
	cmd.Flags().IntVar(&app.config.Daemon.MaxUploadBytes, "max-upload-bytes", defaultConf.MaxUploadBytes, "maximum upload bytes for HTTP server")

	cmd.Flags().StringVar(&app.config.Daemon.ListenHost, "listen-host", defaultConf.ListenHost, "host to listen on")
	cmd.Flags().IntVar(&app.config.Daemon.ListenPort, "listen-port", defaultConf.ListenPort, "port to listen on")

	cmd.Flags().StringVar(&app.config.Daemon.MetricsHost, "metrics-host", defaultConf.MetricsHost, "host for the metrics endpoint")
	cmd.Flags().IntVar(&app.config.Daemon.MetricsPort, "metrics-port", defaultConf.MetricsPort, "port for the metrics endpoint")

	err := cmd.MarkFlagFilename("daemon-config")
	if err != nil {
		// This should never happen.
		panic(fmt.Sprintf("failed to mark daemon-config flag as filename: %v", err))
	}

	err = cmd.MarkFlagDirname("reports-dir")
	if err != nil {
		// This should never happen.
		panic(fmt.Sprintf("failed to mark reports-dir flag as required: %v", err))
	}
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
	a.daemon, err = webservice.New(context.Background(), cm, dConf)
	close(a.ready)
	if err != nil {
		return fmt.Errorf("failed to create server: %v", err)
	}

	return a.daemon.Run()
}
