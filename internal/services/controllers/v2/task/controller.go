package task

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/cel"
	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/partition"
	"github.com/hatchet-dev/hatchet/internal/services/shared/recoveryutils"
	"github.com/hatchet-dev/hatchet/pkg/config/shared"
	hatcheterrors "github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	v2 "github.com/hatchet-dev/hatchet/pkg/repository/v2"
)

type TasksController interface {
	Start(ctx context.Context) error
}

type TasksControllerImpl struct {
	mq             msgqueue.MessageQueue
	l              *zerolog.Logger
	queueLogger    *zerolog.Logger
	pgxStatsLogger *zerolog.Logger
	repo           v2.TaskRepository
	dv             datautils.DataDecoderValidator
	s              gocron.Scheduler
	a              *hatcheterrors.Wrapped
	p              *partition.Partition
	celParser      *cel.CELParser

	reassignMutexes sync.Map
}

type TasksControllerOpt func(*TasksControllerOpts)

type TasksControllerOpts struct {
	mq             msgqueue.MessageQueue
	l              *zerolog.Logger
	repo           v2.TaskRepository
	dv             datautils.DataDecoderValidator
	alerter        hatcheterrors.Alerter
	p              *partition.Partition
	queueLogger    *zerolog.Logger
	pgxStatsLogger *zerolog.Logger
}

func defaultTasksControllerOpts() *TasksControllerOpts {
	l := logger.NewDefaultLogger("tasks-controller")
	alerter := hatcheterrors.NoOpAlerter{}

	queueLogger := logger.NewDefaultLogger("queue")
	pgxStatsLogger := logger.NewDefaultLogger("pgx-stats")

	return &TasksControllerOpts{
		l:              &l,
		dv:             datautils.NewDataDecoderValidator(),
		alerter:        alerter,
		queueLogger:    &queueLogger,
		pgxStatsLogger: &pgxStatsLogger,
	}
}

func WithMessageQueue(mq msgqueue.MessageQueue) TasksControllerOpt {
	return func(opts *TasksControllerOpts) {
		opts.mq = mq
	}
}

func WithLogger(l *zerolog.Logger) TasksControllerOpt {
	return func(opts *TasksControllerOpts) {
		opts.l = l
	}
}

func WithQueueLoggerConfig(lc *shared.LoggerConfigFile) TasksControllerOpt {
	return func(opts *TasksControllerOpts) {
		l := logger.NewStdErr(lc, "queue")
		opts.queueLogger = &l
	}
}

func WithPgxStatsLoggerConfig(lc *shared.LoggerConfigFile) TasksControllerOpt {
	return func(opts *TasksControllerOpts) {
		l := logger.NewStdErr(lc, "pgx-stats")
		opts.pgxStatsLogger = &l
	}
}

func WithAlerter(a hatcheterrors.Alerter) TasksControllerOpt {
	return func(opts *TasksControllerOpts) {
		opts.alerter = a
	}
}

func WithRepository(r v2.TaskRepository) TasksControllerOpt {
	return func(opts *TasksControllerOpts) {
		opts.repo = r
	}
}

func WithPartition(p *partition.Partition) TasksControllerOpt {
	return func(opts *TasksControllerOpts) {
		opts.p = p
	}
}

func WithDataDecoderValidator(dv datautils.DataDecoderValidator) TasksControllerOpt {
	return func(opts *TasksControllerOpts) {
		opts.dv = dv
	}
}

func New(fs ...TasksControllerOpt) (*TasksControllerImpl, error) {
	opts := defaultTasksControllerOpts()

	for _, f := range fs {
		f(opts)
	}

	if opts.mq == nil {
		return nil, fmt.Errorf("task queue is required. use WithMessageQueue")
	}

	if opts.repo == nil {
		return nil, fmt.Errorf("repository is required. use WithRepository")
	}

	if opts.p == nil {
		return nil, errors.New("partition is required. use WithPartition")
	}

	newLogger := opts.l.With().Str("service", "tasks-controller").Logger()
	opts.l = &newLogger

	s, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))

	if err != nil {
		return nil, fmt.Errorf("could not create scheduler: %w", err)
	}

	a := hatcheterrors.NewWrapped(opts.alerter)
	a.WithData(map[string]interface{}{"service": "tasks-controller"})

	return &TasksControllerImpl{
		mq:             opts.mq,
		l:              opts.l,
		queueLogger:    opts.queueLogger,
		pgxStatsLogger: opts.pgxStatsLogger,
		repo:           opts.repo,
		dv:             opts.dv,
		s:              s,
		a:              a,
		p:              opts.p,
		celParser:      cel.NewCELParser(),
	}, nil
}

func (tc *TasksControllerImpl) Start() (func() error, error) {
	mqBuffer := msgqueue.NewMQBuffer(msgqueue.TASK_PROCESSING_QUEUE, tc.mq, tc.handleBufferedMsgs)
	wg := sync.WaitGroup{}

	tc.s.Start()

	cleanupBuffer, err := mqBuffer.Start()

	if err != nil {
		return nil, fmt.Errorf("could not start message queue buffer: %w", err)
	}

	cleanup := func() error {
		if err := cleanupBuffer(); err != nil {
			return err
		}

		if err := tc.s.Shutdown(); err != nil {
			return fmt.Errorf("could not shutdown scheduler: %w", err)
		}

		wg.Wait()

		return nil
	}

	return cleanup, nil
}

func (tc *TasksControllerImpl) handleBufferedMsgs(tenantId, msgId string, msgs []*msgqueue.Message) (err error) {
	defer func() {
		if r := recover(); r != nil {
			recoverErr := recoveryutils.RecoverWithAlert(tc.l, tc.a, r)

			if recoverErr != nil {
				err = recoverErr
			}
		}
	}()

	switch msgId {
	case "create-task":
		// return ec.handleJobRunQueued(ctx, task)
	case "replay-task":
		// return ec.handleStepRunReplay(ctx, task)
	case "fail-task":
		// return ec.handleStepRunRetry(ctx, task)
	case "complete-task":
		// return ec.handleStepRunQueued(ctx, task)
	case "start-task":
		// return ec.handleStepRunStarted(ctx, task)
	case "cancel-task":
		// return ec.handleJobRunCancelled(ctx, task)
	}

	return fmt.Errorf("unknown message id: %s", msgId)
}

// handleCreateTask is responsible for inserting tasks into the database. tasks are assigned a UUID externally which is
// passed through the message queue. tasks are given an internal ID once inserted into the database.
func (tc *TasksControllerImpl) handleCreateTask(ctx context.Context, tenantId string, msgs []*msgqueue.Message) error {
	panic("not implemented")
}

func (tc *TasksControllerImpl) handleReplayTask(ctx context.Context, tenantId string, msgs []*msgqueue.Message) error {
	panic("not implemented")

	// call repository method ReplayTask
}

func (tc *TasksControllerImpl) handleFailTask(ctx context.Context, tenantId string, msgs []*msgqueue.Message) error {
	panic("not implemented")

	// call repository method FailTask
}

func (tc *TasksControllerImpl) handleCompleteTask(ctx context.Context, tenantId string, msgs []*msgqueue.Message) error {
	panic("not implemented")

	// call repository method CompleteTask
}

func (tc *TasksControllerImpl) handleStartTask(ctx context.Context, tenantId string, msgs []*msgqueue.Message) error {
	panic("not implemented")

	// call repository method StartTask
}

func (tc *TasksControllerImpl) handleCancelTask(ctx context.Context, tenantId string, msgs []*msgqueue.Message) error {
	panic("not implemented")

	// call repository method CancelTask
}
