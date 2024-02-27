package main

import (
	"fmt"
	"os"
	"sync"
	"time"

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

		startServerOrDie(cf, interruptChan)
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

func startServerOrDie(cf *loader.ConfigLoader, interruptCh <-chan interface{}) {
	ctx, cancel := cmdutils.InterruptContextFromChan(interruptCh)
	defer cancel()

	// init the repository
	cleanup, sc, err := cf.LoadServerConfig()
	defer func() {
		if err := cleanup(); err != nil {
			panic(fmt.Errorf("could not cleanup server config: %v", err))
		}
	}()

	if err != nil {
		panic(err)
	}

	errCh := make(chan error)

	wg := sync.WaitGroup{}

	if sc.InternalClient != nil {
		wg.Add(1)

		w, err := worker.NewWorker(
			worker.WithRepository(sc.Repository),
			worker.WithClient(sc.InternalClient),
			worker.WithVCSProviders(sc.VCSProviders),
		)

		if err != nil {
			panic(err)
		}

		go func() {
			defer wg.Done()

			time.Sleep(5 * time.Second)

			err := w.Start(ctx)

			if err != nil {
				errCh <- err
				return
			}
		}()
	}

	wg.Add(1)

	go func() {
		defer wg.Done()

		runner := run.NewAPIServer(sc)

		err = runner.Run(ctx)

		if err != nil {
			errCh <- err
			return
		}
	}()

Loop:
	for {
		select {
		case err := <-errCh:
			fmt.Fprintf(os.Stderr, "%s", err)

			// exit with non-zero exit code
			os.Exit(1) //nolint:gocritic
		case <-interruptCh:
			break Loop
		}
	}

	cancel()

	// TODO: should wait with a timeout
	// wg.Wait()
}
