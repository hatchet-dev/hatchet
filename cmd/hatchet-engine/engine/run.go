package engine

import (
	"context"
	"fmt"
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
	serverCleanup, sc, err := cf.LoadServerConfig()
	if err != nil {
		return fmt.Errorf("could not load server config: %w", err)
	}
	var l = sc.Logger

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

	teardown = append(teardown, Teardown{
		name: "server",
		fn: func() error {
			return serverCleanup()
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
			l.Debug().Msgf("starting grpc dispatcher")
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
			l.Debug().Msgf("starting grpc server")
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
			l.Debug().Msgf("starting events controller")
			defer wg.Done()

			err := ec.Start(ctx)
			if err != nil {
				errCh <- err
			}

			l.Debug().Msgf("events controller has shutdown")
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
			l.Debug().Msgf("starting jobs controller")
			defer wg.Done()

			err := jc.Start(ctx)
			if err != nil {
				errCh <- err
			}

			l.Debug().Msgf("jobs controller has shutdown")
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
			l.Debug().Msgf("starting workflows controller")
			defer wg.Done()

			err := wc.Start(ctx)
			if err != nil {
				errCh <- err
			}

			l.Debug().Msgf("workflows controller has shutdown")
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
			l.Debug().Msgf("starting ticker")
			defer wg.Done()

			err := t.Start(ctx)
			if err != nil {
				errCh <- err
			}

			l.Debug().Msgf("ticker has shutdown")
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
			l.Debug().Msgf("starting heartbeater")
			defer wg.Done()

			err := h.Start(ctx)
			if err != nil {
				errCh <- err
			}

			l.Debug().Msgf("heartbeater has shutdown")
		}()
	}

	l.Debug().Msgf("engine has started")

Loop:
	for {
		select {
		case err := <-errCh:
			return fmt.Errorf("engine error: %w", err)
		case <-ctx.Done():
			l.Debug().Msgf("interrupt received, shutting down")
			break Loop
		}
	}

	l.Debug().Msgf("waiting for all services to shutdown...")
	wg.Wait()
	l.Debug().Msgf("all services have shutdown")

	l.Debug().Msgf("waiting for all other services to gracefully exit...")
	for i, t := range teardown {
		l.Debug().Msgf("shutting down %s (%d/%d)", t.name, i+1, len(teardown))
		err := t.fn()

		if err != nil {
			return fmt.Errorf("could not teardown %s: %w", t.name, err)
		}
		l.Debug().Msgf("successfully shutdown %s (%d/%d)", t.name, i+1, len(teardown))
	}
	l.Debug().Msgf("all services have successfully gracefully exited")

	err = sc.Disconnect()
	if err != nil {
		return fmt.Errorf("could not disconnect from repository: %w", err)
	}

	l.Debug().Msgf("successfully shutdown")

	return nil
}
