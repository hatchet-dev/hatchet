package engine

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hatchet-dev/hatchet/internal/services/admin"
	adminv1 "github.com/hatchet-dev/hatchet/internal/services/admin/v1"
	"github.com/hatchet-dev/hatchet/internal/services/controllers/events"
	"github.com/hatchet-dev/hatchet/internal/services/controllers/jobs"
	"github.com/hatchet-dev/hatchet/internal/services/controllers/retention"
	"github.com/hatchet-dev/hatchet/internal/services/controllers/v1/olap"
	"github.com/hatchet-dev/hatchet/internal/services/controllers/v1/task"
	"github.com/hatchet-dev/hatchet/internal/services/controllers/workflows"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher"
	"github.com/hatchet-dev/hatchet/internal/services/grpc"
	"github.com/hatchet-dev/hatchet/internal/services/health"
	"github.com/hatchet-dev/hatchet/internal/services/ingestor"
	"github.com/hatchet-dev/hatchet/internal/services/partition"
	"github.com/hatchet-dev/hatchet/internal/services/scheduler"
	schedulerv1 "github.com/hatchet-dev/hatchet/internal/services/scheduler/v1"
	"github.com/hatchet-dev/hatchet/internal/services/ticker"
	"github.com/hatchet-dev/hatchet/internal/services/webhooks"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/config/loader"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"

	"golang.org/x/sync/errgroup"
)

type Teardown struct {
	Name string
	Fn   func() error
}

func init() {
	svcName := os.Getenv("SERVER_OTEL_SERVICE_NAME")
	collectorURL := os.Getenv("SERVER_OTEL_COLLECTOR_URL")
	insecure := os.Getenv("SERVER_OTEL_INSECURE")
	traceIdRatio := os.Getenv("SERVER_OTEL_TRACE_ID_RATIO")

	var insecureBool bool

	if insecureStr := strings.ToLower(strings.TrimSpace(insecure)); insecureStr == "t" || insecureStr == "true" {
		insecureBool = true
	}

	// we do this to we get the tracer set globally, which is needed by some of the otel
	// integrations for the database before start
	_, err := telemetry.InitTracer(&telemetry.TracerOpts{
		ServiceName:  svcName,
		CollectorURL: collectorURL,
		TraceIdRatio: traceIdRatio,
		Insecure:     insecureBool,
	})

	if err != nil {
		panic(fmt.Errorf("could not initialize tracer: %w", err))
	}
}

func Run(ctx context.Context, cf *loader.ConfigLoader, version string) error {
	serverCleanup, server, err := cf.CreateServerFromConfig(version)
	if err != nil {
		return fmt.Errorf("could not load server config: %w", err)
	}

	var l = server.Logger

	teardown, err := RunWithConfig(ctx, server)

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
			return server.Disconnect()
		},
	})

	time.Sleep(server.Runtime.ShutdownWait)

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
	isV1 := sc.HasService("all") || sc.HasService("scheduler") || sc.HasService("controllers") || sc.HasService("grpc-api")

	if isV1 {
		return runV1Config(ctx, sc)
	}

	return runV0Config(ctx, sc)
}

