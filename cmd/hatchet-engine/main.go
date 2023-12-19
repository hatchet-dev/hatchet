package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/hatchet-dev/hatchet/cmd/cmdutils"
	"github.com/hatchet-dev/hatchet/internal/config/loader"
	"github.com/hatchet-dev/hatchet/internal/services/admin"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher"
	"github.com/hatchet-dev/hatchet/internal/services/eventscontroller"
	"github.com/hatchet-dev/hatchet/internal/services/grpc"
	"github.com/hatchet-dev/hatchet/internal/services/ingestor"
	"github.com/hatchet-dev/hatchet/internal/services/jobscontroller"
	"github.com/hatchet-dev/hatchet/internal/services/ticker"
	"github.com/spf13/cobra"
)

var printVersion bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "hatchet-engine",
	Short: "hatchet-engine runs the Hatchet engine.",
	Run: func(cmd *cobra.Command, args []string) {
		if printVersion {
			fmt.Println(Version)
			os.Exit(0)
		}

		cf := &loader.ConfigLoader{}
		interruptChan := cmdutils.InterruptChan()

		startEngineOrDie(cf, interruptChan)
	},
}

// Version will be linked by an ldflag during build
var Version string = "v0.1.0-alpha.0"

func main() {
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

func startEngineOrDie(cf *loader.ConfigLoader, interruptCh <-chan interface{}) {
	sc, err := cf.LoadServerConfig()

	if err != nil {
		panic(err)
	}

	errCh := make(chan error)
	ctx, cancel := cmdutils.InterruptContext(interruptCh)
	wg := sync.WaitGroup{}

	if sc.HasService("grpc") {
		wg.Add(1)

		// create the dispatcher
		d, err := dispatcher.New(
			dispatcher.WithTaskQueue(sc.TaskQueue),
			dispatcher.WithRepository(sc.Repository),
			dispatcher.WithLogger(sc.Logger),
		)

		if err != nil {
			errCh <- err
			return
		}

		go func() {
			defer wg.Done()
			err := d.Start(ctx)

			if err != nil {
				panic(err)
			}
		}()

		// create the event ingestor
		ei, err := ingestor.NewIngestor(
			ingestor.WithEventRepository(
				sc.Repository.Event(),
			),
			ingestor.WithTaskQueue(sc.TaskQueue),
		)

		if err != nil {
			errCh <- err
			return
		}

		adminSvc, err := admin.NewAdminService(
			admin.WithRepository(sc.Repository),
			admin.WithTaskQueue(sc.TaskQueue),
		)

		// create the grpc server
		s, err := grpc.NewServer(
			grpc.WithIngestor(ei),
			grpc.WithDispatcher(d),
			grpc.WithAdmin(adminSvc),
			grpc.WithLogger(sc.Logger),
			grpc.WithTLSConfig(sc.TLSConfig),
		)

		if err != nil {
			errCh <- err
			return
		}

		go func() {
			err = s.Start(ctx)

			if err != nil {
				errCh <- err
				return
			}
		}()
	}

	if sc.HasService("eventscontroller") {
		// create separate events controller process
		go func() {
			ec, err := eventscontroller.New(
				eventscontroller.WithTaskQueue(sc.TaskQueue),
				eventscontroller.WithRepository(sc.Repository),
			)

			if err != nil {
				errCh <- err
				return
			}

			err = ec.Start(ctx)

			if err != nil {
				errCh <- err
			}
		}()
	}

	if sc.HasService("jobscontroller") {
		// create separate jobs controller process
		go func() {
			jc, err := jobscontroller.New(
				jobscontroller.WithTaskQueue(sc.TaskQueue),
				jobscontroller.WithRepository(sc.Repository),
			)

			if err != nil {
				errCh <- err
				return
			}

			err = jc.Start(ctx)

			if err != nil {
				errCh <- err
			}
		}()
	}

	if sc.HasService("ticker") {
		// create a ticker
		go func() {
			t, err := ticker.New(
				ticker.WithTaskQueue(sc.TaskQueue),
				ticker.WithRepository(sc.Repository),
			)

			if err != nil {
				errCh <- err
				return
			}

			err = t.Start(ctx)

			if err != nil {
				errCh <- err
			}
		}()
	}

Loop:
	for {
		select {
		case err := <-errCh:
			fmt.Fprintf(os.Stderr, "%s", err)

			// exit with non-zero exit code
			os.Exit(1)

			break Loop
		case <-interruptCh:
			break Loop
		}
	}

	cancel()

	wg.Wait()

	err = sc.Disconnect()

	if err != nil {
		panic(err)
	}
}
