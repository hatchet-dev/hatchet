package engine

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

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
)

func StartEngineOrDie(cf *loader.ConfigLoader, ctx context.Context) {
	sc, err := cf.LoadServerConfig()

	if err != nil {
		panic(err)
	}

	errCh := make(chan error)
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
		wg.Add(2)

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

			log.Printf("dispatcher has shutdown") // ✅
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
			defer wg.Done()
			err = s.Start(ctx)

			if err != nil {
				errCh <- err
				return
			}

			log.Printf("grpc server has shutdown")
		}()
	}

	if sc.HasService("eventscontroller") {
		wg.Add(1)
		// create separate events controller process
		go func() {
			defer wg.Done()

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

			log.Printf("events controller has shutdown") // ✅
		}()
	}

	if sc.HasService("jobscontroller") {
		wg.Add(1)

		// create separate jobs controller process
		go func() {
			defer wg.Done()

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

			log.Printf("jobs controller has shutdown") // ✅
		}()
	}

	if sc.HasService("workflowscontroller") {
		wg.Add(1)

		// create separate jobs controller process
		go func() {
			defer wg.Done()

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

			log.Printf("workflows controller has shutdown") // ✅
		}()
	}

	if sc.HasService("ticker") {
		wg.Add(1)

		// create a ticker
		go func() {
			defer wg.Done()

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

			log.Printf("ticker has shutdown") // ✅
		}()
	}

	if sc.HasService("heartbeater") {
		wg.Add(1)

		go func() {
			defer wg.Done()

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

			log.Printf("heartbeater has shutdown")
		}()
	}

	log.Printf("engine has started")

Loop:
	for {
		select {
		case err := <-errCh:
			fmt.Fprintf(os.Stderr, "engine error, exitting: %s", err)

			// exit with non-zero exit code
			os.Exit(1) //nolint:gocritic
		case <-ctx.Done():
			log.Printf("interrupt received, shutting down")
			break Loop
		}
	}

	log.Printf("waiting for all services to shutdown...")
	wg.Wait()
	log.Printf("all services have shutdown")

	err = sc.Disconnect()

	if err != nil {
		panic(err)
	}

	log.Printf("successfully shutdown")
}
