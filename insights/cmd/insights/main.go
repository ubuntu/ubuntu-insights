// Main package for the insights command line tool.
package main

import (
	"log/slog"
	"os"

	"github.com/ubuntu/ubuntu-insights/insights/cmd/insights/commands"
	"github.com/ubuntu/ubuntu-insights/insights/internal/constants"
)

//go:generate go run ../generate_completion_documentation.go completion ../../generated
//go:generate go run -ldflags=-X=github.com/ubuntu/ubuntu-insights/insights/internal/constants.manGeneration=true ../generate_completion_documentation.go man ../../generated

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