func runV0Config(ctx context.Context, sc *server.ServerConfig) ([]Teardown, error) {
	var l = sc.Logger

	shutdown, err := telemetry.InitTracer(&telemetry.TracerOpts{
		ServiceName:  sc.OpenTelemetry.ServiceName,
		CollectorURL: sc.OpenTelemetry.CollectorURL,
		TraceIdRatio: sc.OpenTelemetry.TraceIdRatio,
		Insecure:     sc.OpenTelemetry.Insecure,
	})
	if err != nil {
		return nil, fmt.Errorf("could not initialize tracer: %w", err)
	}

	p, err := partition.NewPartition(l, sc.EngineRepository.Tenant())

	if err != nil {
		return nil, fmt.Errorf("could not create partitioner: %w", err)
	}

	teardown := []Teardown{}

	teardown = append(teardown, Teardown{
		Name: "partitioner",
		Fn:   p.Shutdown,
	})

	var h *health.Health
	healthProbes := sc.HasService("health")
	if healthProbes {
		h = health.New(sc.EngineRepository, sc.MessageQueue, sc.Version)
		cleanup, err := h.Start()
		if err != nil {
			return nil, fmt.Errorf("could not start health: %w", err)
		}

		teardown = append(teardown, Teardown{
			Name: "health",
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

	// FIXME: jobscontroller and workflowscontroller are deprecated service names, but there's not a clear upgrade
	// path for old config files.
	if sc.HasService("queue") || sc.HasService("jobscontroller") || sc.HasService("workflowscontroller") || sc.HasService("retention") || sc.HasService("ticker") {
		partitionCleanup, err := p.StartControllerPartition(ctx)
		if err != nil {
			return nil, fmt.Errorf("could not create rebalance controller partitions job: %w", err)
		}

		teardown = append(teardown, Teardown{
			Name: "controller partition",
			Fn:   partitionCleanup,
		})

		schedulePartitionCleanup, err := p.StartSchedulerPartition(ctx)

		if err != nil {
			return nil, fmt.Errorf("could not create create scheduler partition: %w", err)
		}

		teardown = append(teardown, Teardown{
			Name: "scheduler partition",
			Fn:   schedulePartitionCleanup,
		})

		// create the dispatcher
		s, err := scheduler.New(
			scheduler.WithAlerter(sc.Alerter),
			scheduler.WithMessageQueue(sc.MessageQueue),
			scheduler.WithRepository(sc.EngineRepository),
			scheduler.WithLogger(sc.Logger),
			scheduler.WithPartition(p),
			scheduler.WithQueueLoggerConfig(&sc.AdditionalLoggers.Queue),
			scheduler.WithSchedulerPool(sc.SchedulingPool),
		)

		if err != nil {
			return nil, fmt.Errorf("could not create dispatcher: %w", err)
		}

		cleanup, err := s.Start()

		if err != nil {
			return nil, fmt.Errorf("could not start dispatcher: %w", err)
		}

		teardown = append(teardown, Teardown{
			Name: "scheduler",
			Fn:   cleanup,
		})

		sv1, err := schedulerv1.New(
			schedulerv1.WithAlerter(sc.Alerter),
			schedulerv1.WithMessageQueue(sc.MessageQueueV1),
			schedulerv1.WithRepository(sc.EngineRepository),
			schedulerv1.WithV2Repository(sc.V1),
			schedulerv1.WithLogger(sc.Logger),
			schedulerv1.WithPartition(p),
			schedulerv1.WithQueueLoggerConfig(&sc.AdditionalLoggers.Queue),
			schedulerv1.WithSchedulerPool(sc.SchedulingPoolV1),
		)

		if err != nil {
			return nil, fmt.Errorf("could not create scheduler (v1): %w", err)
		}

		cleanup, err = sv1.Start()

		if err != nil {
			return nil, fmt.Errorf("could not start scheduler (v1): %w", err)
		}

		teardown = append(teardown, Teardown{
			Name: "schedulerv1",
			Fn:   cleanup,
		})
	}

	if sc.HasService("ticker") {
		t, err := ticker.New(
			ticker.WithMessageQueue(sc.MessageQueue),
			ticker.WithMessageQueueV1(sc.MessageQueueV1),
			ticker.WithRepository(sc.EngineRepository),
			ticker.WithRepositoryV1(sc.V1),
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

	if sc.HasService("queue") || sc.HasService("jobscontroller") || sc.HasService("workflowscontroller") {
		jc, err := jobs.New(
			jobs.WithAlerter(sc.Alerter),
			jobs.WithMessageQueue(sc.MessageQueue),
			jobs.WithRepository(sc.EngineRepository),
			jobs.WithLogger(sc.Logger),
			jobs.WithPartition(p),
			jobs.WithQueueLoggerConfig(&sc.AdditionalLoggers.Queue),
			jobs.WithPgxStatsLoggerConfig(&sc.AdditionalLoggers.PgxStats),
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
			workflows.WithPartition(p),
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

		tasks, err := task.New(
			task.WithAlerter(sc.Alerter),
			task.WithMessageQueue(sc.MessageQueueV1),
			task.WithRepository(sc.EngineRepository),
			task.WithV1Repository(sc.V1),
			task.WithLogger(sc.Logger),
			task.WithPartition(p),
			task.WithQueueLoggerConfig(&sc.AdditionalLoggers.Queue),
			task.WithPgxStatsLoggerConfig(&sc.AdditionalLoggers.PgxStats),
		)

		if err != nil {
			return nil, fmt.Errorf("could not create tasks controller: %w", err)
		}

		cleanupTasks, err := tasks.Start()

		if err != nil {
			return nil, fmt.Errorf("could not start tasks controller: %w", err)
		}

		teardown = append(teardown, Teardown{
			Name: "tasks controller",
			Fn:   cleanupTasks,
		})

		olap, err := olap.New(
			olap.WithAlerter(sc.Alerter),
			olap.WithMessageQueue(sc.MessageQueueV1),
			olap.WithRepository(sc.V1),
			olap.WithLogger(sc.Logger),
			olap.WithPartition(p),
			olap.WithTenantAlertManager(sc.TenantAlerter),
		)

		if err != nil {
			return nil, fmt.Errorf("could not create olap controller: %w", err)
		}

		cleanupOlap, err := olap.Start()

		if err != nil {
			return nil, fmt.Errorf("could not start olap controller: %w", err)
		}

		teardown = append(teardown, Teardown{
			Name: "olap controller",
			Fn:   cleanupOlap,
		})
	}

	if sc.HasService("retention") {
		rc, err := retention.New(
			retention.WithAlerter(sc.Alerter),
			retention.WithRepository(sc.EngineRepository),
			retention.WithLogger(sc.Logger),
			retention.WithTenantAlerter(sc.TenantAlerter),
			retention.WithPartition(p),
			retention.WithDataRetention(sc.EnableDataRetention),
			retention.WithWorkerRetention(sc.EnableWorkerRetention),
		)

		if err != nil {
			return nil, fmt.Errorf("could not create retention controller: %w", err)
		}

		cleanupRetention, err := rc.Start()
		if err != nil {
			return nil, fmt.Errorf("could not start retention controller: %w", err)
		}
		teardown = append(teardown, Teardown{
			Name: "retention controller",
			Fn:   cleanupRetention,
		})
	}

	if sc.HasService("grpc") {
		cacheInstance := cache.New(10 * time.Second)

		// create the dispatcher
		d, err := dispatcher.New(
			dispatcher.WithAlerter(sc.Alerter),
			dispatcher.WithMessageQueue(sc.MessageQueue),
			dispatcher.WithMessageQueueV1(sc.MessageQueueV1),
			dispatcher.WithRepository(sc.EngineRepository),
			dispatcher.WithRepositoryV1(sc.V1),
			dispatcher.WithLogger(sc.Logger),
			dispatcher.WithEntitlementsRepository(sc.EntitlementRepository),
			dispatcher.WithCache(cacheInstance),
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
			ingestor.WithMessageQueueV1(sc.MessageQueueV1),
			ingestor.WithEntitlementsRepository(sc.EntitlementRepository),
			ingestor.WithStepRunRepository(sc.EngineRepository.StepRun()),
			ingestor.WithRepositoryV1(sc.V1),
		)

		if err != nil {
			return nil, fmt.Errorf("could not create ingestor: %w", err)
		}

		adminSvc, err := admin.NewAdminService(
			admin.WithRepository(sc.EngineRepository),
			admin.WithRepositoryV1(sc.V1),
			admin.WithMessageQueue(sc.MessageQueue),
			admin.WithMessageQueueV1(sc.MessageQueueV1),
			admin.WithEntitlementsRepository(sc.EntitlementRepository),
		)
		if err != nil {
			return nil, fmt.Errorf("could not create admin service: %w", err)
		}

		adminv1Svc, err := adminv1.NewAdminService(
			adminv1.WithRepository(sc.V1),
			adminv1.WithMessageQueue(sc.MessageQueueV1),
		)

		if err != nil {
			return nil, fmt.Errorf("could not create admin service (v1): %w", err)
		}

		grpcOpts := []grpc.ServerOpt{
			grpc.WithConfig(sc),
			grpc.WithIngestor(ei),
			grpc.WithDispatcher(d),
			grpc.WithAdmin(adminSvc),
			grpc.WithAdminV1(adminv1Svc),
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

				cacheInstance.Stop()
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
		cleanup1, err := p.StartTenantWorkerPartition(ctx)

		if err != nil {
			return nil, fmt.Errorf("could not create rebalance controller partitions job: %w", err)
		}

		teardown = append(teardown, Teardown{
			Name: "tenant worker partition",
			Fn:   cleanup1,
		})

		wh := webhooks.New(sc, p, l)

		cleanup2, err := wh.Start()
		if err != nil {
			return nil, fmt.Errorf("could not create webhook worker: %w", err)
		}

		teardown = append(teardown, Teardown{
			Name: "webhook worker",
			Fn:   cleanup2,
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

	<-ctx.Done()

	if healthProbes {
		h.SetReady(false)
	}

	return teardown, nil
}

func runV1Config(ctx context.Context, sc *server.ServerConfig) ([]Teardown, error) {
	var l = sc.Logger

	shutdown, err := telemetry.InitTracer(&telemetry.TracerOpts{
		ServiceName:  sc.OpenTelemetry.ServiceName,
		CollectorURL: sc.OpenTelemetry.CollectorURL,
		TraceIdRatio: sc.OpenTelemetry.TraceIdRatio,
		Insecure:     sc.OpenTelemetry.Insecure,
	})
	if err != nil {
		return nil, fmt.Errorf("could not initialize tracer: %w", err)
	}

	p, err := partition.NewPartition(l, sc.EngineRepository.Tenant())

	if err != nil {
		return nil, fmt.Errorf("could not create partitioner: %w", err)
	}

	teardown := []Teardown{}

	teardown = append(teardown, Teardown{
		Name: "partitioner",
		Fn:   p.Shutdown,
	})

	healthProbes := sc.Runtime.Healthcheck
	var h *health.Health

	if healthProbes {
		h = health.New(sc.EngineRepository, sc.MessageQueue, sc.Version)

		cleanup, err := h.Start()

		if err != nil {
			return nil, fmt.Errorf("could not start health: %w", err)
		}

		teardown = append(teardown, Teardown{
			Name: "health",
			Fn:   cleanup,
		})
	}

	if sc.HasService("all") || sc.HasService("controllers") {
		partitionCleanup, err := p.StartControllerPartition(ctx)

		if err != nil {
			return nil, fmt.Errorf("could not create rebalance controller partitions job: %w", err)
		}

		teardown = append(teardown, Teardown{
			Name: "controller partition",
			Fn:   partitionCleanup,
		})

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

		t, err := ticker.New(
			ticker.WithMessageQueue(sc.MessageQueue),
			ticker.WithMessageQueueV1(sc.MessageQueueV1),
			ticker.WithRepository(sc.EngineRepository),
			ticker.WithRepositoryV1(sc.V1),
			ticker.WithLogger(sc.Logger),
			ticker.WithTenantAlerter(sc.TenantAlerter),
			ticker.WithEntitlementsRepository(sc.EntitlementRepository),
		)

		if err != nil {
			return nil, fmt.Errorf("could not create ticker: %w", err)
		}

		cleanup, err = t.Start()

		if err != nil {
			return nil, fmt.Errorf("could not start ticker: %w", err)
		}

		teardown = append(teardown, Teardown{
			Name: "ticker",
			Fn:   cleanup,
		})

		jc, err := jobs.New(
			jobs.WithAlerter(sc.Alerter),
			jobs.WithMessageQueue(sc.MessageQueue),
			jobs.WithRepository(sc.EngineRepository),
			jobs.WithLogger(sc.Logger),
			jobs.WithPartition(p),
			jobs.WithQueueLoggerConfig(&sc.AdditionalLoggers.Queue),
			jobs.WithPgxStatsLoggerConfig(&sc.AdditionalLoggers.PgxStats),
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
			workflows.WithPartition(p),
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

		rc, err := retention.New(
			retention.WithAlerter(sc.Alerter),
			retention.WithRepository(sc.EngineRepository),
			retention.WithLogger(sc.Logger),
			retention.WithTenantAlerter(sc.TenantAlerter),
			retention.WithPartition(p),
			retention.WithDataRetention(sc.EnableDataRetention),
			retention.WithWorkerRetention(sc.EnableWorkerRetention),
		)

		if err != nil {
			return nil, fmt.Errorf("could not create retention controller: %w", err)
		}

		cleanupRetention, err := rc.Start()

		if err != nil {
			return nil, fmt.Errorf("could not start retention controller: %w", err)
		}

		teardown = append(teardown, Teardown{
			Name: "retention controller",
			Fn:   cleanupRetention,
		})

		tasks, err := task.New(
			task.WithAlerter(sc.Alerter),
			task.WithMessageQueue(sc.MessageQueueV1),
			task.WithRepository(sc.EngineRepository),
			task.WithV1Repository(sc.V1),
			task.WithLogger(sc.Logger),
			task.WithPartition(p),
			task.WithQueueLoggerConfig(&sc.AdditionalLoggers.Queue),
			task.WithPgxStatsLoggerConfig(&sc.AdditionalLoggers.PgxStats),
		)

		if err != nil {
			return nil, fmt.Errorf("could not create tasks controller: %w", err)
		}

		cleanupTasks, err := tasks.Start()

		if err != nil {
			return nil, fmt.Errorf("could not start tasks controller: %w", err)
		}

		teardown = append(teardown, Teardown{
			Name: "tasks controller",
			Fn:   cleanupTasks,
		})

		olap, err := olap.New(
			olap.WithAlerter(sc.Alerter),
			olap.WithMessageQueue(sc.MessageQueueV1),
			olap.WithRepository(sc.V1),
			olap.WithLogger(sc.Logger),
			olap.WithPartition(p),
			olap.WithTenantAlertManager(sc.TenantAlerter),
		)

		if err != nil {
			return nil, fmt.Errorf("could not create olap controller: %w", err)
		}

		cleanupOlap, err := olap.Start()

		if err != nil {
			return nil, fmt.Errorf("could not start olap controller: %w", err)
		}

		teardown = append(teardown, Teardown{
			Name: "olap controller",
			Fn:   cleanupOlap,
		})

		cleanup1, err := p.StartTenantWorkerPartition(ctx)

		if err != nil {
			return nil, fmt.Errorf("could not create rebalance controller partitions job: %w", err)
		}

		teardown = append(teardown, Teardown{
			Name: "tenant worker partition",
			Fn:   cleanup1,
		})

		wh := webhooks.New(sc, p, l)

		cleanup2, err := wh.Start()

		if err != nil {
			return nil, fmt.Errorf("could not create webhook worker: %w", err)
		}

		teardown = append(teardown, Teardown{
			Name: "webhook worker",
			Fn:   cleanup2,
		})
	}

	if sc.HasService("all") || sc.HasService("grpc-api") {
		cacheInstance := cache.New(10 * time.Second)

		// create the dispatcher
		d, err := dispatcher.New(
			dispatcher.WithAlerter(sc.Alerter),
			dispatcher.WithMessageQueue(sc.MessageQueue),
			dispatcher.WithMessageQueueV1(sc.MessageQueueV1),
			dispatcher.WithRepository(sc.EngineRepository),
			dispatcher.WithRepositoryV1(sc.V1),
			dispatcher.WithLogger(sc.Logger),
			dispatcher.WithEntitlementsRepository(sc.EntitlementRepository),
			dispatcher.WithCache(cacheInstance),
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
			ingestor.WithMessageQueueV1(sc.MessageQueueV1),
			ingestor.WithEntitlementsRepository(sc.EntitlementRepository),
			ingestor.WithStepRunRepository(sc.EngineRepository.StepRun()),
			ingestor.WithRepositoryV1(sc.V1),
		)

		if err != nil {
			return nil, fmt.Errorf("could not create ingestor: %w", err)
		}

		adminSvc, err := admin.NewAdminService(
			admin.WithRepository(sc.EngineRepository),
			admin.WithRepositoryV1(sc.V1),
			admin.WithMessageQueue(sc.MessageQueue),
			admin.WithMessageQueueV1(sc.MessageQueueV1),
			admin.WithEntitlementsRepository(sc.EntitlementRepository),
		)

		if err != nil {
			return nil, fmt.Errorf("could not create admin service: %w", err)
		}

		adminv1Svc, err := adminv1.NewAdminService(
			adminv1.WithRepository(sc.V1),
			adminv1.WithMessageQueue(sc.MessageQueueV1),
		)

		if err != nil {
			return nil, fmt.Errorf("could not create admin service (v1): %w", err)
		}

		grpcOpts := []grpc.ServerOpt{
			grpc.WithConfig(sc),
			grpc.WithIngestor(ei),
			grpc.WithDispatcher(d),
			grpc.WithAdmin(adminSvc),
			grpc.WithAdminV1(adminv1Svc),
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

				cacheInstance.Stop()
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

	if sc.HasService("all") || sc.HasService("scheduler") {
		partitionCleanup, err := p.StartSchedulerPartition(ctx)

		if err != nil {
			return nil, fmt.Errorf("could not create create scheduler partition: %w", err)
		}

		teardown = append(teardown, Teardown{
			Name: "scheduler partition",
			Fn:   partitionCleanup,
		})

		// create the dispatcher
		s, err := scheduler.New(
			scheduler.WithAlerter(sc.Alerter),
			scheduler.WithMessageQueue(sc.MessageQueue),
			scheduler.WithRepository(sc.EngineRepository),
			scheduler.WithLogger(sc.Logger),
			scheduler.WithPartition(p),
			scheduler.WithQueueLoggerConfig(&sc.AdditionalLoggers.Queue),
			scheduler.WithSchedulerPool(sc.SchedulingPool),
		)

		if err != nil {
			return nil, fmt.Errorf("could not create dispatcher: %w", err)
		}

		cleanup, err := s.Start()

		if err != nil {
			return nil, fmt.Errorf("could not start dispatcher: %w", err)
		}

		teardown = append(teardown, Teardown{
			Name: "scheduler",
			Fn:   cleanup,
		})

		sv1, err := schedulerv1.New(
			schedulerv1.WithAlerter(sc.Alerter),
			schedulerv1.WithMessageQueue(sc.MessageQueueV1),
			schedulerv1.WithRepository(sc.EngineRepository),
			schedulerv1.WithV2Repository(sc.V1),
			schedulerv1.WithLogger(sc.Logger),
			schedulerv1.WithPartition(p),
			schedulerv1.WithQueueLoggerConfig(&sc.AdditionalLoggers.Queue),
			schedulerv1.WithSchedulerPool(sc.SchedulingPoolV1),
		)

		if err != nil {
			return nil, fmt.Errorf("could not create scheduler (v1): %w", err)
		}

		cleanup, err = sv1.Start()

		if err != nil {
			return nil, fmt.Errorf("could not start scheduler (v1): %w", err)
		}

		teardown = append(teardown, Teardown{
			Name: "schedulerv1",
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

	<-ctx.Done()

	if healthProbes {
		h.SetReady(false)
	}

	return teardown, nil
}
