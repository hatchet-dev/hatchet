package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-migrate/migrate"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
)

var printVersion bool
var migrateDown string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "hatchet-migrate",
	Short: "hatchet-migrate runs database migrations for Hatchet.",
	Run: func(cmd *cobra.Command, args []string) {
		if printVersion {
			fmt.Println(Version)

			os.Exit(0)
		}

		ctx, cancel := cmdutils.NewInterruptContext()
		defer cancel()

		if migrateDown != "" {
			migrate.RunDownMigration(ctx, migrateDown)
		} else {
			migrate.RunMigrations(ctx)
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
		&migrateDown,
		"down",
		"",
		"migrate down to a specific version (e.g., 20240115180414).",
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
