package ticker

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/internal/taskqueue"
	"github.com/rs/zerolog"
)

type Ticker interface {
	Start(ctx context.Context) error
}

type TickerImpl struct {
	tq   taskqueue.TaskQueue
	l    *zerolog.Logger
	repo repository.Repository
	s    *gocron.Scheduler

	crons           sync.Map
	jobRuns         sync.Map
	stepRuns        sync.Map
	stepRunRequeues sync.Map

	dv datautils.DataDecoderValidator

	tickerId string
}

type timeoutCtx struct {
	ctx    context.Context
	cancel context.CancelFunc
}

type TickerOpt func(*TickerOpts)

type TickerOpts struct {
	tq       taskqueue.TaskQueue
	l        *zerolog.Logger
	repo     repository.Repository
	tickerId string

	dv datautils.DataDecoderValidator
}

func defaultTickerOpts() *TickerOpts {
	logger := zerolog.New(os.Stderr)
	return &TickerOpts{
		l:        &logger,
		tickerId: uuid.New().String(),
		dv:       datautils.NewDataDecoderValidator(),
	}
}

func WithTaskQueue(tq taskqueue.TaskQueue) TickerOpt {
	return func(opts *TickerOpts) {
		opts.tq = tq
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

	if opts.tq == nil {
		return nil, fmt.Errorf("task queue is required. use WithTaskQueue")
	}

	if opts.repo == nil {
		return nil, fmt.Errorf("repository is required. use WithRepository")
	}

	s := gocron.NewScheduler(time.UTC)

	return &TickerImpl{
		tq:       opts.tq,
		l:        opts.l,
		repo:     opts.repo,
		s:        s,
		dv:       opts.dv,
		tickerId: opts.tickerId,
	}, nil
}

func (t *TickerImpl) Start(ctx context.Context) error {
	t.l.Debug().Msg("starting ticker")

	// register the ticker
	ticker, err := t.repo.Ticker().CreateNewTicker(&repository.CreateTickerOpts{
		ID: t.tickerId,
	})

	if err != nil {
		return err
	}

	// subscribe to a task queue with the dispatcher id
	taskChan, err := t.tq.Subscribe(ctx, taskqueue.QueueTypeFromTicker(ticker))

	if err != nil {
		return err
	}

	_, err = t.s.Every(5).Seconds().Do(t.runStepRunRequeue(ctx))

	if err != nil {
		return fmt.Errorf("could not schedule step run requeue: %w", err)
	}

	_, err = t.s.Every(5).Seconds().Do(t.runUpdateHeartbeat(ctx))

	if err != nil {
		return fmt.Errorf("could not schedule heartbeat update: %w", err)
	}

	t.s.StartAsync()

	for {
		select {
		case <-ctx.Done():
			t.l.Debug().Msg("removing ticker")

			// delete the ticker
			err = t.repo.Ticker().Delete(t.tickerId)

			if err != nil {
				t.l.Err(err).Msg("could not delete ticker")
				return err
			}

			// add the task after the ticker is deleted
			err := t.tq.AddTask(
				ctx,
				taskqueue.JOB_PROCESSING_QUEUE,
				tickerRemoved(t.tickerId),
			)

			if err != nil {
				t.l.Err(err).Msg("could not add ticker removed task")
				return err
			}

			// return err
			return nil
		case task := <-taskChan:
			err = t.handleTask(ctx, task)

			if err != nil {
				t.l.Error().Err(err).Msgf("could not handle event task %s", task.ID)
			}
		}
	}
}

func (t *TickerImpl) handleTask(ctx context.Context, task *taskqueue.Task) error {
	switch task.ID {
	case "schedule-step-run-timeout":
		return t.handleScheduleStepRunTimeout(ctx, task)
	case "cancel-step-run-timeout":
		return t.handleCancelStepRunTimeout(ctx, task)
	case "schedule-job-run-timeout":
		return t.handleScheduleJobRunTimeout(ctx, task)
	case "cancel-job-run-timeout":
		return t.handleCancelJobRunTimeout(ctx, task)
	// case "schedule-step-requeue":
	// 	return t.handleScheduleStepRunRequeue(ctx, task)
	// case "cancel-step-requeue":
	// 	return t.handleCancelStepRunRequeue(ctx, task)
	case "schedule-cron":
		return t.handleScheduleCron(ctx, task)
	case "cancel-cron":
		return t.handleCancelCron(ctx, task)
	}

	return fmt.Errorf("unknown task: %s in queue %s", task.ID, string(task.Queue))
}

func (t *TickerImpl) runStepRunRequeue(ctx context.Context) func() {
	return func() {
		t.l.Debug().Msgf("ticker: checking step run requeue")

		// list all tenants
		tenants, err := t.repo.Tenant().ListTenants()

		if err != nil {
			t.l.Err(err).Msg("could not list tenants")
			return
		}

		for i := range tenants {
			t.l.Debug().Msgf("adding step run requeue task for tenant %s", tenants[i].ID)

			err := t.tq.AddTask(
				ctx,
				taskqueue.JOB_PROCESSING_QUEUE,
				tasktypes.TenantToStepRunRequeueTask(tenants[i]),
			)

			if err != nil {
				t.l.Err(err).Msg("could not add step run requeue task")
			}
		}
	}
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

func tickerRemoved(tickerId string) *taskqueue.Task {
	payload, _ := datautils.ToJSONMap(tasktypes.RemoveTickerTaskPayload{
		TickerId: tickerId,
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.RemoveTickerTaskMetadata{})

	return &taskqueue.Task{
		ID:       "ticker-removed",
		Queue:    taskqueue.JOB_PROCESSING_QUEUE,
		Payload:  payload,
		Metadata: metadata,
	}
}
