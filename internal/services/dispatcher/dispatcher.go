package dispatcher

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
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/internal/taskqueue"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/internal/telemetry/servertel"
)

type Dispatcher interface {
	contracts.DispatcherServer
	Start() (func() error, error)
}

type DispatcherImpl struct {
	contracts.UnimplementedDispatcherServer

	s            gocron.Scheduler
	tq           taskqueue.TaskQueue
	l            *zerolog.Logger
	dv           datautils.DataDecoderValidator
	repo         repository.Repository
	dispatcherId string
	workers      sync.Map
}

type DispatcherOpt func(*DispatcherOpts)

type DispatcherOpts struct {
	tq           taskqueue.TaskQueue
	l            *zerolog.Logger
	dv           datautils.DataDecoderValidator
	repo         repository.Repository
	dispatcherId string
}

func defaultDispatcherOpts() *DispatcherOpts {
	logger := logger.NewDefaultLogger("dispatcher")
	return &DispatcherOpts{
		l:            &logger,
		dv:           datautils.NewDataDecoderValidator(),
		dispatcherId: uuid.New().String(),
	}
}

func WithTaskQueue(tq taskqueue.TaskQueue) DispatcherOpt {
	return func(opts *DispatcherOpts) {
		opts.tq = tq
	}
}

func WithRepository(r repository.Repository) DispatcherOpt {
	return func(opts *DispatcherOpts) {
		opts.repo = r
	}
}

func WithLogger(l *zerolog.Logger) DispatcherOpt {
	return func(opts *DispatcherOpts) {
		opts.l = l
	}
}

func WithDataDecoderValidator(dv datautils.DataDecoderValidator) DispatcherOpt {
	return func(opts *DispatcherOpts) {
		opts.dv = dv
	}
}

func WithDispatcherId(dispatcherId string) DispatcherOpt {
	return func(opts *DispatcherOpts) {
		opts.dispatcherId = dispatcherId
	}
}

func New(fs ...DispatcherOpt) (*DispatcherImpl, error) {
	opts := defaultDispatcherOpts()

	for _, f := range fs {
		f(opts)
	}

	if opts.tq == nil {
		return nil, fmt.Errorf("task queue is required. use WithTaskQueue")
	}

	if opts.repo == nil {
		return nil, fmt.Errorf("repository is required. use WithRepository")
	}

	newLogger := opts.l.With().Str("service", "dispatcher").Logger()
	opts.l = &newLogger

	// create a new scheduler
	s, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))

	if err != nil {
		return nil, fmt.Errorf("could not create scheduler for dispatcher: %w", err)
	}

	return &DispatcherImpl{
		tq:           opts.tq,
		l:            opts.l,
		dv:           opts.dv,
		repo:         opts.repo,
		dispatcherId: opts.dispatcherId,
		workers:      sync.Map{},
		s:            s,
	}, nil
}

func (d *DispatcherImpl) Start() (func() error, error) {
	// register the dispatcher by creating a new dispatcher in the database
	dispatcher, err := d.repo.Dispatcher().CreateNewDispatcher(&repository.CreateDispatcherOpts{
		ID: d.dispatcherId,
	})

	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	// subscribe to a task queue with the dispatcher id
	cleanupQueue, taskChan, err := d.tq.Subscribe(taskqueue.QueueTypeFromDispatcherID(dispatcher.ID))

	if err != nil {
		cancel()
		return nil, err
	}

	_, err = d.s.NewJob(
		gocron.DurationJob(time.Second*5),
		gocron.NewTask(
			d.runUpdateHeartbeat(ctx),
		),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not schedule heartbeat update: %w", err)
	}

	d.s.Start()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case task := <-taskChan:
				go func(task *taskqueue.Task) {
					err = d.handleTask(ctx, task)

					if err != nil {
						d.l.Error().Err(err).Msgf("could not handle dispatcher task %s", task.ID)
					}
				}(task)
			}
		}
	}()

	cleanup := func() error {
		d.l.Debug().Msgf("dispatcher is shutting down...")
		cancel()

		if err := cleanupQueue(); err != nil {
			return fmt.Errorf("could not cleanup queue: %w", err)
		}

		// drain the existing connections
		d.l.Debug().Msg("draining existing connections")

		d.workers.Range(func(key, value interface{}) bool {
			w := value.(subscribedWorker)

			w.finished <- true

			return true
		})

		err = d.repo.Dispatcher().Delete(dispatcher.ID)
		if err != nil {
			return fmt.Errorf("could not delete dispatcher: %w", err)
		}

		d.l.Debug().Msgf("deleted dispatcher %s", dispatcher.ID)

		if err := d.s.Shutdown(); err != nil {
			return fmt.Errorf("could not shutdown scheduler: %w", err)
		}

		d.l.Debug().Msgf("dispatcher has shutdown")
		return nil
	}

	return cleanup, nil
}

