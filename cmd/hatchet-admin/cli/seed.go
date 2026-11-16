package cli

import (
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-admin/cli/seed"
	"github.com/hatchet-dev/hatchet/pkg/config/loader"
	"github.com/hatchet-dev/hatchet/pkg/config/shared"
	"github.com/hatchet-dev/hatchet/pkg/logger"
)

// seedCmd seeds the database with initial data
var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "seed create initial data in the database.",
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		configLoader := loader.NewConfigLoader(configDirectory)
		err = runSeed(configLoader)

		if err != nil {
			log.Printf("Fatal: could not run seed command: %v", err)
			os.Exit(1)
		}
	},
}

var seedCypressCmd = &cobra.Command{
	Use:   "seed-cypress",
	Short: "seed-cypress create initial data in the database for cypress.",
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		configLoader := loader.NewConfigLoader(configDirectory)
		err = runSeedForCypress(configLoader)

		if err != nil {
			log.Printf("Fatal: could not run seed command: %v", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(seedCmd)
	rootCmd.AddCommand(seedCypressCmd)
}

func runSeed(cf *loader.ConfigLoader) error {
	// load the config
	dc, err := cf.InitDataLayer()

	if err != nil {
		panic(err)
	}

	defer dc.Disconnect() // nolint: errcheck

	// the database logger defaults to warn, which would hide what this command
	// did. seeding is the whole point of the command, so run it against a
	// console logger at info instead. In-process callers (embedded mode, the
	// test harness) keep the configured database logger and stay quiet.
	l := logger.NewStdErr(&shared.LoggerConfigFile{Level: "info", Format: "console"}, "seed")
	dc.Logger = &l

	return seed.SeedDatabase(dc)
}

func runSeedForCypress(cf *loader.ConfigLoader) error {
	// load the config
	dc, err := cf.InitDataLayer()

	if err != nil {
		panic(err)
	}

	defer dc.Disconnect() // nolint: errcheck

	return seed.SeedDatabaseForCypress(dc)
}
