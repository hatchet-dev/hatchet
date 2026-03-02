package ticker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/integrations/alerting"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/syncx"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

type Ticker interface {
	Start(ctx context.Context) error
}

type TickerImpl struct {
	mqv1 msgqueue.MessageQueue
	l    *zerolog.Logger

	repov1 v1.Repository
	s      gocron.Scheduler
	ta     *alerting.TenantAlertManager

	scheduledWorkflows syncx.Map[string, context.CancelFunc]

	dv datautils.DataDecoderValidator

	tickerId uuid.UUID

	userCronScheduler     gocron.Scheduler
	userCronSchedulerLock sync.Mutex

	// maps a unique key for the cron schedule to a UUID, because the gocron library depends on uuids
	// as unique identifiers for scheduled jobs
	userCronSchedulesToIds map[string]string
}

type TickerOpt func(*TickerOpts)

type TickerOpts struct {
	mqv1 msgqueue.MessageQueue
	l    *zerolog.Logger

	repov1   v1.Repository
	tickerId uuid.UUID
	ta       *alerting.TenantAlertManager

	dv datautils.DataDecoderValidator
}

func defaultTickerOpts() *TickerOpts {
	logger := logger.NewDefaultLogger("ticker")
	return &TickerOpts{
		l:        &logger,
		tickerId: uuid.New(),
		dv:       datautils.NewDataDecoderValidator(),
	}
}

func WithMessageQueueV1(mq msgqueue.MessageQueue) TickerOpt {
	return func(opts *TickerOpts) {
		opts.mqv1 = mq
	}
}

func WithRepositoryV1(r v1.Repository) TickerOpt {
	return func(opts *TickerOpts) {
		opts.repov1 = r
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

func New(fs ...TickerOpt) (*TickerImpl, error) {
	opts := defaultTickerOpts()

	for _, f := range fs {
		f(opts)
	}

	if opts.mqv1 == nil {
		return nil, fmt.Errorf("task queue v1 is required. use WithMessageQueueV1")
	}

	if opts.repov1 == nil {
		return nil, fmt.Errorf("repository v1 is required. use WithRepositoryV1")
	}

	if opts.ta == nil {
		return nil, fmt.Errorf("tenant alerter is required. use WithTenantAlerter")
	}

	newLogger := opts.l.With().Str("service", "ticker").Logger()
	opts.l = &newLogger

	s, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))

	if err != nil {
		return nil, fmt.Errorf("could not create scheduler: %w", err)
	}

	return &TickerImpl{
		mqv1:                   opts.mqv1,
		l:                      opts.l,
		repov1:                 opts.repov1,
		s:                      s,
		dv:                     opts.dv,
		tickerId:               opts.tickerId,
		ta:                     opts.ta,
		userCronSchedulesToIds: make(map[string]string),
	}, nil
}

func (t *TickerImpl) Start() (func() error, error) {
	ctx, cancel := context.WithCancel(context.Background())

	t.l.Debug().Msgf("starting ticker %s", t.tickerId)

	// register the ticker
	_, err := t.repov1.Ticker().CreateNewTicker(ctx, &v1.CreateTickerOpts{
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
		// crons only have a resolution of 1 minute, so only poll every 15 seconds
		gocron.DurationJob(time.Second*15),
		gocron.NewTask(
			t.runPollCronSchedules(ctx),
		),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
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
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not create poll cron schedules job: %w", err)
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

	userCronScheduler, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not create user cron scheduler: %w", err)
	}

	t.userCronScheduler = userCronScheduler

	t.s.Start()
	t.userCronScheduler.Start()

	cleanup := func() error {
		t.l.Debug().Msg("removing ticker")

		cancel()

		if err := t.s.Shutdown(); err != nil {
			return fmt.Errorf("could not shutdown scheduler: %w", err)
		}

		if err := t.userCronScheduler.Shutdown(); err != nil {
			return fmt.Errorf("could not shutdown user cron scheduler: %w", err)
		}

		deleteCtx, deleteCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer deleteCancel()

		// delete the ticker
		err = t.repov1.Ticker().DeactivateTicker(deleteCtx, t.tickerId)

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
		_, err := t.repov1.Ticker().UpdateTicker(ctx, t.tickerId, &v1.UpdateTickerOpts{
			LastHeartbeatAt: &now,
		})

		if err != nil {
			t.l.Err(err).Msg("could not update heartbeat")
		}
	}
}
