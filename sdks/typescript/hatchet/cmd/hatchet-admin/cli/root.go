package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version will be linked by an ldflag during build
var Version = "v0.1.0-alpha.0"

var printVersion bool

var configDirectory string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "hatchet-admin",
	Short: "hatchet-admin performs administrative tasks for a Hatchet instance.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if printVersion {
			fmt.Println(Version)
			os.Exit(0)
		}

		// var err error

		// configLoader := loader.NewConfigLoader(configDirectory)
		// sc, err = configLoader.LoadServerConfig()

		// if err != nil {
		// 	log.Printf("Fatal: could not load server config: %v\n", err)
		// 	os.Exit(1)
		// }
	},
	Run: func(cmd *cobra.Command, args []string) {
	},
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
