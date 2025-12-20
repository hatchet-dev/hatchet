package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version will be linked by an ldflag during build
var Version = "v0.1.0-alpha.0"

var printVersion bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "hatchet",
	Short: "hatchet is the client CLI for Hatchet.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if printVersion {
			fmt.Println(Version)
			os.Exit(0)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(
		&printVersion,
		"version",
		false,
		"The version of the hatchet cli.",
	)
}
