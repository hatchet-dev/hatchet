package engine

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/internal/config/loader"
	"github.com/hatchet-dev/hatchet/internal/services/admin"
	"github.com/hatchet-dev/hatchet/internal/services/controllers/events"
	"github.com/hatchet-dev/hatchet/internal/services/controllers/jobs"
	"github.com/hatchet-dev/hatchet/internal/services/controllers/workflows"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher"
	"github.com/hatchet-dev/hatchet/internal/services/grpc"
	"github.com/hatchet-dev/hatchet/internal/services/health"
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

	shutdown, err := telemetry.InitTracer(&telemetry.TracerOpts{
		ServiceName:  sc.OpenTelemetry.ServiceName,
		CollectorURL: sc.OpenTelemetry.CollectorURL,
	})
	if err != nil {
		return fmt.Errorf("could not initialize tracer: %w", err)
	}

	var teardown []Teardown

	var h *health.Health
	healthProbes := sc.HasService("health")
	if healthProbes {
		h = health.New(sc.Repository, sc.TaskQueue)
		cleanup := h.Start()
		teardown = append(teardown, Teardown{
			name: "health",
			fn:   cleanup,
		})
	}

	if sc.HasService("ticker") {
		t, err := ticker.New(
			ticker.WithTaskQueue(sc.TaskQueue),
			ticker.WithRepository(sc.Repository),
			ticker.WithLogger(sc.Logger),
		)

		if err != nil {
			return fmt.Errorf("could not create ticker: %w", err)
		}

		cleanup, err := t.Start()
		if err != nil {
			return fmt.Errorf("could not start ticker: %w", err)
		}
		teardown = append(teardown, Teardown{
			name: "ticker",
			fn:   cleanup,
		})
	}

	if sc.HasService("eventscontroller") {
		ec, err := events.New(
			events.WithTaskQueue(sc.TaskQueue),
			events.WithRepository(sc.Repository),
			events.WithLogger(sc.Logger),
		)
		if err != nil {
			return fmt.Errorf("could not create events controller: %w", err)
		}

		cleanup, err := ec.Start()
		if err != nil {
			return fmt.Errorf("could not start events controller: %w", err)
		}
		teardown = append(teardown, Teardown{
			name: "events controller",
			fn:   cleanup,
		})
	}

	if sc.HasService("jobscontroller") {
		jc, err := jobs.New(
			jobs.WithTaskQueue(sc.TaskQueue),
			jobs.WithRepository(sc.Repository),
			jobs.WithLogger(sc.Logger),
		)

		if err != nil {
			return fmt.Errorf("could not create jobs controller: %w", err)
		}

		cleanup, err := jc.Start()
		if err != nil {
			return fmt.Errorf("could not start jobs controller: %w", err)
		}
		teardown = append(teardown, Teardown{
			name: "jobs controller",
			fn:   cleanup,
		})
	}

	if sc.HasService("workflowscontroller") {
		wc, err := workflows.New(
			workflows.WithTaskQueue(sc.TaskQueue),
			workflows.WithRepository(sc.Repository),
			workflows.WithLogger(sc.Logger),
		)
		if err != nil {
			return fmt.Errorf("could not create workflows controller: %w", err)
		}

		cleanup, err := wc.Start()
		if err != nil {
			return fmt.Errorf("could not start workflows controller: %w", err)
		}
		teardown = append(teardown, Teardown{
			name: "workflows controller",
			fn:   cleanup,
		})
	}

	if sc.HasService("heartbeater") {
		h, err := heartbeat.New(
			heartbeat.WithTaskQueue(sc.TaskQueue),
			heartbeat.WithRepository(sc.Repository),
			heartbeat.WithLogger(sc.Logger),
		)

		if err != nil {
			return fmt.Errorf("could not create heartbeater: %w", err)
		}

		cleanup, err := h.Start()
		if err != nil {
			return fmt.Errorf("could not start heartbeater: %w", err)
		}
		teardown = append(teardown, Teardown{
			name: "heartbeater",
			fn:   cleanup,
		})
	}

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

		dispatcherCleanup, err := d.Start()
		if err != nil {
			return fmt.Errorf("could not start dispatcher: %w", err)
		}

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

		grpcServerCleanup, err := s.Start()
		if err != nil {
			return fmt.Errorf("could not start grpc server: %w", err)
		}

		cleanup := func() error {
			g := new(errgroup.Group)

			g.Go(func() error {
				err := dispatcherCleanup()
				if err != nil {
					return fmt.Errorf("failed to cleanup dispatcher: %w", err)
				}
				return nil
			})

			g.Go(func() error {
				err := grpcServerCleanup()
				if err != nil {
					return fmt.Errorf("failed to cleanup GRPC server: %w", err)
				}
				return nil
			})

			if err := g.Wait(); err != nil {
				return fmt.Errorf("could not teardown grpc dispatcher: %w", err)
			}

			return nil
		}

		teardown = append(teardown, Teardown{
			name: "grpc",
			fn:   cleanup,
		})
	}

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
	teardown = append(teardown, Teardown{
		name: "database",
		fn: func() error {
			return sc.Disconnect()
		},
	})

	l.Debug().Msgf("engine has started")

	if healthProbes {
		h.SetReady(true)
	}

	<-ctx.Done()

	if healthProbes {
		h.SetReady(false)
	}

	time.Sleep(sc.Runtime.ShutdownWait)

	l.Debug().Msgf("interrupt received, shutting down")

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

	l.Debug().Msgf("successfully shutdown")

	return nil
}
