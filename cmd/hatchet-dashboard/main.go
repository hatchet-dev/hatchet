package main

import (
	"fmt"
	"log"

	"os"

	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-api/api"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-engine/engine"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/config/loader"
)

var printVersion bool
var configDirectory string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "hatchet-dashboard",
	Short: "hatchet-dashboard runs a Hatchet instance with  API and engine served on the same instance.",
	Run: func(cmd *cobra.Command, args []string) {
		if printVersion {
			fmt.Println(Version)
			os.Exit(0)
		}

		cf := loader.NewConfigLoader(configDirectory)
		interruptChan := cmdutils.InterruptChan()

		if err := start(cf, interruptChan, Version); err != nil {
			log.Println("error starting API:", err)
			os.Exit(1)
		}
	},
}

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

// runs api and engine in the same process.
func start(cf *loader.ConfigLoader, interruptCh <-chan interface{}, version string) error {

	_, msgQueueKindSet := os.LookupEnv("SERVER_MSGQUEUE_KIND")
	_, msgQueueRabbitMQURLSet := os.LookupEnv("SERVER_MSGQUEUE_RABBITMQ_URL")
	// for legacy reasons let us also check for these two variables
	_, taskQueueKindSet := os.LookupEnv("SERVER_TASKQUEUE_KIND")
	_, taskQueueRabbitMQURLSet := os.LookupEnv("SERVER_TASKQUEUE_RABBITMQ_URL")

	if !msgQueueKindSet && !msgQueueRabbitMQURLSet && !taskQueueKindSet && !taskQueueRabbitMQURLSet {
		err := os.Setenv("SERVER_MSGQUEUE_KIND", "postgres")

		if err != nil {
			return fmt.Errorf("error setting SERVER_MSGQUEUE_KIND to postgres: %w", err)
		}
	}

	// api process
	go func() {
		if err := api.Start(cf, interruptCh, version); err != nil {
			log.Printf("api failure: %s", err.Error())
			os.Exit(1)
		}
	}()

	ctx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	go func() {
		if err := engine.Run(ctx, cf, version); err != nil {
			log.Printf("engine failure: %s", err.Error())
			os.Exit(1)
		}
	}()

	<-interruptCh

	return nil
}
