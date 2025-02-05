// Main package for the insights command line tool.
package main

import (
	"log/slog"
	"os"

	"github.com/ubuntu/ubuntu-insights/cmd/insights/commands"
	"github.com/ubuntu/ubuntu-insights/internal/constants"
)

func main() {
	slog.SetLogLoggerLevel(constants.DefaultLogLevel)

	a, err := commands.New()
	if err != nil {
		os.Exit(1)
	}

	os.Exit(run(a))
}

type app interface {
	Run() error
	UsageError() bool
}

func run(a app) int {
	if err := a.Run(); err != nil {
		slog.Error(err.Error())

		if a.UsageError() {
			return 2
		}
		return 1
	}

	return 0
}
