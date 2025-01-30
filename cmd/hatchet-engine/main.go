package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	"context"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-engine/engine"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/config/loader"

	"net/http"
	_ "net/http/pprof"
)

var printVersion bool
var configDirectory string
var noGracefulShutdown bool

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

		context := context.Background()
		if !noGracefulShutdown {
			ctx, cancel := cmdutils.NewInterruptContext()
			defer cancel()
			context = ctx
		}

		// enable pprof if requested
		if os.Getenv("PPROF_ENABLED") == "true" {
			go func() {
				log.Println(http.ListenAndServe("localhost:6060", nil))
			}()
		}

		if err := engine.Run(context, cf, Version); err != nil {
			log.Printf("engine failure: %s", err.Error())
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

	rootCmd.PersistentFlags().BoolVar(
		&noGracefulShutdown,
		"no-graceful-shutdown",
		false,
		"Whether not to shut down gracefully (useful for nodemon/air).",
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
