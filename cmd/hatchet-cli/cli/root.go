package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
)

// Version will be linked by an ldflag during build
var Version = "v0.1.0-alpha.0"

var home string
var printVersion bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use: "hatchet",
	// TODO: create longer description with some intro commands
	Short: "hatchet is the CLI for Hatchet.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if printVersion {
			fmt.Println(Version)
			os.Exit(0)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
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

	var err error
	home, err = homedir.Dir()

	if err != nil {
		color.New(color.FgRed).Fprintf(os.Stderr, "could not get home directory: %v\n", err)
		os.Exit(1)
	}

	hatchetDir := filepath.Join(home, ".hatchet")

	if _, err := os.Stat(hatchetDir); os.IsNotExist(err) {
		os.Mkdir(hatchetDir, 0700) // nolint: errcheck
	} else if err != nil {
		color.New(color.FgRed).Fprintf(os.Stderr, "could not create hatchet directory: %v\n", err)
		os.Exit(1)
	}
}
