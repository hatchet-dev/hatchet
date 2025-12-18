package cli

import (
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/docker"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Stops and starts the Hatchet server locally using Docker", // TODO: improve description
	Run: func(cmd *cobra.Command, args []string) {
		dockerDriver, err := docker.NewDockerDriver(cmd.Context())

		if err != nil {
			color.New(color.FgRed).Fprintf(os.Stderr, "could not initialize docker driver: %v\n", err)
			os.Exit(1)
		}

		err = dockerDriver.RunPostgresContainer(cmd.Context())

		if err != nil {
			color.New(color.FgRed).Fprintf(os.Stderr, "could not start postgres container: %v\n", err)
			os.Exit(1)
		}

		err = dockerDriver.RunHatchetLite(cmd.Context())

		if err != nil {
			color.New(color.FgRed).Fprintf(os.Stderr, "could not start hatchet container: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
}
