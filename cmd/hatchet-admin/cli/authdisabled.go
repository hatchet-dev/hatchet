package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/pkg/authmode"
)

var authDisabledCmd = &cobra.Command{
	Use:    "authdisabled",
	Short:  "exit 0 if this is an authdisabled build, 1 otherwise",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		if !authmode.Disabled {
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(authDisabledCmd)
}
