package cli

import (
	"fmt"
	"os"

	"github.com/hatchet-dev/hatchet/internal/config/server"
	"github.com/spf13/cobra"
)

// Version will be linked by an ldflag during build
var Version string = "v0.1.0-alpha.0"

var printVersion bool

var sc *server.ServerConfig

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "hatchet-admin",
	Short: "hatchet-admin performs administrative tasks for a Hatchet instance.",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
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
