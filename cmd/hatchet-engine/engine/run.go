package engine

import (
	"context"
	"fmt"
	"log"
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

type Teardown struct {
	name string
	fn   func() error
}

func Run(ctx context.Context, cf *loader.ConfigLoader) error {
	sc, err := cf.LoadServerConfig()

	if err != nil {
		return fmt.Errorf("could not load server config: %w", err)
	}

	errCh := make(chan error)
	wg := sync.WaitGroup{}

	shutdown, err := telemetry.InitTracer(&telemetry.TracerOpts{
		ServiceName:  sc.OpenTelemetry.ServiceName,
		CollectorURL: sc.OpenTelemetry.CollectorURL,
	})

	if err != nil {
		return fmt.Errorf("could not initialize tracer: %w", err)
	}

	var teardown []Teardown

	teardown = append(teardown, Teardown{
		name: "telemetry",
		fn: func() error {
			return shutdown(ctx)
		},
	})

	if sc.HasService("grpc") {
		// create the dispatcher
		d, err := dispatcher.New(
			dispatcher.WithTaskQueue(sc.TaskQueue),
			dispatcher.WithRepository(sc.Repository),
			dispatcher.WithLogger(sc.Logger),
		)

		if err != nil {
			return fmt.Errorf("could not create dispatcher: %w", err)
		}

		go func() {
			cleanup, err := d.Start()
			if err != nil {
				panic(err)
			}

			teardown = append([]Teardown{{
				name: "grpc dispatcher",
				fn:   cleanup,
			}}, teardown...)
		}()

		// create the event ingestor
		ei, err := ingestor.NewIngestor(
			ingestor.WithEventRepository(
				sc.Repository.Event(),
			),
			ingestor.WithTaskQueue(sc.TaskQueue),
		)

		if err != nil {
			return fmt.Errorf("could not create ingestor: %w", err)
		}

		adminSvc, err := admin.NewAdminService(
			admin.WithRepository(sc.Repository),
			admin.WithTaskQueue(sc.TaskQueue),
		)

		if err != nil {
			return fmt.Errorf("could not create admin service: %w", err)
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
			return fmt.Errorf("could not create grpc server: %w", err)
		}

		go func() {
			cleanup, err := s.Start()

			if err != nil {
				panic(err)
			}

			teardown = append([]Teardown{{
				name: "grpc server",
				fn:   cleanup,
			}}, teardown...)
		}()
	}

	if sc.HasService("eventscontroller") {
		wg.Add(1)

		ec, err := events.New(
			events.WithTaskQueue(sc.TaskQueue),
			events.WithRepository(sc.Repository),
			events.WithLogger(sc.Logger),
		)

		if err != nil {
			return fmt.Errorf("could not create events controller: %w", err)
		}

		// create separate events controller process
		go func() {
			defer wg.Done()

			err = ec.Start(ctx)

			if err != nil {
				errCh <- err
			}

			log.Printf("events controller has shutdown") // ✅
		}()
	}

	if sc.HasService("jobscontroller") {
		wg.Add(1)

		jc, err := jobs.New(
			jobs.WithTaskQueue(sc.TaskQueue),
			jobs.WithRepository(sc.Repository),
			jobs.WithLogger(sc.Logger),
		)

		if err != nil {
			return fmt.Errorf("could not create jobs controller: %w", err)
		}

		// create separate jobs controller process
		go func() {
			defer wg.Done()

			err = jc.Start(ctx)

			if err != nil {
				errCh <- err
			}

			log.Printf("jobs controller has shutdown") // ✅
		}()
	}

	if sc.HasService("workflowscontroller") {
		wg.Add(1)

		wc, err := workflows.New(
			workflows.WithTaskQueue(sc.TaskQueue),
			workflows.WithRepository(sc.Repository),
			workflows.WithLogger(sc.Logger),
		)

		if err != nil {
			return fmt.Errorf("could not create workflows controller: %w", err)
		}

		// create separate jobs controller process
		go func() {
			defer wg.Done()

			err = wc.Start(ctx)

			if err != nil {
				errCh <- err
			}

			log.Printf("workflows controller has shutdown") // ✅
		}()
	}

	if sc.HasService("ticker") {
		wg.Add(1)

		t, err := ticker.New(
			ticker.WithTaskQueue(sc.TaskQueue),
			ticker.WithRepository(sc.Repository),
			ticker.WithLogger(sc.Logger),
		)

		if err != nil {
			return fmt.Errorf("could not create ticker: %w", err)
		}

		// create a ticker
		go func() {
			defer wg.Done()

			err = t.Start(ctx)

			if err != nil {
				errCh <- err
			}

			log.Printf("ticker has shutdown") // ✅
		}()
	}

	if sc.HasService("heartbeater") {
		wg.Add(1)

		h, err := heartbeat.New(
			heartbeat.WithTaskQueue(sc.TaskQueue),
			heartbeat.WithRepository(sc.Repository),
			heartbeat.WithLogger(sc.Logger),
		)

		if err != nil {
			return fmt.Errorf("could not create heartbeater: %w", err)
		}

		go func() {
			defer wg.Done()

			err = h.Start(ctx)

			if err != nil {
				errCh <- err
			}

			log.Printf("heartbeater has shutdown") // ✅
		}()
	}

	log.Printf("engine has started")

Loop:
	for {
		select {
		case err := <-errCh:
			return fmt.Errorf("engine error: %w", err)
		case <-ctx.Done():
			log.Printf("interrupt received, shutting down")
			break Loop
		}
	}

	log.Printf("waiting for all services to shutdown...")
	wg.Wait()
	log.Printf("all services have shutdown")

	log.Printf("waiting for all other services to gracefully exit...")
	for i, t := range teardown {
		log.Printf("shutting down %s (%d/%d)", t.name, i+1, len(teardown))
		err := t.fn()

		if err != nil {
			return fmt.Errorf("could not teardown %s: %w", t.name, err)
		}
		log.Printf("successfully shutdown %s (%d/%d)", t.name, i+1, len(teardown))
	}
	log.Printf("all services have successfully gracefully exited")

	err = sc.Disconnect()
	if err != nil {
		return fmt.Errorf("could not disconnect from repository: %w", err)
	}

	log.Printf("successfully shutdown")

	return nil
}
