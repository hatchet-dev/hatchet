package engine

import (
	"context"
	"fmt"
	"os"
	"time"

	"golang.org/x/sync/errgroup"

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
	"github.com/hatchet-dev/hatchet/internal/services/webhooks"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/config/loader"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type Teardown struct {
	Name string
	Fn   func() error
}

func init() {
	svcName := os.Getenv("SERVER_OTEL_SERVICE_NAME")
	collectorURL := os.Getenv("SERVER_OTEL_COLLECTOR_URL")

	// we do this to we get the tracer set globally, which is needed by some of the otel
	// integrations for the database before start
	_, err := telemetry.InitTracer(&telemetry.TracerOpts{
		ServiceName:  svcName,
		CollectorURL: collectorURL,
	})

	if err != nil {
		panic(fmt.Errorf("could not initialize tracer: %w", err))
	}
}

func Run(ctx context.Context, cf *loader.ConfigLoader, version string) error {
	serverCleanup, sc, err := cf.LoadServerConfig(version)
	if err != nil {
		return fmt.Errorf("could not load server config: %w", err)
	}

	var l = sc.Logger

	teardown, err := RunWithConfig(ctx, sc)

	if err != nil {
		return fmt.Errorf("could not run with config: %w", err)
	}

	teardown = append(teardown, Teardown{
		Name: "server",
		Fn: func() error {
			return serverCleanup()
		},
	})
	teardown = append(teardown, Teardown{
		Name: "database",
		Fn: func() error {
			return sc.Disconnect()
		},
	})

	time.Sleep(sc.Runtime.ShutdownWait)

	l.Debug().Msgf("interrupt received, shutting down")

	l.Debug().Msgf("waiting for all other services to gracefully exit...")
	for i, t := range teardown {
		l.Debug().Msgf("shutting down %s (%d/%d)", t.Name, i+1, len(teardown))
		err := t.Fn()

		if err != nil {
			return fmt.Errorf("could not teardown %s: %w", t.Name, err)
		}
		l.Debug().Msgf("successfully shutdown %s (%d/%d)", t.Name, i+1, len(teardown))
	}
	l.Debug().Msgf("all services have successfully gracefully exited")

	l.Debug().Msgf("successfully shutdown")

	return nil
}

