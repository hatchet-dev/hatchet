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
	"github.com/hatchet-dev/hatchet/internal/logger"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/repository"
)

type Ticker interface {
	Start(ctx context.Context) error
}

type TickerImpl struct {
	mq   msgqueue.MessageQueue
	l    *zerolog.Logger
	repo repository.EngineRepository
	s    gocron.Scheduler

	crons              sync.Map
	scheduledWorkflows sync.Map

	dv datautils.DataDecoderValidator

	tickerId string
}

type TickerOpt func(*TickerOpts)

type TickerOpts struct {
	mq       msgqueue.MessageQueue
	l        *zerolog.Logger
	repo     repository.EngineRepository
	tickerId string

	dv datautils.DataDecoderValidator
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

func WithLogger(l *zerolog.Logger) TickerOpt {
	return func(opts *TickerOpts) {
		opts.l = l
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

	newLogger := opts.l.With().Str("service", "ticker").Logger()
	opts.l = &newLogger

	s, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))

	if err != nil {
		return nil, fmt.Errorf("could not create scheduler: %w", err)
	}

	return &TickerImpl{
		mq:       opts.mq,
		l:        opts.l,
		repo:     opts.repo,
		s:        s,
		dv:       opts.dv,
		tickerId: opts.tickerId,
	}, nil
}

func (t *TickerImpl) Start() (func() error, error) {
	ctx, cancel := context.WithCancel(context.Background())

	t.l.Debug().Msgf("starting ticker %s", t.tickerId)

	// register the ticker
	_, err := t.repo.Ticker().CreateNewTicker(&repository.CreateTickerOpts{
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
			t.runPollStepRuns(ctx),
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
			t.runStreamEventCleanup(),
		),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not schedule stream event cleanup: %w", err)
	}

	t.s.Start()

	cleanup := func() error {
		t.l.Debug().Msg("removing ticker")

		cancel()

		// delete the ticker
		err = t.repo.Ticker().Delete(t.tickerId)

		if err != nil {
			t.l.Err(err).Msg("could not delete ticker")
			return err
		}

		if err := t.s.Shutdown(); err != nil {
			return fmt.Errorf("could not shutdown scheduler: %w", err)
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
		_, err := t.repo.Ticker().UpdateTicker(t.tickerId, &repository.UpdateTickerOpts{
			LastHeartbeatAt: &now,
		})

		if err != nil {
			t.l.Err(err).Msg("could not update heartbeat")
		}
	}
}

func (t *TickerImpl) runStreamEventCleanup() func() {
	return func() {
		t.l.Debug().Msgf("ticker: cleaning up stream event")

		err := t.repo.StreamEvent().CleanupStreamEvents()

		if err != nil {
			t.l.Err(err).Msg("could not cleanup stream events")
		}
	}
}
