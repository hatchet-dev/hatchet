package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/internal/config/loader"
	"github.com/hatchet-dev/hatchet/internal/services/admin"
	"github.com/hatchet-dev/hatchet/internal/services/controllers/events"
	"github.com/hatchet-dev/hatchet/internal/services/controllers/jobs"
	"github.com/hatchet-dev/hatchet/internal/services/controllers/workflows"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher"
	"github.com/hatchet-dev/hatchet/internal/services/grpc"
	"github.com/hatchet-dev/hatchet/internal/services/heartbeat"
	"github.com/hatchet-dev/hatchet/internal/services/ingestor"
	"github.com/hatchet-dev/hatchet/internal/services/ticker"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
)

var printVersion bool
var configDirectory string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "hatchet-engine",
	Short: "hatchet-engine runs the Hatchet engine.",
	Run: func(cmd *cobra.Command, args []string) {
		if printVersion {
			fmt.Println(Version)
			os.Exit(0)
		}

		cf := loader.NewConfigLoader(configDirectory)
		interruptChan := cmdutils.InterruptChan()

		startEngineOrDie(cf, interruptChan)
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

func startEngineOrDie(cf *loader.ConfigLoader, interruptCh <-chan interface{}) {
	sc, err := cf.LoadServerConfig()

	if err != nil {
		panic(err)
	}

	errCh := make(chan error)
	ctx, cancel := cmdutils.InterruptContextFromChan(interruptCh)
	wg := sync.WaitGroup{}

	shutdown, err := telemetry.InitTracer(&telemetry.TracerOpts{
		ServiceName:  sc.OpenTelemetry.ServiceName,
		CollectorURL: sc.OpenTelemetry.CollectorURL,
	})

	if err != nil {
		panic(fmt.Sprintf("could not initialize tracer: %s", err))
	}

	defer shutdown(ctx) // nolint: errcheck

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

		if err != nil {
			errCh <- err
			return
		}

		grpcOpts := []grpc.ServerOpt{
			grpc.WithConfig(sc),
			grpc.WithIngestor(ei),
			grpc.WithDispatcher(d),
			grpc.WithAdmin(adminSvc),
			grpc.WithLogger(sc.Logger),
			grpc.WithTLSConfig(sc.TLSConfig),
			grpc.WithPort(sc.Runtime.GRPCPort),
			grpc.WithBindAddress(sc.Runtime.GRPCBindAddress),
		}

		if sc.Runtime.GRPCInsecure {
			grpcOpts = append(grpcOpts, grpc.WithInsecure())
		}

		// create the grpc server
		s, err := grpc.NewServer(
			grpcOpts...,
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
			ec, err := events.New(
				events.WithTaskQueue(sc.TaskQueue),
				events.WithRepository(sc.Repository),
				events.WithLogger(sc.Logger),
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
			jc, err := jobs.New(
				jobs.WithTaskQueue(sc.TaskQueue),
				jobs.WithRepository(sc.Repository),
				jobs.WithLogger(sc.Logger),
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

	if sc.HasService("workflowscontroller") {
		// create separate jobs controller process
		go func() {
			jc, err := workflows.New(
				workflows.WithTaskQueue(sc.TaskQueue),
				workflows.WithRepository(sc.Repository),
				workflows.WithLogger(sc.Logger),
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
				ticker.WithLogger(sc.Logger),
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

	if sc.HasService("heartbeater") {
		go func() {
			h, err := heartbeat.New(
				heartbeat.WithTaskQueue(sc.TaskQueue),
				heartbeat.WithRepository(sc.Repository),
				heartbeat.WithLogger(sc.Logger),
			)

			if err != nil {
				errCh <- err
				return
			}

			err = h.Start(ctx)

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
			os.Exit(1) //nolint:gocritic
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
