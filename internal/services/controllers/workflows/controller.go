package workflows

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/logger"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/taskqueue"
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

	newLogger := opts.l.With().Str("service", "jobs-controller").Logger()
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
	case "group-key-action-started":
		// return ec.handleJobRunTimedOut(ctx, task)
	case "group-key-action-finished":
		// return ec.handleStepRunQueued(ctx, task)
	case "group-key-action-errored":
		// return ec.handleStepRunRequeue(ctx, task)
	case "workflow-run-finished":
		// return ec.handleStepRunStarted(ctx, task)
	}

	return fmt.Errorf("unknown task: %s in queue %s", task.ID, string(task.Queue))
}
