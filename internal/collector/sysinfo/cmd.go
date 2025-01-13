// package sysinfo allows collecting "common" system information for all insight reports.
package sysinfo

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
)

func runCmd(cmd *exec.Cmd) io.Reader {
	pr, pw := io.Pipe()
	cmd.Stdout = pw

	go func() {
		err := cmd.Run()
		if err != nil {
			s := fmt.Sprintf("'%v' returned an error: %s", cmd.Args, err)
			pw.CloseWithError(errors.New(s))
			return
		}
		pw.Close()
	}()

	return pr
}
