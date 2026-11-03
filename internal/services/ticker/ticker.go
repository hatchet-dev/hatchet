package ticker

import (
	"context"
	"errors"
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
	mqv1   msgqueue.MessageQueue
	pubsub msgqueue.PubSub
	l      *zerolog.Logger

	repov1 v1.Repository
	s      gocron.Scheduler
	ta     *alerting.TenantAlertManager

	scheduledWorkflows syncx.Map[string, context.CancelFunc]

	dv datautils.DataDecoderValidator

	tickerId uuid.UUID

	userCronScheduler     gocron.Scheduler
	userCronSchedulerLock sync.Mutex

	userCronRefreshLock sync.Mutex
	cronRefreshInterval time.Duration

	// maps a unique key for the cron schedule to a UUID, because the gocron library depends on uuids
	// as unique identifiers for scheduled jobs
	userCronSchedulesToIds map[string]string
}

type TickerOpt func(*TickerOpts)

type TickerOpts struct {
	mqv1   msgqueue.MessageQueue
	pubsub msgqueue.PubSub
	l      *zerolog.Logger

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

func WithPubSub(pubsub msgqueue.PubSub) TickerOpt {
	return func(opts *TickerOpts) {
		opts.pubsub = pubsub
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

	if opts.pubsub == nil {
		return nil, fmt.Errorf("pubsub is required. use WithPubSub")
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

	userCronScheduler, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))

	if err != nil {
		return nil, fmt.Errorf("could not create user cron scheduler: %w", err)
	}

	return &TickerImpl{
		mqv1:                   opts.mqv1,
		pubsub:                 opts.pubsub,
		l:                      opts.l,
		repov1:                 opts.repov1,
		s:                      s,
		dv:                     opts.dv,
		tickerId:               opts.tickerId,
		ta:                     opts.ta,
		userCronScheduler:      userCronScheduler,
		userCronSchedulesToIds: make(map[string]string),
		cronRefreshInterval:    time.Second,
	}, nil
}

func (t *TickerImpl) Start() (func() error, error) {
	ctx, cancel := context.WithCancel(context.Background())

	t.l.Debug().Ctx(ctx).Msgf("starting ticker %s", t.tickerId)

	_, err := t.repov1.Ticker().CreateNewTicker(ctx, &v1.CreateTickerOpts{
		ID: t.tickerId,
	})

	if err != nil {
		cancel()
		return nil, err
	}

	// initialize the cron schedules, so no need to wait for 15 seconds or
	// a cron to be created
	t.refreshCronSchedules(ctx)()

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
			t.refreshCronSchedules(ctx),
		),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not create poll cron schedules job: %w", err)
	}

	_, err = t.s.NewJob(
		// we look ahead every 5 seconds
		// FIXME: add another queue similar to cron jobs so that tasks scheduled less than 5 seconds
		// into the future do not take 5 seconds to be triggered
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

	cleanupCronRefresh := t.subscribeCronRefresh(ctx)

	t.s.Start()
	t.userCronScheduler.Start()

	cleanup := func() error {
		t.l.Debug().Ctx(ctx).Msg("removing ticker")

		cancel()

		if err := cleanupCronRefresh(); err != nil {
			t.l.Err(err).Ctx(ctx).Msg("could not clean up cron refresh subscriptions")
		}

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
			t.l.Err(err).Ctx(ctx).Msg("could not delete ticker")
			return err
		}

		return nil
	}

	return cleanup, nil
}

// subscribeCronRefresh refreshes the cron schedules when a cron changes. A
// failed subscription is logged rather than fatal: the 15s poll still picks up
// the change.
func (t *TickerImpl) subscribeCronRefresh(ctx context.Context) func() error {
	requests := make(chan struct{}, 1)

	go t.runCronRefreshes(ctx, requests)

	handler := func(*msgqueue.Message) error {
		select {
		case requests <- struct{}{}:
		default:
		}

		return nil
	}

	cleanups := make([]func() error, 0, 2)

	if cleanup, err := t.pubsub.Sub(msgqueue.CronRefreshTopic(), handler); err != nil {
		t.l.Err(err).Ctx(ctx).Msg("could not subscribe to cron refresh topic")
	} else {
		cleanups = append(cleanups, cleanup)
	}

	// older API and engine versions send cron changes to this queue instead
	if cleanup, err := t.mqv1.Subscribe(msgqueue.TICKER_UPDATE_QUEUE, handler, msgqueue.NoOpHook); err != nil {
		t.l.Err(err).Ctx(ctx).Msg("could not subscribe to cron trigger update queue")
	} else {
		cleanups = append(cleanups, cleanup)
	}

	return func() error {
		var errs error

		for _, cleanup := range cleanups {
			errs = errors.Join(errs, cleanup())
		}

		return errs
	}
}

// runCronRefreshes refreshes once per request, at most once per
// cronRefreshInterval. Every ticker receives every cron change, and workers
// re-register their crons on startup, so a burst of changes must not turn into
// one full cron poll per message on each ticker. Requests that arrive during a
// refresh or its cooldown fold into one more refresh.
func (t *TickerImpl) runCronRefreshes(ctx context.Context, requests <-chan struct{}) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-requests:
		}

		t.refreshCronSchedules(ctx)()

		select {
		case <-ctx.Done():
			return
		case <-time.After(t.cronRefreshInterval):
		}
	}
}

func (t *TickerImpl) runUpdateHeartbeat(ctx context.Context) func() {
	return func() {
		t.l.Debug().Ctx(ctx).Msgf("ticker: updating heartbeat")

		now := time.Now().UTC()

		// update the heartbeat
		_, err := t.repov1.Ticker().UpdateTicker(ctx, t.tickerId, &v1.UpdateTickerOpts{
			LastHeartbeatAt: &now,
		})

		if err != nil {
			t.l.Err(err).Ctx(ctx).Msg("could not update heartbeat")
		}
	}
}
