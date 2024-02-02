package workflows

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/logger"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/internal/taskqueue"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
)

type WorkflowsController interface {
	Start(ctx context.Context) error
}

type WorkflowsControllerImpl struct {
	tq   taskqueue.TaskQueue
	l    *zerolog.Logger
	repo repository.Repository
	dv   datautils.DataDecoderValidator
}

type WorkflowsControllerOpt func(*WorkflowsControllerOpts)

type WorkflowsControllerOpts struct {
	tq   taskqueue.TaskQueue
	l    *zerolog.Logger
	repo repository.Repository
	dv   datautils.DataDecoderValidator
}

func defaultWorkflowsControllerOpts() *WorkflowsControllerOpts {
	logger := logger.NewDefaultLogger("workflows-controller")
	return &WorkflowsControllerOpts{
		l:  &logger,
		dv: datautils.NewDataDecoderValidator(),
	}
}

func WithTaskQueue(tq taskqueue.TaskQueue) WorkflowsControllerOpt {
	return func(opts *WorkflowsControllerOpts) {
		opts.tq = tq
	}
}

func WithLogger(l *zerolog.Logger) WorkflowsControllerOpt {
	return func(opts *WorkflowsControllerOpts) {
		opts.l = l
	}
}

func WithRepository(r repository.Repository) WorkflowsControllerOpt {
	return func(opts *WorkflowsControllerOpts) {
		opts.repo = r
	}
}

func WithDataDecoderValidator(dv datautils.DataDecoderValidator) WorkflowsControllerOpt {
	return func(opts *WorkflowsControllerOpts) {
		opts.dv = dv
	}
}

func New(fs ...WorkflowsControllerOpt) (*WorkflowsControllerImpl, error) {
	opts := defaultWorkflowsControllerOpts()

	for _, f := range fs {
		f(opts)
	}

	if opts.tq == nil {
		return nil, fmt.Errorf("task queue is required. use WithTaskQueue")
	}

	if opts.repo == nil {
		return nil, fmt.Errorf("repository is required. use WithRepository")
	}

	newLogger := opts.l.With().Str("service", "workflows-controller").Logger()
	opts.l = &newLogger

	return &WorkflowsControllerImpl{
		tq:   opts.tq,
		l:    opts.l,
		repo: opts.repo,
		dv:   opts.dv,
	}, nil
}

func (wc *WorkflowsControllerImpl) Start(ctx context.Context) error {
	wc.l.Debug().Msg("starting workflows controller")

	taskChan, err := wc.tq.Subscribe(ctx, taskqueue.WORKFLOW_PROCESSING_QUEUE)

	if err != nil {
		return err
	}

	// TODO: close when ctx is done
	for task := range taskChan {
		go func(task *taskqueue.Task) {
			err = wc.handleTask(ctx, task)

			if err != nil {
				wc.l.Error().Err(err).Msg("could not handle job task")
			}
		}(task)
	}

	return nil
}

func (wc *WorkflowsControllerImpl) handleTask(ctx context.Context, task *taskqueue.Task) error {
	switch task.ID {
	case "workflow-run-queued":
		return wc.handleWorkflowRunQueued(ctx, task)
	case "group-key-action-requeue-ticker":
		return wc.handleGroupKeyActionRequeue(ctx, task)
	case "get-group-key-run-started":
		return wc.handleGroupKeyRunStarted(ctx, task)
	case "get-group-key-run-finished":
		return wc.handleGroupKeyRunFinished(ctx, task)
	case "get-group-key-run-failed":
		return wc.handleGroupKeyRunFailed(ctx, task)
	case "workflow-run-finished":
		// return ec.handleStepRunStarted(ctx, task)
	}

	return fmt.Errorf("unknown task: %s in queue %s", task.ID, string(task.Queue))
}

func (ec *WorkflowsControllerImpl) handleGroupKeyRunStarted(ctx context.Context, task *taskqueue.Task) error {
	ctx, span := telemetry.NewSpan(ctx, "get-group-key-run-started")
	defer span.End()

	payload := tasktypes.GetGroupKeyRunStartedTaskPayload{}
	metadata := tasktypes.GetGroupKeyRunStartedTaskMetadata{}

	err := ec.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode step run started task payload: %w", err)
	}

	err = ec.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode step run started task metadata: %w", err)
	}

	// update the get group key run in the database
	startedAt, err := time.Parse(time.RFC3339, payload.StartedAt)

	if err != nil {
		return fmt.Errorf("could not parse started at: %w", err)
	}

	_, err = ec.repo.GetGroupKeyRun().UpdateGetGroupKeyRun(metadata.TenantId, payload.GetGroupKeyRunId, &repository.UpdateGetGroupKeyRunOpts{
		StartedAt: &startedAt,
		Status:    repository.StepRunStatusPtr(db.StepRunStatusRunning),
	})

	return err
}

