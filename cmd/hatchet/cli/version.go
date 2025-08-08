package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is the version of Hatchet CLI
// This will be overwritten at build time by goreleaser
var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Hatchet",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
