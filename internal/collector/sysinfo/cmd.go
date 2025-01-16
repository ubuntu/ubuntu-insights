// Package sysinfo allows collecting "common" system information for all insight reports.
package sysinfo

import (
	"bytes"
	"context"
	"os/exec"
)

func runCmd(ctx context.Context, cmd string, args ...string) (stdout bytes.Buffer, stderr bytes.Buffer, err error) {
	c := exec.CommandContext(ctx, cmd, args...)
	c.Stdout = &stdout
	c.Stderr = &stderr
	err = c.Run()

	return stdout, stderr, err
}
