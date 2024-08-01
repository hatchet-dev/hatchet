package ticker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/integrations/alerting"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

type Ticker interface {
	Start(ctx context.Context) error
}

type TickerImpl struct {
	mq msgqueue.MessageQueue
	l  *zerolog.Logger

	entitlements repository.EntitlementsRepository

	repo repository.EngineRepository
	s    gocron.Scheduler
	ta   *alerting.TenantAlertManager

	crons              sync.Map
	scheduledWorkflows sync.Map

	dv datautils.DataDecoderValidator

	tickerId string

	partitionId string
}

type TickerOpt func(*TickerOpts)

type TickerOpts struct {
	mq msgqueue.MessageQueue
	l  *zerolog.Logger

	entitlements repository.EntitlementsRepository
	repo         repository.EngineRepository
	tickerId     string
	ta           *alerting.TenantAlertManager

	dv datautils.DataDecoderValidator

	partitionId string
}

func defaultTickerOpts() *TickerOpts {
	logger := logger.NewDefaultLogger("ticker")
	return &TickerOpts{
		l:        &logger,
		tickerId: uuid.New().String(),
		dv:       datautils.NewDataDecoderValidator(),
	}
}

func WithMessageQueue(mq msgqueue.MessageQueue) TickerOpt {
	return func(opts *TickerOpts) {
		opts.mq = mq
	}
}

func WithRepository(r repository.EngineRepository) TickerOpt {
	return func(opts *TickerOpts) {
		opts.repo = r
	}
}

func WithEntitlementsRepository(r repository.EntitlementsRepository) TickerOpt {
	return func(opts *TickerOpts) {
		opts.entitlements = r
	}
}

func WithLogger(l *zerolog.Logger) TickerOpt {
	return func(opts *TickerOpts) {
		opts.l = l
	}
}

func WithTenantAlerter(ta *alerting.TenantAlertManager) TickerOpt {
	return func(opts *TickerOpts) {
		opts.ta = ta
	}
}

func WithPartitionId(pid string) TickerOpt {
	return func(opts *TickerOpts) {
		opts.partitionId = pid
	}
}

func New(fs ...TickerOpt) (*TickerImpl, error) {
	opts := defaultTickerOpts()

	for _, f := range fs {
		f(opts)
	}

	if opts.mq == nil {
		return nil, fmt.Errorf("task queue is required. use WithMessageQueue")
	}

	if opts.repo == nil {
		return nil, fmt.Errorf("repository is required. use WithRepository")
	}

	if opts.entitlements == nil {
		return nil, fmt.Errorf("entitlements repository is required. use WithEntitlementsRepository")
	}

	if opts.ta == nil {
		return nil, fmt.Errorf("tenant alerter is required. use WithTenantAlerter")
	}

	if opts.partitionId == "" {
		return nil, fmt.Errorf("partition id is required. use WithPartitionId")
	}

	newLogger := opts.l.With().Str("service", "ticker").Logger()
	opts.l = &newLogger

	s, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))

	if err != nil {
		return nil, fmt.Errorf("could not create scheduler: %w", err)
	}

	return &TickerImpl{
		mq:           opts.mq,
		l:            opts.l,
		repo:         opts.repo,
		entitlements: opts.entitlements,
		s:            s,
		dv:           opts.dv,
		tickerId:     opts.tickerId,
		ta:           opts.ta,
		partitionId:  opts.partitionId,
	}, nil
}

func (t *TickerImpl) Start() (func() error, error) {
	ctx, cancel := context.WithCancel(context.Background())

	t.l.Debug().Msgf("starting ticker %s", t.tickerId)

	// register the ticker
	_, err := t.repo.Ticker().CreateNewTicker(ctx, &repository.CreateTickerOpts{
		ID: t.tickerId,
	})

	if err != nil {
		cancel()
		return nil, err
	}

	_, err = t.s.NewJob(
		gocron.DurationJob(time.Second*5),
		gocron.NewTask(
			t.runUpdateHeartbeat(ctx),
		),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not create update heartbeat job: %w", err)
	}

	_, err = t.s.NewJob(
		gocron.DurationJob(time.Second*1),
		gocron.NewTask(
			t.runPollGetGroupKeyRuns(ctx),
		),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not create update heartbeat job: %w", err)
	}

	_, err = t.s.NewJob(
		// crons only have a resolution of 1 minute, so only poll every 15 seconds
		gocron.DurationJob(time.Second*15),
		gocron.NewTask(
			t.runPollCronSchedules(ctx),
		),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not create poll cron schedules job: %w", err)
	}

	_, err = t.s.NewJob(
		// we look ahead every 5 seconds
		gocron.DurationJob(time.Second*5),
		gocron.NewTask(
			t.runPollSchedules(ctx),
		),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not create poll cron schedules job: %w", err)
	}

	_, err = t.s.NewJob(
		gocron.DurationJob(time.Minute*5),
		gocron.NewTask(
			t.runStreamEventCleanup(ctx),
		),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not schedule stream event cleanup: %w", err)
	}

	// poll for tenant alerts every minute, since minimum alerting frequency is 5 minutes
	_, err = t.s.NewJob(
		gocron.DurationJob(time.Minute*1),
		gocron.NewTask(
			t.runPollTenantAlerts(ctx),
		),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not schedule tenant alert polling: %w", err)
	}

	// poll for expiring tokens every 15 minutes
	_, err = t.s.NewJob(
		gocron.DurationJob(time.Minute*15),
		gocron.NewTask(
			t.runExpiringTokenAlerts(ctx),
		),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not schedule tenant alert polling: %w", err)
	}

	// poll for tenant resource limit alerts every 15 minutes
	_, err = t.s.NewJob(
		gocron.DurationJob(time.Minute*15),
		gocron.NewTask(
			t.runTenantResourceLimitAlerts(ctx),
		),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not schedule tenant resource limit alert polling: %w", err)
	}

	// poll to resolve worker semaphore slots every 1 minute
	// _, err = t.s.NewJob(
	// 	gocron.DurationJob(time.Minute*1),
	// 	gocron.NewTask(
	// 		t.runWorkerSemaphoreSlotResolver(ctx),
	// 	),
	// )

	// if err != nil {
	// 	cancel()
	// 	return nil, fmt.Errorf("could not schedule worker semaphore slot resolver polling: %w", err)
	// }

	// poll to resolve unresolved failed step runs every 30 seconds
	// _, err = t.s.NewJob(
	// 	gocron.DurationJob(time.Second*30),
	// 	gocron.NewTask(
	// 		t.runResolveUnresolvedFailedSteps(ctx),
	// 	),
	// )

	// if err != nil {
	// 	cancel()
	// 	return nil, fmt.Errorf("could not resolve unresolved failed steps polling: %w", err)
	// }

	t.s.Start()

	cleanup := func() error {
		t.l.Debug().Msg("removing ticker")

		cancel()

		if err := t.s.Shutdown(); err != nil {
			return fmt.Errorf("could not shutdown scheduler: %w", err)
		}

		deleteCtx, deleteCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer deleteCancel()

		// delete the ticker
		err = t.repo.Ticker().Delete(deleteCtx, t.tickerId)

		if err != nil {
			t.l.Err(err).Msg("could not delete ticker")
			return err
		}

		return nil
	}

	return cleanup, nil
}

