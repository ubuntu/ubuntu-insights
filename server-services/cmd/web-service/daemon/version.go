package daemon

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ubuntu/ubuntu-insights/server-services/internal/shared/constants"
)

func (a *App) installVersion() {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Returns the running version of " + constants.WebServiceCmdName + " and exits",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, args []string) error { return getVersion() },
	}
	a.cmd.AddCommand(cmd)
}

// getVersion returns the current service version.
func getVersion() (err error) {
	fmt.Printf("%s\t%s\n", constants.WebServiceCmdName, constants.Version)
	return nil
}
