package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-engine/engine"
	"github.com/hatchet-dev/hatchet/internal/config/loader"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
)

var printVersion bool
var configDirectory string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "hatchet-engine",
	Short: "hatchet-engine runs the Hatchet engine.",
	Run: func(cmd *cobra.Command, args []string) {
		if printVersion {
			fmt.Println(Version)
			os.Exit(0)
		}

		cf := loader.NewConfigLoader(configDirectory)
		context, cancel := cmdutils.NewInterruptContext()
		defer cancel()

		engine.StartEngineOrDie(cf, context)
	},
}

// Version will be linked by an ldflag during build
var Version = "v0.1.0-alpha.0"

func main() {
	rootCmd.PersistentFlags().BoolVar(
		&printVersion,
		"version",
		false,
		"print version and exit.",
	)

	rootCmd.PersistentFlags().StringVar(
		&configDirectory,
		"config",
		"",
		"The path the config folder.",
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
