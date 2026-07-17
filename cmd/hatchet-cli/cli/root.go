package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/telemetry"
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
	Run: func(cmd *cobra.Command, args []string) {
		err := cmd.Help()

		if err != nil {
			cli.Logger.Fatalf("could not print help: %v", err)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// Fire anonymous usage telemetry in the background; wait for it (bounded)
	// after the command completes so it never delays output. Handled here rather
	// than in a PersistentPreRun so subcommands that override that hook are still
	// counted.
	waitForTelemetry := telemetry.Report(Version)
	defer waitForTelemetry()

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(
		&printVersion,
		"version",
		"v",
		false,
		"The version of the hatchet cli.",
	)
}
