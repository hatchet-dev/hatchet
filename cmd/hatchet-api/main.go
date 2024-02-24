package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/api/v1/server/run"
	"github.com/hatchet-dev/hatchet/internal/config/loader"
	"github.com/hatchet-dev/hatchet/internal/services/worker"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
)

var printVersion bool
var configDirectory string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "hatchet-api",
	Short: "hatchet-api runs a Hatchet instance.",
	Run: func(cmd *cobra.Command, args []string) {
		if printVersion {
			fmt.Println(Version)
			os.Exit(0)
		}

		cf := loader.NewConfigLoader(configDirectory)
		interruptChan := cmdutils.InterruptChan()

		if err := startAPI(cf, interruptChan); err != nil {
			log.Println("error starting API:", err)
			os.Exit(1)
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

func startAPI(cf *loader.ConfigLoader, interruptCh <-chan interface{}) error {
	// init the repository
	cleanup, sc, err := cf.LoadServerConfig()
	defer func() {
		if err := cleanup(); err != nil {
			panic(fmt.Errorf("could not cleanup server config: %v", err))
		}
	}()

	if err != nil {
		return fmt.Errorf("error loading server config: %w", err)
	}

	var teardowns []func() error

	if sc.InternalClient != nil {
		w, err := worker.NewWorker(
			worker.WithRepository(sc.Repository),
			worker.WithClient(sc.InternalClient),
			worker.WithVCSProviders(sc.VCSProviders),
		)

		if err != nil {
			return fmt.Errorf("error creating worker: %w", err)
		}

		cleanup, err := w.Start()
		if err != nil {
			return fmt.Errorf("error starting worker: %w", err)
		}

		teardowns = append(teardowns, cleanup)
	}

	runner := run.NewAPIServer(sc)

	cleanup, err = runner.Run()
	if err != nil {
		return fmt.Errorf("error starting API server: %w", err)
	}

	teardowns = append(teardowns, cleanup)

	sc.Logger.Debug().Msgf("api started successfully")

	<-interruptCh

	sc.Logger.Debug().Msgf("api is shutting down...")

	for _, teardown := range teardowns {
		if err := teardown(); err != nil {
			return err
		}
	}

	sc.Logger.Debug().Msgf("api successfully shut down")

	return nil
}
