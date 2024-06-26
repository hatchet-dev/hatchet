package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-api/api"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/config/loader"
)

var printVersion bool
var configDirectory string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "hatchet-api",
	Short: "hatchet-api runs a Hatchet instance.",
	Run: func(cmd *cobra.Command, args []string) {
		if printVersion {
			fmt.Println(Version)
			os.Exit(0)
		}

		cf := loader.NewConfigLoader(configDirectory)
		interruptChan := cmdutils.InterruptChan()

		if err := api.Start(cf, interruptChan, Version); err != nil {
			log.Println("error starting API:", err)
			os.Exit(1)
		}
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
