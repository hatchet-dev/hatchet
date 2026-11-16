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
	waitForTelemetry := telemetry.Report(Version)
	defer waitForTelemetry()

	updateNotice := startUpdateCheck()

	// Resolve the target command up front: several subcommands replace the
	// inherited PersistentPreRun, so a hook there would not run for them.
	if target, _, err := rootCmd.Find(os.Args[1:]); err == nil {
		migratePlaintextTokens(target)
	}

	cmd, err := rootCmd.ExecuteC()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if isQuietCommand(cmd) {
		return
	}

	if msg := updateNotice(); msg != "" {
		fmt.Fprintln(os.Stderr, msg)
	}
}

// quietCommands never receive out-of-band output such as the update notice or
// the token migration note: their output is consumed by a shell or they are
// the command the notice would point to.
var quietCommands = map[string]bool{
	"upgrade":                       true,
	"version":                       true,
	"help":                          true,
	"env":                           true,
	"completion":                    true,
	cobra.ShellCompRequestCmd:       true,
	cobra.ShellCompNoDescRequestCmd: true,
}

func isQuietCommand(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if quietCommands[c.Name()] {
			return true
		}
	}
	return false
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