func (d *DispatcherImpl) handleTask(ctx context.Context, task *taskqueue.Task) error {
	switch task.ID {
	case "group-key-action-assigned":
		return d.handleGroupKeyActionAssignedTask(ctx, task)
	case "step-run-assigned":
		return d.handleStepRunAssignedTask(ctx, task)
	case "step-run-cancelled":
		return d.handleStepRunCancelled(ctx, task)
	}

	return fmt.Errorf("unknown task: %s", task.ID)
}

func (d *DispatcherImpl) handleGroupKeyActionAssignedTask(ctx context.Context, task *taskqueue.Task) error {
	ctx, span := telemetry.NewSpan(ctx, "group-key-action-assigned")
	defer span.End()

	payload := tasktypes.GroupKeyActionAssignedTaskPayload{}
	metadata := tasktypes.GroupKeyActionAssignedTaskMetadata{}

	err := d.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode dispatcher task payload: %w", err)
	}

	err = d.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode dispatcher task metadata: %w", err)
	}

	// get the worker for this task
	w, err := d.GetWorker(payload.WorkerId)

	if err != nil {
		return fmt.Errorf("could not get worker: %w", err)
	}

	telemetry.WithAttributes(span, servertel.WorkerId(payload.WorkerId))

	// load the workflow run from the database
	workflowRun, err := d.repo.WorkflowRun().GetWorkflowRunById(metadata.TenantId, payload.WorkflowRunId)

	if err != nil {
		return fmt.Errorf("could not get workflow run: %w", err)
	}

	servertel.WithWorkflowRunModel(span, workflowRun)

	err = w.StartGroupKeyAction(ctx, metadata.TenantId, workflowRun)

	if err != nil {
		return fmt.Errorf("could not send group key action to worker: %w", err)
	}

	return nil
}

func (d *DispatcherImpl) handleStepRunAssignedTask(ctx context.Context, task *taskqueue.Task) error {
	ctx, span := telemetry.NewSpan(ctx, "step-run-assigned")
	defer span.End()

	payload := tasktypes.StepRunAssignedTaskPayload{}
	metadata := tasktypes.StepRunAssignedTaskMetadata{}

	err := d.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode dispatcher task payload: %w", err)
	}

	err = d.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode dispatcher task metadata: %w", err)
	}

	// get the worker for this task
	w, err := d.GetWorker(payload.WorkerId)

	if err != nil {
		return fmt.Errorf("could not get worker: %w", err)
	}

	telemetry.WithAttributes(span, servertel.WorkerId(payload.WorkerId))

	// load the step run from the database
	stepRun, err := d.repo.StepRun().GetStepRunById(metadata.TenantId, payload.StepRunId)

	if err != nil {
		return fmt.Errorf("could not get step run: %w", err)
	}

	servertel.WithStepRunModel(span, stepRun)

	err = w.StartStepRun(ctx, metadata.TenantId, stepRun)

	if err != nil {
		return fmt.Errorf("could not send step action to worker: %w", err)
	}

	return nil
}

func (d *DispatcherImpl) handleStepRunCancelled(ctx context.Context, task *taskqueue.Task) error {
	ctx, span := telemetry.NewSpan(ctx, "step-run-cancelled")
	defer span.End()

	payload := tasktypes.StepRunCancelledTaskPayload{}
	metadata := tasktypes.StepRunCancelledTaskMetadata{}

	err := d.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode dispatcher task payload: %w", err)
	}

	err = d.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode dispatcher task metadata: %w", err)
	}

	// get the worker for this task
	w, err := d.GetWorker(payload.WorkerId)

	if err != nil {
		return fmt.Errorf("could not get worker: %w", err)
	}

	telemetry.WithAttributes(span, servertel.WorkerId(payload.WorkerId))

	// load the step run from the database
	stepRun, err := d.repo.StepRun().GetStepRunById(metadata.TenantId, payload.StepRunId)

	if err != nil {
		return fmt.Errorf("could not get step run: %w", err)
	}

	servertel.WithStepRunModel(span, stepRun)

	err = w.CancelStepRun(ctx, metadata.TenantId, stepRun)

	if err != nil {
		return fmt.Errorf("could not send job to worker: %w", err)
	}

	return nil
}

func (d *DispatcherImpl) runUpdateHeartbeat(ctx context.Context) func() {
	return func() {
		d.l.Debug().Msgf("dispatcher: updating heartbeat")

		now := time.Now().UTC()

		// update the heartbeat
		_, err := d.repo.Dispatcher().UpdateDispatcher(d.dispatcherId, &repository.UpdateDispatcherOpts{
			LastHeartbeatAt: &now,
		})

		if err != nil {
			d.l.Err(err).Msg("dispatcher: could not update heartbeat")
		}
	}
}
