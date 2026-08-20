package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-migrate/migrate"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
)

var printVersion bool
var migrateDown string
var upToPenultimate bool
var upToVersion string

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
			if upToPenultimate && upToVersion != "" {
				fmt.Println("cannot use --up-to-penultimate and --up-to together")
				os.Exit(1)
			}

			var opts []migrate.RunMigrationsOpt
			if upToPenultimate {
				opts = append(opts, migrate.WithUpToPenultimate())
			}
			if upToVersion != "" {
				version, err := strconv.ParseInt(upToVersion, 10, 64)
				if err != nil {
					fmt.Printf("invalid --up-to version %q: %v\n", upToVersion, err)
					os.Exit(1)
				}
				opts = append(opts, migrate.WithUpToVersion(version))
			}
			if err := migrate.RunMigrations(ctx, opts...); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
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

	rootCmd.PersistentFlags().BoolVar(
		&upToPenultimate,
		"up-to-penultimate",
		false,
		"migrate up to the second-to-last migration version.",
	)

	rootCmd.PersistentFlags().StringVar(
		&upToVersion,
		"up-to",
		"",
		"migrate up to a specific goose version (e.g., 20260818184000).",
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