func RunWithConfig(ctx context.Context, sc *server.ServerConfig) ([]Teardown, error) {
	var l = sc.Logger

	shutdown, err := telemetry.InitTracer(&telemetry.TracerOpts{
		ServiceName:  sc.OpenTelemetry.ServiceName,
		CollectorURL: sc.OpenTelemetry.CollectorURL,
	})
	if err != nil {
		return nil, fmt.Errorf("could not initialize tracer: %w", err)
	}

	p, err := newPartitioner(sc.EngineRepository.Tenant())

	if err != nil {
		return nil, fmt.Errorf("could not create partitioner: %w", err)
	}

	teardown := []Teardown{}

	teardown = append(teardown, Teardown{
		Name: "partitioner",
		Fn:   p.shutdown,
	})

	var h *health.Health
	healthProbes := sc.HasService("health")
	if healthProbes {
		h = health.New(sc.EngineRepository, sc.MessageQueue)
		cleanup, err := h.Start()
		if err != nil {
			return nil, fmt.Errorf("could not start health: %w", err)
		}

		teardown = append(teardown, Teardown{
			Name: "health",
			Fn:   cleanup,
		})
	}

	if sc.HasService("ticker") {
		t, err := ticker.New(
			ticker.WithMessageQueue(sc.MessageQueue),
			ticker.WithRepository(sc.EngineRepository),
			ticker.WithLogger(sc.Logger),
			ticker.WithTenantAlerter(sc.TenantAlerter),
			ticker.WithEntitlementsRepository(sc.EntitlementRepository),
		)

		if err != nil {
			return nil, fmt.Errorf("could not create ticker: %w", err)
		}

		cleanup, err := t.Start()
		if err != nil {
			return nil, fmt.Errorf("could not start ticker: %w", err)
		}
		teardown = append(teardown, Teardown{
			Name: "ticker",
			Fn:   cleanup,
		})
	}

	if sc.HasService("eventscontroller") {
		ec, err := events.New(
			events.WithMessageQueue(sc.MessageQueue),
			events.WithRepository(sc.EngineRepository),
			events.WithLogger(sc.Logger),
			events.WithEntitlementsRepository(sc.EntitlementRepository),
		)
		if err != nil {
			return nil, fmt.Errorf("could not create events controller: %w", err)
		}

		cleanup, err := ec.Start()
		if err != nil {
			return nil, fmt.Errorf("could not start events controller: %w", err)
		}
		teardown = append(teardown, Teardown{
			Name: "events controller",
			Fn:   cleanup,
		})
	}

	if sc.HasService("queue") {
		partitionTeardown, partitionId, err := p.withControllers(ctx)

		if err != nil {
			return nil, fmt.Errorf("could not create rebalance controller partitions job: %w", err)
		}

		teardown = append(teardown, *partitionTeardown)

		jc, err := jobs.New(
			jobs.WithAlerter(sc.Alerter),
			jobs.WithMessageQueue(sc.MessageQueue),
			jobs.WithRepository(sc.EngineRepository),
			jobs.WithLogger(sc.Logger),
			jobs.WithPartitionId(partitionId),
		)

		if err != nil {
			return nil, fmt.Errorf("could not create jobs controller: %w", err)
		}

		cleanupJobs, err := jc.Start()
		if err != nil {
			return nil, fmt.Errorf("could not start jobs controller: %w", err)
		}
		teardown = append(teardown, Teardown{
			Name: "jobs controller",
			Fn:   cleanupJobs,
		})

		wc, err := workflows.New(
			workflows.WithAlerter(sc.Alerter),
			workflows.WithMessageQueue(sc.MessageQueue),
			workflows.WithRepository(sc.EngineRepository),
			workflows.WithLogger(sc.Logger),
			workflows.WithTenantAlerter(sc.TenantAlerter),
			workflows.WithPartitionId(partitionId),
		)
		if err != nil {
			return nil, fmt.Errorf("could not create workflows controller: %w", err)
		}

		cleanupWorkflows, err := wc.Start()
		if err != nil {
			return nil, fmt.Errorf("could not start workflows controller: %w", err)
		}
		teardown = append(teardown, Teardown{
			Name: "workflows controller",
			Fn:   cleanupWorkflows,
		})
	}

	if sc.HasService("heartbeater") {
		h, err := heartbeat.New(
			heartbeat.WithMessageQueue(sc.MessageQueue),
			heartbeat.WithRepository(sc.EngineRepository),
			heartbeat.WithLogger(sc.Logger),
		)

		if err != nil {
			return nil, fmt.Errorf("could not create heartbeater: %w", err)
		}

		cleanup, err := h.Start()
		if err != nil {
			return nil, fmt.Errorf("could not start heartbeater: %w", err)
		}
		teardown = append(teardown, Teardown{
			Name: "heartbeater",
			Fn:   cleanup,
		})
	}

	if sc.HasService("grpc") {
		// create the dispatcher
		d, err := dispatcher.New(
			dispatcher.WithAlerter(sc.Alerter),
			dispatcher.WithMessageQueue(sc.MessageQueue),
			dispatcher.WithRepository(sc.EngineRepository),
			dispatcher.WithLogger(sc.Logger),
			dispatcher.WithEntitlementsRepository(sc.EntitlementRepository),
		)

		if err != nil {
			return nil, fmt.Errorf("could not create dispatcher: %w", err)
		}

		dispatcherCleanup, err := d.Start()
		if err != nil {
			return nil, fmt.Errorf("could not start dispatcher: %w", err)
		}

		// create the event ingestor
		ei, err := ingestor.NewIngestor(
			ingestor.WithEventRepository(
				sc.EngineRepository.Event(),
			),
			ingestor.WithStreamEventsRepository(
				sc.EngineRepository.StreamEvent(),
			),
			ingestor.WithLogRepository(
				sc.EngineRepository.Log(),
			),
			ingestor.WithMessageQueue(sc.MessageQueue),
			ingestor.WithEntitlementsRepository(sc.EntitlementRepository),
		)
		if err != nil {
			return nil, fmt.Errorf("could not create ingestor: %w", err)
		}

		adminSvc, err := admin.NewAdminService(
			admin.WithRepository(sc.EngineRepository),
			admin.WithMessageQueue(sc.MessageQueue),
			admin.WithEntitlementsRepository(sc.EntitlementRepository),
		)
		if err != nil {
			return nil, fmt.Errorf("could not create admin service: %w", err)
		}

		grpcOpts := []grpc.ServerOpt{
			grpc.WithConfig(sc),
			grpc.WithIngestor(ei),
			grpc.WithDispatcher(d),
			grpc.WithAdmin(adminSvc),
			grpc.WithLogger(sc.Logger),
			grpc.WithAlerter(sc.Alerter),
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
			return nil, fmt.Errorf("could not create grpc server: %w", err)
		}

		grpcServerCleanup, err := s.Start()
		if err != nil {
			return nil, fmt.Errorf("could not start grpc server: %w", err)
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
			Name: "grpc",
			Fn:   cleanup,
		})
	}

	if sc.HasService("webhookscontroller") {
		partitionTeardown, partitionId, err := p.withTenantWorkers(ctx)

		if err != nil {
			return nil, fmt.Errorf("could not create rebalance controller partitions job: %w", err)
		}

		teardown = append(teardown, *partitionTeardown)

		wh := webhooks.New(sc, partitionId)

		cleanup, err := wh.Start()
		if err != nil {
			return nil, fmt.Errorf("could not create webhook worker: %w", err)
		}

		teardown = append(teardown, Teardown{
			Name: "webhook worker",
			Fn:   cleanup,
		})
	}

	teardown = append(teardown, Teardown{
		Name: "telemetry",
		Fn: func() error {
			return shutdown(ctx)
		},
	})

	l.Debug().Msgf("engine has started")

	if healthProbes {
		h.SetReady(true)
	}

	p.start()

	<-ctx.Done()

	if healthProbes {
		h.SetReady(false)
	}

	return teardown, nil
}
