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
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
)

type Ticker interface {
	Start(ctx context.Context) error
}

type TickerImpl struct {
	mq   msgqueue.MessageQueue
	l    *zerolog.Logger
	repo repository.Repository
	s    gocron.Scheduler

	crons              sync.Map
	scheduledWorkflows sync.Map
	stepRuns           sync.Map
	getGroupKeyRuns    sync.Map

	dv datautils.DataDecoderValidator

	tickerId string
}

type timeoutCtx struct {
	ctx    context.Context
	cancel context.CancelFunc
}

type TickerOpt func(*TickerOpts)

type TickerOpts struct {
	mq       msgqueue.MessageQueue
	l        *zerolog.Logger
	repo     repository.Repository
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

func WithRepository(r repository.Repository) TickerOpt {
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
	ticker, err := t.repo.Ticker().CreateNewTicker(&repository.CreateTickerOpts{
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

	t.s.Start()

	wg := sync.WaitGroup{}

	f := func(task *msgqueue.Message) error {
		wg.Add(1)

		defer wg.Done()

		err := t.handleTask(ctx, task)
		if err != nil {
			t.l.Error().Err(err).Msgf("could not handle ticker task %s", task.ID)
			return err
		}

		return nil
	}

	// subscribe to a task queue with the dispatcher id
	cleanupQueue, err := t.mq.Subscribe(msgqueue.QueueTypeFromTickerID(ticker.ID), msgqueue.NoOpHook, f)

	if err != nil {
		cancel()
		return nil, err
	}

	cleanup := func() error {
		t.l.Debug().Msg("removing ticker")

		cancel()

		if err := cleanupQueue(); err != nil {
			return fmt.Errorf("could not cleanup queue: %w", err)
		}

		wg.Wait()

		// delete the ticker
		err = t.repo.Ticker().Delete(t.tickerId)

		if err != nil {
			t.l.Err(err).Msg("could not delete ticker")
			return err
		}

		// add the task after the ticker is deleted
		err = t.mq.AddMessage(
			ctx,
			msgqueue.JOB_PROCESSING_QUEUE,
			tickerRemoved(t.tickerId),
		)

		if err != nil {
			t.l.Err(err).Msg("could not add ticker removed task")
			return err
		}

		if err := t.s.Shutdown(); err != nil {
			return fmt.Errorf("could not shutdown scheduler: %w", err)
		}

		return nil
	}

	return cleanup, nil
}

func (t *TickerImpl) handleTask(ctx context.Context, task *msgqueue.Message) error {
	switch task.ID {
	case "schedule-step-run-timeout":
		return t.handleScheduleStepRunTimeout(ctx, task)
	case "cancel-step-run-timeout":
		return t.handleCancelStepRunTimeout(ctx, task)
	case "schedule-get-group-key-run-timeout":
		return t.handleScheduleGetGroupKeyRunTimeout(ctx, task)
	case "cancel-get-group-key-run-timeout":
		return t.handleCancelGetGroupKeyRunTimeout(ctx, task)
	case "schedule-cron":
		return t.handleScheduleCron(ctx, task)
	case "cancel-cron":
		return t.handleCancelCron(ctx, task)
	case "schedule-workflow":
		return t.handleScheduleWorkflow(ctx, task)
	case "cancel-workflow":
		return t.handleCancelWorkflow(ctx, task)
	}

	return fmt.Errorf("unknown task: %s", task.ID)
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

func tickerRemoved(tickerId string) *msgqueue.Message {
	payload, _ := datautils.ToJSONMap(tasktypes.RemoveTickerTaskPayload{
		TickerId: tickerId,
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.RemoveTickerTaskMetadata{})

	return &msgqueue.Message{
		ID:       "ticker-removed",
		Payload:  payload,
		Metadata: metadata,
		Retries:  3,
	}
}