func (t *TickerImpl) runUpdateHeartbeat(ctx context.Context) func() {
	return func() {
		t.l.Debug().Msgf("ticker: updating heartbeat")

		now := time.Now().UTC()

		// update the heartbeat
		_, err := t.repo.Ticker().UpdateTicker(ctx, t.tickerId, &repository.UpdateTickerOpts{
			LastHeartbeatAt: &now,
		})

		if err != nil {
			t.l.Err(err).Msg("could not update heartbeat")
		}
	}
}

func (t *TickerImpl) runStreamEventCleanup(ctx context.Context) func() {
	return func() {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		t.l.Debug().Msgf("ticker: cleaning up stream event")

		err := t.repo.StreamEvent().CleanupStreamEvents(ctx)

		if err != nil {
			t.l.Err(err).Msg("could not cleanup stream events")
		}
	}
}

func (t *TickerImpl) runWorkerSemaphoreSlotResolverTenant(ctx context.Context, tenant *dbsqlc.Tenant) error {
	tenantId := tenant.ID

	tenantIdStr := sqlchelpers.UUIDToStr(tenantId)

	t.l.Debug().Msgf("ticker: resolving orphaned worker semaphore slots for tenant %s", tenantIdStr)

	// keep resolving until the context is done
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		n, err := t.repo.Worker().ResolveWorkerSemaphoreSlots(ctx, tenantId)

		if err != nil {
			t.l.Err(err).Msgf("could not resolve orphaned worker semaphore slots for tenant %s", tenantIdStr)
			return err
		}

		if n.HasResolved {
			t.l.Warn().Msgf("resolved orphaned worker semaphore slots for tenant %s", tenantIdStr)
		}

		if !n.HasMore {
			return nil
		}
	}
}

func (t *TickerImpl) runWorkerSemaphoreSlotResolver(ctx context.Context) func() {
	return func() {
		ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()

		t.l.Debug().Msgf("ticker: resolving orphaned worker semaphore slots")

		// list all tenants
		tenants, err := t.repo.Tenant().ListTenantsByControllerPartition(ctx, t.partitionId)

		if err != nil {
			t.l.Err(err).Msg("could not list tenants")
			return
		}

		g := new(errgroup.Group)

		for i := range tenants {
			g.Go(func() error {
				return t.runWorkerSemaphoreSlotResolverTenant(ctx, tenants[i])
			})
		}

		err = g.Wait()

		if err != nil {
			t.l.Err(err).Msg("could not run worker semaphore slot resolver")
		}
	}
}

// func (t *TickerImpl) runResolveUnresolvedFailedSteps(ctx context.Context) func() {
// 	return func() {

// 		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
// 		defer cancel()

// 		toResolve, err := t.repo.Ticker().PollUnresolvedFailedStepRuns(ctx)

// 		if err != nil {
// 			t.l.Err(err).Msg("could not poll unresolved failed step runs")
// 			return
// 		}

// 		if len(toResolve) > 0 {
// 			t.l.Warn().Msgf("attempting to resolve %d unresolved failed step runs", len(toResolve))
// 		}

// 		for _, stepRun := range toResolve {
// 			_, err := t.repo.StepRun().ResolveRelatedStatuses(ctx, stepRun.TenantId, stepRun.ID)

// 			if err != nil {
// 				t.l.Err(err).Msgf("could not resolve step run %s", sqlchelpers.UUIDToStr(stepRun.ID))
// 			}
// 		}

// 	}
// }
