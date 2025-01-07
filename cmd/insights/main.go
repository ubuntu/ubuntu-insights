package main

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/ubuntu/ubuntu-insights/cmd/insights/commands"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
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
		log.Error().Msg(er.Error())

		if a.UsageError() {
			return 2
		}
		return 1
	}

	return 0
}
