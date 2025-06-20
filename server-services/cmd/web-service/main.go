// Package main is the entry point for the web service application.
package main

import (
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/ubuntu/ubuntu-insights/server-services/cmd/web-service/daemon"
)

func main() {
	a, err := daemon.New()
	if err != nil {
		os.Exit(1)
	}

	os.Exit(run(a))
}

type app interface {
	Run() error
	UsageError() bool
	Hup() bool
	Quit()
}

func run(a app) int {
	defer installSignalHandler(a)()

	if err := a.Run(); err != nil {
		slog.Error(err.Error())

		if a.UsageError() {
			return 2
		}
		return 1
	}

	return 0
}

func installSignalHandler(a app) func() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			switch v, ok := <-c; v {
			case syscall.SIGINT, syscall.SIGTERM:
				a.Quit()
				return
			case syscall.SIGHUP:
				if a.Hup() {
					a.Quit()
					return
				}
			default:
				// channel was closed: we exited
				if !ok {
					slog.Debug("Signal channel closed")
					return
				}
			}
		}
	}()

	return func() {
		signal.Stop(c)
		close(c)
		wg.Wait()
	}
}
