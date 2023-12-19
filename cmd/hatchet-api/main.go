package main

import (
	"fmt"
	"os"

	"github.com/hatchet-dev/hatchet/api/v1/server/run"
	"github.com/hatchet-dev/hatchet/cmd/cmdutils"
	"github.com/hatchet-dev/hatchet/internal/config/loader"
	"github.com/spf13/cobra"
)

var printVersion bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "hatchet-api",
	Short: "hatchet-api runs a Hatchet instance.",
	Run: func(cmd *cobra.Command, args []string) {
		if printVersion {
			fmt.Println(Version)
			os.Exit(0)
		}

		cf := &loader.ConfigLoader{}
		interruptChan := cmdutils.InterruptChan()

		startServerOrDie(cf, interruptChan)
	},
}

// Version will be linked by an ldflag during build
var Version string = "v0.1.0-alpha.0"

func main() {
	rootCmd.PersistentFlags().BoolVar(
		&printVersion,
		"version",
		false,
		"print version and exit.",
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func startServerOrDie(configLoader *loader.ConfigLoader, interruptCh <-chan interface{}) {
	// init the repository
	cf := &loader.ConfigLoader{}

	sc, err := cf.LoadServerConfig()

	if err != nil {
		panic(err)
	}

	ctx, cancel := cmdutils.InterruptContext(interruptCh)
	defer cancel()

	runner := run.NewAPIServer(sc)

	err = runner.Run(ctx)

	if err != nil {
		panic(err)
	}
}
