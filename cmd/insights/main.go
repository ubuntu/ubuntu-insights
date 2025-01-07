package main

import (
	"os"

	"log/slog"

	"github.com/ubuntu/ubuntu-insights/cmd/insights/commands"
	"github.com/ubuntu/ubuntu-insights/internal/constants"
)

func main() {
	opts := &slog.HandlerOptions{
		Level: constants.DefaultLogLevel,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, opts))
	slog.SetDefault(logger)

	a, err := commands.New()
	if err != nil {
		os.Exit(1)
	}

	os.Exit(run(a))
}

type app interface {
	Run() error
	UsageError() bool
	Quit()
}

func run(a app) int {
	if er := a.Run(); er != nil {
		slog.Error(er.Error())

		if a.UsageError() {
			return 2
		}
		return 1
	}

	return 0
}
