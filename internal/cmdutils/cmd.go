// Package cmdutils provides utility functions for running commands.
package cmdutils

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"time"
)

// Run executes the command specified by cmd with arguments args using the provided context.
// Returns stdout and stderr output and error code.
func Run(ctx context.Context, cmd string, args ...string) (stdout, stderr *bytes.Buffer, err error) {
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}

	c := exec.CommandContext(ctx, cmd, args...)
	c.Stdout = stdout
	c.Stderr = stderr
	c.Env = append(c.Env, "LANG=C")
	c.Env = append(c.Env, os.Environ()...)
	err = c.Run()

	return stdout, stderr, err
}

// RunWithTimeout calls Run but a timeout is added to the provided context.
func RunWithTimeout(ctx context.Context, timeout time.Duration, cmd string, args ...string) (stdout, stderr *bytes.Buffer, err error) {
	c, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return Run(c, cmd, args...)
}
