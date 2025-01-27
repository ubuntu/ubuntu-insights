// Package sysinfo allows collecting "common" system information for all insight reports.
package sysinfo

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"time"
)

func runCmd(ctx context.Context, cmd string, args ...string) (stdout, stderr *bytes.Buffer, err error) {
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

func runCmdWithTimeout(ctx context.Context, timeout time.Duration, cmd string, args ...string) (*bytes.Buffer, *bytes.Buffer, error) {
	c, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return runCmd(c, cmd, args...)
}
