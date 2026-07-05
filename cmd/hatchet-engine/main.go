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
	"github.com/hatchet-dev/hatchet/pkg/version"
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

		if err := engine.Run(context, cf, Version); err != nil {
			log.Printf("engine failure: %s", err.Error())
			os.Exit(1)
		}
	},
}

// Version is the engine version reported to SDK clients. Release builds override it via
// `-ldflags "-X main.Version=<tag>"`; otherwise it falls back to the canonical pkg/version constant,
// which is the single source of truth (kept in sync with the release tag by the
// check-version-matches-tag CI guard).
var Version = version.Version

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