func (wc *WorkflowsControllerImpl) handleGroupKeyRunFinished(ctx context.Context, task *taskqueue.Task) error {
	ctx, span := telemetry.NewSpan(ctx, "handle-step-run-finished")
	defer span.End()

	payload := tasktypes.GetGroupKeyRunFinishedTaskPayload{}
	metadata := tasktypes.GetGroupKeyRunFinishedTaskMetadata{}

	err := wc.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode step run started task payload: %w", err)
	}

	err = wc.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode step run started task metadata: %w", err)
	}

	// update the step run in the database
	finishedAt, err := time.Parse(time.RFC3339, payload.FinishedAt)

	if err != nil {
		return fmt.Errorf("could not parse started at: %w", err)
	}

	groupKeyRun, err := wc.repo.GetGroupKeyRun().UpdateGetGroupKeyRun(metadata.TenantId, payload.GetGroupKeyRunId, &repository.UpdateGetGroupKeyRunOpts{
		FinishedAt: &finishedAt,
		Status:     repository.StepRunStatusPtr(db.StepRunStatusSucceeded),
		Output:     &payload.GroupKey,
	})

	if err != nil {
		return fmt.Errorf("could not update step run: %w", err)
	}

	errGroup := new(errgroup.Group)

	errGroup.Go(func() error {
		workflowVersion, err := wc.repo.Workflow().GetWorkflowVersionById(metadata.TenantId, groupKeyRun.WorkflowRun().WorkflowVersionID)

		if err != nil {
			return fmt.Errorf("could not get workflow version: %w", err)
		}

		concurrency, _ := workflowVersion.Concurrency()

		switch concurrency.LimitStrategy {
		case db.ConcurrencyLimitStrategyCancelInProgress:
			err = wc.queueByCancelInProgress(ctx, metadata.TenantId, payload.GroupKey, workflowVersion)
		default:
			return fmt.Errorf("unimplemented concurrency limit strategy: %s", concurrency.LimitStrategy)
		}

		return err
	})

	// cancel the timeout task
	errGroup.Go(func() error {
		groupKeyRunTicker, ok := groupKeyRun.Ticker()

		if ok {
			err = wc.tq.AddTask(
				ctx,
				taskqueue.QueueTypeFromTickerID(groupKeyRunTicker.ID),
				cancelGetGroupKeyRunTimeoutTask(groupKeyRunTicker, groupKeyRun),
			)

			if err != nil {
				return fmt.Errorf("could not add cancel step run timeout task to task queue: %w", err)
			}
		}

		return nil
	})

	return errGroup.Wait()
}

func (wc *WorkflowsControllerImpl) handleGroupKeyRunFailed(ctx context.Context, task *taskqueue.Task) error {
	ctx, span := telemetry.NewSpan(ctx, "handle-group-key-run-failed")
	defer span.End()

	payload := tasktypes.GetGroupKeyRunFailedTaskPayload{}
	metadata := tasktypes.GetGroupKeyRunFailedTaskMetadata{}

	err := wc.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode step run started task payload: %w", err)
	}

	err = wc.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode step run started task metadata: %w", err)
	}

	// update the group key run in the database
	failedAt, err := time.Parse(time.RFC3339, payload.FailedAt)
	if err != nil {
		return fmt.Errorf("could not parse started at: %w", err)
	}

	groupKeyRun, err := wc.repo.GetGroupKeyRun().UpdateGetGroupKeyRun(metadata.TenantId, payload.GetGroupKeyRunId, &repository.UpdateGetGroupKeyRunOpts{
		FinishedAt: &failedAt,
		Error:      &payload.Error,
		Status:     repository.StepRunStatusPtr(db.StepRunStatusFailed),
	})

	if err != nil {
		return fmt.Errorf("could not update step run: %w", err)
	}

	// cancel the ticker for the step run
	getGroupKeyRunTicker, ok := groupKeyRun.Ticker()

	if ok {
		err = wc.tq.AddTask(
			ctx,
			taskqueue.QueueTypeFromTickerID(getGroupKeyRunTicker.ID),
			cancelGetGroupKeyRunTimeoutTask(getGroupKeyRunTicker, groupKeyRun),
		)

		if err != nil {
			return fmt.Errorf("could not add cancel step run timeout task to task queue: %w", err)
		}
	}

	return nil
}

func cancelGetGroupKeyRunTimeoutTask(ticker *db.TickerModel, getGroupKeyRun *db.GetGroupKeyRunModel) *taskqueue.Task {
	payload, _ := datautils.ToJSONMap(tasktypes.CancelGetGroupKeyRunTimeoutTaskPayload{
		GetGroupKeyRunId: getGroupKeyRun.ID,
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.CancelGetGroupKeyRunTimeoutTaskMetadata{
		TenantId: getGroupKeyRun.TenantID,
	})

	return &taskqueue.Task{
		ID:       "cancel-get-group-key-run-timeout",
		Queue:    taskqueue.QueueTypeFromTickerID(ticker.ID),
		Payload:  payload,
		Metadata: metadata,
	}
}
