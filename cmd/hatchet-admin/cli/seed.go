package cli

import (
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/cmd/internal"
	"github.com/hatchet-dev/hatchet/pkg/config/loader"
)

// seedCmd seeds the database with initial data
var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "seed create initial data in the database.",
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		configLoader := loader.NewConfigLoader(configDirectory)
		err = internal.RunSeed(configLoader)

		if err != nil {
			log.Printf("Fatal: could not run seed command: %v", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(seedCmd)
}
