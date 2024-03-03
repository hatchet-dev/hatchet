package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/logger"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/internal/schema"
	"github.com/hatchet-dev/hatchet/internal/services/shared/defaults"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/internal/taskqueue"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/internal/telemetry/servertel"
)

type JobsController interface {
	Start(ctx context.Context) error
}

type JobsControllerImpl struct {
	tq   taskqueue.TaskQueue
	l    *zerolog.Logger
	repo repository.Repository
	dv   datautils.DataDecoderValidator
}

type JobsControllerOpt func(*JobsControllerOpts)

type JobsControllerOpts struct {
	tq   taskqueue.TaskQueue
	l    *zerolog.Logger
	repo repository.Repository
	dv   datautils.DataDecoderValidator
}

func defaultJobsControllerOpts() *JobsControllerOpts {
	logger := logger.NewDefaultLogger("jobs-controller")
	return &JobsControllerOpts{
		l:  &logger,
		dv: datautils.NewDataDecoderValidator(),
	}
}

func WithTaskQueue(tq taskqueue.TaskQueue) JobsControllerOpt {
	return func(opts *JobsControllerOpts) {
		opts.tq = tq
	}
}

func WithLogger(l *zerolog.Logger) JobsControllerOpt {
	return func(opts *JobsControllerOpts) {
		opts.l = l
	}
}

func WithRepository(r repository.Repository) JobsControllerOpt {
	return func(opts *JobsControllerOpts) {
		opts.repo = r
	}
}

func WithDataDecoderValidator(dv datautils.DataDecoderValidator) JobsControllerOpt {
	return func(opts *JobsControllerOpts) {
		opts.dv = dv
	}
}

func New(fs ...JobsControllerOpt) (*JobsControllerImpl, error) {
	opts := defaultJobsControllerOpts()

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

	return &JobsControllerImpl{
		tq:   opts.tq,
		l:    opts.l,
		repo: opts.repo,
		dv:   opts.dv,
	}, nil
}

func (jc *JobsControllerImpl) Start() (func() error, error) {
	cleanupQueue, taskChan, err := jc.tq.Subscribe(taskqueue.JOB_PROCESSING_QUEUE)

	if err != nil {
		return nil, fmt.Errorf("could not subscribe to job processing queue: %w", err)
	}

	wg := sync.WaitGroup{}

	go func() {
		for task := range taskChan {
			wg.Add(1)
			go func(task *taskqueue.Task) {
				defer wg.Done()

				err := jc.handleTask(context.Background(), task)
				if err != nil {
					jc.l.Error().Err(err).Msg("could not handle job task")
				}
			}(task)
		}
	}()

	cleanup := func() error {
		if err := cleanupQueue(); err != nil {
			return fmt.Errorf("could not cleanup job processing queue: %w", err)
		}

		wg.Wait()

		return nil
	}

	return cleanup, nil
}

func (ec *JobsControllerImpl) handleTask(ctx context.Context, task *taskqueue.Task) error {
	switch task.ID {
	case "job-run-queued":
		return ec.handleJobRunQueued(ctx, task)
	case "job-run-timed-out":
		return ec.handleJobRunTimedOut(ctx, task)
	case "step-run-retry":
		return ec.handleStepRunRetry(ctx, task)
	case "step-run-queued":
		return ec.handleStepRunQueued(ctx, task)
	case "step-run-requeue-ticker":
		return ec.handleStepRunRequeue(ctx, task)
	case "step-run-started":
		return ec.handleStepRunStarted(ctx, task)
	case "step-run-finished":
		return ec.handleStepRunFinished(ctx, task)
	case "step-run-failed":
		return ec.handleStepRunFailed(ctx, task)
	case "step-run-cancelled":
		return ec.handleStepRunCancelled(ctx, task)
	case "step-run-timed-out":
		return ec.handleStepRunTimedOut(ctx, task)
	case "ticker-removed":
		return ec.handleTickerRemoved(ctx, task)
	}

	return fmt.Errorf("unknown task: %s", task.ID)
}

func (ec *JobsControllerImpl) handleJobRunQueued(ctx context.Context, task *taskqueue.Task) error {
	ctx, span := telemetry.NewSpan(ctx, "handle-job-run-queued")
	defer span.End()

	payload := tasktypes.JobRunQueuedTaskPayload{}
	metadata := tasktypes.JobRunQueuedTaskMetadata{}

	err := ec.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode job task payload: %w", err)
	}

	err = ec.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode job task metadata: %w", err)
	}

	// get the job run in the database
	jobRun, err := ec.repo.JobRun().GetJobRunById(metadata.TenantId, payload.JobRunId)

	if err != nil {
		return fmt.Errorf("could not get job run: %w", err)
	}

	servertel.WithJobRunModel(span, jobRun)

	// schedule the first step in the job run
	stepRuns := jobRun.StepRuns()

	if len(stepRuns) == 0 {
		return fmt.Errorf("job run has no step runs")
	}

	// list the step runs without a parent
	for _, stepRun := range stepRuns {
		stepRunCp := stepRun

		if len(stepRunCp.Parents()) == 0 {
			// send a task to the taskqueue
			err = ec.tq.AddTask(
				ctx,
				taskqueue.JOB_PROCESSING_QUEUE,
				tasktypes.StepRunQueuedToTask(
					jobRun.Job(),
					&stepRunCp,
				),
			)

			if err != nil {
				return fmt.Errorf("could not add job queued task to task queue: %w", err)
			}
		}
	}

	// send a task to schedule the job run's timeout
	tickers, err := ec.getValidTickers()

	if err != nil {
		return err
	}

	ticker := &tickers[0]

	ticker, err = ec.repo.Ticker().AddJobRun(ticker.ID, jobRun)

	if err != nil {
		return fmt.Errorf("could not add job run to ticker: %w", err)
	}

	scheduleTimeoutTask, err := scheduleJobRunTimeoutTask(ticker, jobRun)

	if err != nil {
		return fmt.Errorf("could not schedule job run timeout task: %w", err)
	}

	err = ec.tq.AddTask(
		ctx,
		taskqueue.QueueTypeFromTickerID(ticker.ID),
		scheduleTimeoutTask,
	)

	if err != nil {
		return fmt.Errorf("could not add schedule job run timeout task to task queue: %w", err)
	}

	return nil
}

func (ec *JobsControllerImpl) handleJobRunTimedOut(ctx context.Context, task *taskqueue.Task) error {
	ctx, span := telemetry.NewSpan(ctx, "handle-job-run-timed-out")
	defer span.End()

	payload := tasktypes.JobRunTimedOutTaskPayload{}
	metadata := tasktypes.JobRunTimedOutTaskMetadata{}

	err := ec.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode job task payload: %w", err)
	}

	err = ec.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode job task metadata: %w", err)
	}

	// get the job run in the database
	jobRun, err := ec.repo.JobRun().GetJobRunById(metadata.TenantId, payload.JobRunId)

	if err != nil {
		return fmt.Errorf("could not get job run: %w", err)
	}

	servertel.WithJobRunModel(span, jobRun)

	stepRuns, err := ec.repo.StepRun().ListStepRuns(metadata.TenantId, &repository.ListStepRunsOpts{
		JobRunId: &jobRun.ID,
		Status:   repository.StepRunStatusPtr(db.StepRunStatusRunning),
	})

	if err != nil {
		return fmt.Errorf("could not list step runs: %w", err)
	}

	if len(stepRuns) > 1 {
		return fmt.Errorf("job run has multiple running step runs")
	}

	if len(stepRuns) == 1 {
		currStepRun := stepRuns[0]

		// cancel current step run
		now := time.Now().UTC()

		// cancel current step run
		stepRun, updateInfo, err := ec.repo.StepRun().UpdateStepRun(metadata.TenantId, currStepRun.ID, &repository.UpdateStepRunOpts{
			CancelledAt:     &now,
			CancelledReason: repository.StringPtr("JOB_RUN_TIMED_OUT"),
			Status:          repository.StepRunStatusPtr(db.StepRunStatusCancelled),
		})

		if err != nil {
			return fmt.Errorf("could not update step run: %w", err)
		}

		defer ec.handleStepRunUpdateInfo(stepRun, updateInfo)

		workerId, ok := stepRun.WorkerID()

		if !ok {
			return fmt.Errorf("step run has no worker id")
		}

		worker, err := ec.repo.Worker().GetWorkerById(workerId)

		if err != nil {
			return fmt.Errorf("could not get worker: %w", err)
		}

		dispatcherId, ok := worker.DispatcherID()

		// send a task to the taskqueue
		if ok {
			err = ec.tq.AddTask(
				ctx,
				taskqueue.QueueTypeFromDispatcherID(dispatcherId),
				stepRunCancelledTask(metadata.TenantId, currStepRun.ID, worker.ID, dispatcherId, "JOB_RUN_TIMED_OUT"),
			)

			if err != nil {
				return fmt.Errorf("could not add job assigned task to task queue: %w", err)
			}
		}

		// cancel the ticker for the step run
		stepRunTicker, ok := stepRun.Ticker()

		if ok {
			err = ec.tq.AddTask(
				ctx,
				taskqueue.QueueTypeFromTickerID(stepRunTicker.ID),
				cancelStepRunTimeoutTask(stepRunTicker, stepRun),
			)

			if err != nil {
				return fmt.Errorf("could not add cancel step run timeout task to task queue: %w", err)
			}
		}
	}

	return nil
}

func (ec *JobsControllerImpl) handleStepRunRetry(ctx context.Context, task *taskqueue.Task) error {
	ctx, span := telemetry.NewSpan(ctx, "handle-step-run-retry")
	defer span.End()

	payload := tasktypes.StepRunRetryTaskPayload{}
	metadata := tasktypes.StepRunRetryTaskMetadata{}

	err := ec.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode job task payload: %w", err)
	}

	err = ec.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode job task metadata: %w", err)
	}

	err = ec.repo.StepRun().ArchiveStepRunResult(metadata.TenantId, payload.StepRunId)

	if err != nil {
		return fmt.Errorf("could not archive step run result: %w", err)
	}

	ec.l.Error().Err(fmt.Errorf("starting step run retry"))

	stepRun, err := ec.repo.StepRun().GetStepRunById(metadata.TenantId, payload.StepRunId)

	if err != nil {
		return fmt.Errorf("could not get step run: %w", err)
	}

	var inputBytes []byte
	var retryCount = stepRun.RetryCount + 1

	// update the input schema for the step run based on the new input
	if payload.InputData != "" {
		inputBytes = []byte(payload.InputData)

		// Merge the existing input data with the new input data. We don't blindly trust the
		// input data because the user could have deleted fields that are required by the step.
		// A better solution would be to validate the user input ahead of time.
		// NOTE: this is an expensive operation.
		if currentInput, ok := stepRun.Input(); ok {
			inputMap, err := datautils.JSONBytesToMap([]byte(payload.InputData))

			if err != nil {
				return fmt.Errorf("could not convert input data to map: %w", err)
			}

			currentInputMap, err := datautils.JSONBytesToMap(currentInput)

			if err != nil {
				return fmt.Errorf("could not convert current input to map: %w", err)
			}

			mergedInput := datautils.MergeMaps(currentInputMap, inputMap)

			mergedInputBytes, err := json.Marshal(mergedInput)

			if err != nil {
				return fmt.Errorf("could not marshal merged input: %w", err)
			}

			inputBytes = mergedInputBytes
		}

		jsonSchemaBytes, err := schema.SchemaBytesFromBytes(inputBytes)

		if err != nil {
			return err
		}

		_, err = ec.repo.StepRun().UpdateStepRunInputSchema(metadata.TenantId, stepRun.ID, jsonSchemaBytes)

		if err != nil {
			return err
		}

		// if the input data has been manually set, we reset the retry count as this is a user-triggered retry
		retryCount = 0
	} else {
		var ok bool

		inputBytes, ok = stepRun.Input()

		if !ok {
			return fmt.Errorf("could not get step run input: %w", err)
		}
	}

	// update step run
	_, _, err = ec.repo.StepRun().UpdateStepRun(metadata.TenantId, stepRun.ID, &repository.UpdateStepRunOpts{
		Input:      inputBytes,
		Status:     repository.StepRunStatusPtr(db.StepRunStatusPending),
		IsRerun:    true,
		RetryCount: &retryCount,
	})

	if err != nil {
		return fmt.Errorf("could not update step run: %w", err)
	}

	// requeue the step run in the task queue
	jobRun, err := ec.repo.JobRun().GetJobRunById(metadata.TenantId, stepRun.JobRunID)

	if err != nil {
		return fmt.Errorf("could not get job run: %w", err)
	}

	// send a task to the taskqueue
	return ec.tq.AddTask(
		ctx,
		taskqueue.JOB_PROCESSING_QUEUE,
		tasktypes.StepRunQueuedToTask(jobRun.Job(), stepRun),
	)
}

func (ec *JobsControllerImpl) handleStepRunQueued(ctx context.Context, task *taskqueue.Task) error {
	ctx, span := telemetry.NewSpan(ctx, "handle-step-run-queued")
	defer span.End()

	payload := tasktypes.StepRunTaskPayload{}
	metadata := tasktypes.StepRunTaskMetadata{}

	err := ec.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode job task payload: %w", err)
	}

	err = ec.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode job task metadata: %w", err)
	}

	return ec.queueStepRun(ctx, metadata.TenantId, metadata.StepId, payload.StepRunId)
}

// handleStepRunRequeue looks for any step runs that haven't been assigned that are past their requeue time
func (ec *JobsControllerImpl) handleStepRunRequeue(ctx context.Context, task *taskqueue.Task) error {
	ctx, span := telemetry.NewSpan(ctx, "handle-step-run-requeue")
	defer span.End()

	payload := tasktypes.StepRunRequeueTaskPayload{}
	metadata := tasktypes.StepRunRequeueTaskMetadata{}

	err := ec.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode step run requeue task payload: %w", err)
	}

	err = ec.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode step run requeue task metadata: %w", err)
	}

	stepRuns, err := ec.repo.StepRun().ListStepRunsToRequeue(payload.TenantId)

	if err != nil {
		return fmt.Errorf("could not list step runs: %w", err)
	}

	g := new(errgroup.Group)

	for i := range stepRuns {
		stepRunCp := stepRuns[i]

		// wrap in func to get defer on the span to avoid leaking spans
		g.Go(func() error {
			var innerStepRun *db.StepRunModel

			ctx, span := telemetry.NewSpan(ctx, "handle-step-run-requeue-step-run")
			defer span.End()

			stepRunId := sqlchelpers.UUIDToStr(stepRunCp.ID)

			ec.l.Debug().Msgf("requeueing step run %s", stepRunId)

			now := time.Now().UTC().UTC()

			// if the current time is after the scheduleTimeoutAt, then mark this as timed out
			scheduleTimeoutAt := stepRunCp.ScheduleTimeoutAt.Time

			// timed out if there was no scheduleTimeoutAt set and the current time is after the step run created at time plus the default schedule timeout,
			// or if the scheduleTimeoutAt is set and the current time is after the scheduleTimeoutAt
			isTimedOut := !scheduleTimeoutAt.IsZero() && scheduleTimeoutAt.Before(now)

			if isTimedOut {
				var updateInfo *repository.StepRunUpdateInfo

				innerStepRun, updateInfo, err = ec.repo.StepRun().UpdateStepRun(payload.TenantId, stepRunId, &repository.UpdateStepRunOpts{
					CancelledAt:     &now,
					CancelledReason: repository.StringPtr("SCHEDULING_TIMED_OUT"),
					Status:          repository.StepRunStatusPtr(db.StepRunStatusCancelled),
				})

				if err != nil {
					return fmt.Errorf("could not update step run %s: %w", stepRunId, err)
				}

				defer ec.handleStepRunUpdateInfo(innerStepRun, updateInfo)

				return nil
			}

			requeueAfter := time.Now().UTC().Add(time.Second * 5)

			innerStepRun, _, err := ec.repo.StepRun().UpdateStepRun(payload.TenantId, stepRunId, &repository.UpdateStepRunOpts{
				RequeueAfter: &requeueAfter,
			})

			if err != nil {
				return fmt.Errorf("could not update step run %s: %w", stepRunId, err)
			}

			return ec.scheduleStepRun(ctx, payload.TenantId, innerStepRun.StepID, innerStepRun.ID)
		})
	}

	return g.Wait()
}

func (ec *JobsControllerImpl) queueStepRun(ctx context.Context, tenantId, stepId, stepRunId string) error {
	ctx, span := telemetry.NewSpan(ctx, "queue-step-run")
	defer span.End()

	// add the rendered data to the step run
	stepRun, err := ec.repo.StepRun().GetStepRunById(tenantId, stepRunId)

	if err != nil {
		return fmt.Errorf("could not get step run: %w", err)
	}

	servertel.WithStepRunModel(span, stepRun)

	updateStepOpts := &repository.UpdateStepRunOpts{}

	// set scheduling timeout
	if scheduleTimeoutAt, ok := stepRun.ScheduleTimeoutAt(); !ok || scheduleTimeoutAt.IsZero() {
		var timeoutDuration time.Duration

		// get the schedule timeout from the step
		stepScheduleTimeout := stepRun.Step().ScheduleTimeout

		if stepScheduleTimeout != "" {
			timeoutDuration, _ = time.ParseDuration(stepScheduleTimeout)
		} else {
			timeoutDuration = defaults.DefaultScheduleTimeout
		}

		scheduleTimeoutAt := time.Now().UTC().Add(timeoutDuration)

		updateStepOpts.ScheduleTimeoutAt = &scheduleTimeoutAt
	}

	// If the step run input is not set, then we should set it. This will be set upstream if we've rerun
	// the step run manually with new inputs. It will not be set when the step is automatically queued.
	if in, ok := stepRun.Input(); !ok || string(json.RawMessage(in)) == "{}" {
		lookupDataModel, ok := stepRun.JobRun().LookupData()

		if ok && lookupDataModel != nil {
			data, ok := lookupDataModel.Data()

			if !ok {
				return fmt.Errorf("job run has no lookup data")
			}

			lookupData := &datautils.JobRunLookupData{}

			err := datautils.FromJSONType(&data, lookupData)

			if err != nil {
				return fmt.Errorf("could not get job run lookup data: %w", err)
			}

			userData := map[string]interface{}{}

			if setUserData, ok := stepRun.Step().CustomUserData(); ok {
				err := json.Unmarshal(setUserData, &userData)

				if err != nil {
					return fmt.Errorf("could not unmarshal custom user data: %w", err)
				}
			}

			// input data is the triggering event data and any parent step data
			inputData := datautils.StepRunData{
				Input:       lookupData.Input,
				TriggeredBy: lookupData.TriggeredBy,
				Parents:     map[string]datautils.StepData{},
				UserData:    userData,
				Overrides:   map[string]interface{}{},
			}

			// add all parents to the input data
			for _, parent := range stepRun.Parents() {
				readableId, ok := parent.Step().ReadableID()

				if ok && readableId != "" {
					parentData, exists := lookupData.Steps[readableId]

					if exists {
						inputData.Parents[readableId] = parentData
					}
				}
			}

			inputDataBytes, err := json.Marshal(inputData)

			if err != nil {
				return fmt.Errorf("could not convert input data to json: %w", err)
			}

			// defer the update of the input schema to the step run
			defer func() {
				jsonSchemaBytes, err := schema.SchemaBytesFromBytes(inputDataBytes)

				if err != nil {
					ec.l.Err(err).Msgf("could not get schema bytes from bytes: %s", err.Error())
					return
				}

				_, err = ec.repo.StepRun().UpdateStepRunInputSchema(stepRun.TenantID, stepRun.ID, jsonSchemaBytes)

				if err != nil {
					ec.l.Err(err).Msgf("could not update step run input schema: %s", err.Error())
					return
				}
			}()

			updateStepOpts.Input = inputDataBytes
		}
	}

	// begin transaction and make sure step run is in a pending status
	// if the step run is no longer is a pending status, we should return with no error
	updateStepOpts.Status = repository.StepRunStatusPtr(db.StepRunStatusPendingAssignment)

	// indicate that the step run is pending assignment
	_, err = ec.repo.StepRun().QueueStepRun(tenantId, stepRunId, updateStepOpts)

	if err != nil {
		if errors.Is(err, repository.ErrStepRunIsNotPending) {
			ec.l.Debug().Msgf("step run %s is not pending, skipping scheduling", stepRunId)
			return nil
		}

		return fmt.Errorf("could not update step run: %w", err)
	}

	return ec.scheduleStepRun(ctx, tenantId, stepId, stepRunId)
}

func (ec *JobsControllerImpl) scheduleStepRun(ctx context.Context, tenantId, stepId, stepRunId string) error {
	ctx, span := telemetry.NewSpan(ctx, "schedule-step-run")
	defer span.End()

	stepRun, err := ec.repo.StepRun().GetStepRunById(tenantId, stepRunId)

	if err != nil {
		return fmt.Errorf("could not get step run: %w", err)
	}

	servertel.WithStepRunModel(span, stepRun)

	// Assign the step run to a worker.
	//
	// 1. Get a list of workers that can run this step. If there are no workers available, then simply return with
	//    no additional transactions and this step run will be requeued.
	// 2. Pick a worker to run the step and get the dispatcher currently connected to this worker.
	// 3. Update the step run's designated worker.
	//
	// After creating the worker, send a task to the taskqueue, which will be picked up by the dispatcher.
	after := time.Now().UTC().Add(-6 * time.Second)

	workers, err := ec.repo.Worker().ListWorkers(tenantId, &repository.ListWorkersOpts{
		Action:             &stepRun.Step().ActionID,
		LastHeartbeatAfter: &after,
		Assignable:         repository.BoolPtr(true),
	})

	if err != nil {
		return fmt.Errorf("could not list workers for step: %w", err)
	}

	if len(workers) == 0 {
		ec.l.Info().Msgf("no workers available for step %s; requeuing", stepId)
		return nil
	}

	// pick the worker with the least jobs currently assigned (this heuristic can and should change)
	selectedWorker := workers[0]

	for _, worker := range workers {
		if worker.RunningStepRuns < selectedWorker.RunningStepRuns {
			selectedWorker = worker
		}
	}

	selectedWorkerId := sqlchelpers.UUIDToStr(selectedWorker.Worker.ID)

	telemetry.WithAttributes(span, servertel.WorkerId(selectedWorkerId))

	// update the job run's designated worker
	err = ec.repo.Worker().AddStepRun(tenantId, selectedWorkerId, stepRunId)

	if err != nil {
		return fmt.Errorf("could not add step run to worker: %w", err)
	}

	// pick a ticker to use for timeout
	tickers, err := ec.getValidTickers()

	if err != nil {
		return err
	}

	ticker := &tickers[0]

	ticker, err = ec.repo.Ticker().AddStepRun(ticker.ID, stepRunId)

	if err != nil {
		return fmt.Errorf("could not add step run to ticker: %w", err)
	}

	scheduleTimeoutTask, err := scheduleStepRunTimeoutTask(ticker, stepRun)

	if err != nil {
		return fmt.Errorf("could not schedule step run timeout task: %w", err)
	}

	dispatcherId := sqlchelpers.UUIDToStr(selectedWorker.Worker.DispatcherId)

	// send a task to the dispatcher
	err = ec.tq.AddTask(
		ctx,
		taskqueue.QueueTypeFromDispatcherID(dispatcherId),
		stepRunAssignedTask(tenantId, stepRunId, selectedWorkerId, dispatcherId),
	)

	if err != nil {
		return fmt.Errorf("could not add job assigned task to task queue: %w", err)
	}

	// send a task to the ticker
	err = ec.tq.AddTask(
		ctx,
		taskqueue.QueueTypeFromTickerID(ticker.ID),
		scheduleTimeoutTask,
	)

	if err != nil {
		return fmt.Errorf("could not add schedule step run timeout task to task queue: %w", err)
	}

	return nil
}

func (ec *JobsControllerImpl) handleStepRunStarted(ctx context.Context, task *taskqueue.Task) error {
	ctx, span := telemetry.NewSpan(ctx, "handle-step-run-started")
	defer span.End()

	payload := tasktypes.StepRunStartedTaskPayload{}
	metadata := tasktypes.StepRunStartedTaskMetadata{}

	err := ec.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode step run started task payload: %w", err)
	}

	err = ec.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode step run started task metadata: %w", err)
	}

	// update the step run in the database
	startedAt, err := time.Parse(time.RFC3339, payload.StartedAt)

	if err != nil {
		return fmt.Errorf("could not parse started at: %w", err)
	}

	stepRun, updateInfo, err := ec.repo.StepRun().UpdateStepRun(metadata.TenantId, payload.StepRunId, &repository.UpdateStepRunOpts{
		StartedAt: &startedAt,
		Status:    repository.StepRunStatusPtr(db.StepRunStatusRunning),
	})

	if err != nil {
		return fmt.Errorf("could not update step run: %w", err)
	}

	defer ec.handleStepRunUpdateInfo(stepRun, updateInfo)

	return nil
}

func (ec *JobsControllerImpl) handleStepRunFinished(ctx context.Context, task *taskqueue.Task) error {
	ctx, span := telemetry.NewSpan(ctx, "handle-step-run-finished")
	defer span.End()

	payload := tasktypes.StepRunFinishedTaskPayload{}
	metadata := tasktypes.StepRunFinishedTaskMetadata{}

	err := ec.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode step run started task payload: %w", err)
	}

	err = ec.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode step run started task metadata: %w", err)
	}

	// update the step run in the database
	finishedAt, err := time.Parse(time.RFC3339, payload.FinishedAt)

	if err != nil {
		return fmt.Errorf("could not parse started at: %w", err)
	}

	var stepOutput []byte

	if payload.StepOutputData != "" {
		stepOutputStr, err := strconv.Unquote(payload.StepOutputData)

		if err != nil {
			stepOutputStr = payload.StepOutputData
		}

		stepOutput = []byte(stepOutputStr)
	}

	stepRun, updateInfo, err := ec.repo.StepRun().UpdateStepRun(metadata.TenantId, payload.StepRunId, &repository.UpdateStepRunOpts{
		FinishedAt: &finishedAt,
		Status:     repository.StepRunStatusPtr(db.StepRunStatusSucceeded),
		Output:     stepOutput,
	})

	if err != nil {
		return fmt.Errorf("could not update step run: %w", err)
	}

	defer ec.handleStepRunUpdateInfo(stepRun, updateInfo)

	servertel.WithStepRunModel(span, stepRun)

	jobRun, err := ec.repo.JobRun().GetJobRunById(metadata.TenantId, stepRun.JobRunID)

	if err != nil {
		return fmt.Errorf("could not get job run: %w", err)
	}

	servertel.WithJobRunModel(span, jobRun)

	// queue the next step runs
	nextStepRuns, err := ec.repo.StepRun().ListStartableStepRuns(metadata.TenantId, jobRun.ID, stepRun.ID)

	if err != nil {
		return fmt.Errorf("could not list startable step runs: %w", err)
	}

	for _, nextStepRun := range nextStepRuns {
		err = ec.queueStepRun(ctx, metadata.TenantId, sqlchelpers.UUIDToStr(nextStepRun.StepId), sqlchelpers.UUIDToStr(nextStepRun.ID))

		if err != nil {
			return fmt.Errorf("could not queue next step run: %w", err)
		}
	}

	// cancel the timeout task
	stepRunTicker, ok := stepRun.Ticker()

	if ok {
		err = ec.tq.AddTask(
			ctx,
			taskqueue.QueueTypeFromTickerID(stepRunTicker.ID),
			cancelStepRunTimeoutTask(stepRunTicker, stepRun),
		)

		if err != nil {
			return fmt.Errorf("could not add cancel step run timeout task to task queue: %w", err)
		}
	}

	return nil
}

func (ec *JobsControllerImpl) handleStepRunFailed(ctx context.Context, task *taskqueue.Task) error {
	ctx, span := telemetry.NewSpan(ctx, "handle-step-run-failed")
	defer span.End()

	payload := tasktypes.StepRunFailedTaskPayload{}
	metadata := tasktypes.StepRunFailedTaskMetadata{}

	err := ec.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode step run started task payload: %w", err)
	}

	err = ec.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode step run started task metadata: %w", err)
	}

	// update the step run in the database
	failedAt, err := time.Parse(time.RFC3339, payload.FailedAt)
	if err != nil {
		return fmt.Errorf("could not parse started at: %w", err)
	}

	stepRun, err := ec.repo.StepRun().GetStepRunById(metadata.TenantId, payload.StepRunId)

	if err != nil {
		return fmt.Errorf("could not get step run: %w", err)
	}

	// determine if step run should be retried or not
	shouldRetry := stepRun.RetryCount < stepRun.Step().Retries

	status := db.StepRunStatusFailed

	if shouldRetry {
		status = db.StepRunStatusPending
	}

	stepRun, updateInfo, err := ec.repo.StepRun().UpdateStepRun(metadata.TenantId, payload.StepRunId, &repository.UpdateStepRunOpts{
		FinishedAt: &failedAt,
		Error:      &payload.Error,
		Status:     repository.StepRunStatusPtr(status),
	})

	if err != nil {
		return fmt.Errorf("could not update step run: %w", err)
	}

	defer ec.handleStepRunUpdateInfo(stepRun, updateInfo)

	servertel.WithStepRunModel(span, stepRun)

	// cancel the ticker for the step run
	stepRunTicker, ok := stepRun.Ticker()

	if ok {
		err = ec.tq.AddTask(
			ctx,
			taskqueue.QueueTypeFromTickerID(stepRunTicker.ID),
			cancelStepRunTimeoutTask(stepRunTicker, stepRun),
		)

		if err != nil {
			return fmt.Errorf("could not add cancel step run timeout task to task queue: %w", err)
		}
	}

	if shouldRetry {
		// send a task to the taskqueue
		return ec.tq.AddTask(
			ctx,
			taskqueue.JOB_PROCESSING_QUEUE,
			tasktypes.StepRunRetryToTask(stepRun, nil),
		)
	}

	return nil
}

func (ec *JobsControllerImpl) handleStepRunTimedOut(ctx context.Context, task *taskqueue.Task) error {
	ctx, span := telemetry.NewSpan(ctx, "handle-step-run-timed-out")
	defer span.End()

	payload := tasktypes.StepRunTimedOutTaskPayload{}
	metadata := tasktypes.StepRunTimedOutTaskMetadata{}

	err := ec.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode step run started task payload: %w", err)
	}

	err = ec.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode step run started task metadata: %w", err)
	}

	return ec.cancelStepRun(ctx, metadata.TenantId, payload.StepRunId, "TIMED_OUT")
}

func (ec *JobsControllerImpl) handleStepRunCancelled(ctx context.Context, task *taskqueue.Task) error {
	ctx, span := telemetry.NewSpan(ctx, "handle-step-run-cancelled")
	defer span.End()

	payload := tasktypes.StepRunNotifyCancelTaskPayload{}
	metadata := tasktypes.StepRunNotifyCancelTaskMetadata{}

	err := ec.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode step run notify cancel task payload: %w", err)
	}

	err = ec.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode step run notify cancel task metadata: %w", err)
	}

	return ec.cancelStepRun(ctx, metadata.TenantId, payload.StepRunId, payload.CancelledReason)
}

func (ec *JobsControllerImpl) cancelStepRun(ctx context.Context, tenantId, stepRunId, reason string) error {
	ctx, span := telemetry.NewSpan(ctx, "cancel-step-run")
	defer span.End()

	// cancel current step run
	now := time.Now().UTC()

	stepRun, updateInfo, err := ec.repo.StepRun().UpdateStepRun(tenantId, stepRunId, &repository.UpdateStepRunOpts{
		CancelledAt:     &now,
		CancelledReason: repository.StringPtr(reason),
		Status:          repository.StepRunStatusPtr(db.StepRunStatusCancelled),
	})

	if err != nil {
		return fmt.Errorf("could not update step run: %w", err)
	}

	defer ec.handleStepRunUpdateInfo(stepRun, updateInfo)

	servertel.WithStepRunModel(span, stepRun)

	workerId, ok := stepRun.WorkerID()

	if !ok {
		return fmt.Errorf("step run has no worker id")
	}

	worker, err := ec.repo.Worker().GetWorkerById(workerId)

	if err != nil {
		return fmt.Errorf("could not get worker: %w", err)
	}

	dispatcherId, ok := worker.DispatcherID()

	// send a task to the taskqueue
	if ok {
		err = ec.tq.AddTask(
			ctx,
			taskqueue.QueueTypeFromDispatcherID(dispatcherId),
			stepRunCancelledTask(tenantId, stepRunId, worker.ID, dispatcherId, reason),
		)

		if err != nil {
			return fmt.Errorf("could not add job assigned task to task queue: %w", err)
		}
	}

	return nil
}

func (ec *JobsControllerImpl) handleStepRunUpdateInfo(stepRun *db.StepRunModel, updateInfo *repository.StepRunUpdateInfo) {
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)

			if !ok {
				err = fmt.Errorf("%v", r)
			}

			ec.l.Error().Err(err).Msg("recovered from panic")

			return
		}
	}()

	if updateInfo.WorkflowRunFinalState {
		err := ec.tq.AddTask(
			context.Background(),
			taskqueue.WORKFLOW_PROCESSING_QUEUE,
			tasktypes.WorkflowRunFinishedToTask(stepRun.TenantID, updateInfo.WorkflowRunId, updateInfo.WorkflowRunStatus),
		)

		if err != nil {
			ec.l.Error().Err(err).Msg("could not add workflow run finished task to task queue")
		}
	}
}

func (ec *JobsControllerImpl) handleTickerRemoved(ctx context.Context, task *taskqueue.Task) error {
	ctx, span := telemetry.NewSpan(ctx, "handle-ticker-removed")
	defer span.End()

	payload := tasktypes.RemoveTickerTaskPayload{}
	metadata := tasktypes.RemoveTickerTaskMetadata{}

	err := ec.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode ticker removed task payload: %w", err)
	}

	err = ec.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode ticker removed task metadata: %w", err)
	}

	ec.l.Debug().Msgf("handling ticker removed for ticker %s", payload.TickerId)

	// reassign all step runs to a different ticker
	tickers, err := ec.getValidTickers()

	if err != nil {
		return err
	}

	// reassign all step runs randomly to tickers
	numTickers := len(tickers)

	// get all step runs assigned to the ticker
	stepRuns, err := ec.repo.StepRun().ListAllStepRuns(&repository.ListAllStepRunsOpts{
		TickerId: repository.StringPtr(payload.TickerId),
		Status:   repository.StepRunStatusPtr(db.StepRunStatusRunning),
	})

	if err != nil {
		return fmt.Errorf("could not list step runs: %w", err)
	}

	for i, stepRun := range stepRuns {
		stepRunCp := stepRun
		ticker := tickers[i%numTickers]

		_, err = ec.repo.Ticker().AddStepRun(ticker.ID, stepRun.ID)

		if err != nil {
			return fmt.Errorf("could not update step run: %w", err)
		}

		scheduleTimeoutTask, err := scheduleStepRunTimeoutTask(&ticker, &stepRunCp)

		if err != nil {
			return fmt.Errorf("could not schedule step run timeout task: %w", err)
		}

		// send a task to the ticker
		err = ec.tq.AddTask(
			ctx,
			taskqueue.QueueTypeFromTickerID(ticker.ID),
			scheduleTimeoutTask,
		)

		if err != nil {
			return fmt.Errorf("could not add schedule step run timeout task to task queue: %w", err)
		}
	}

	// get all job runs assigned to the ticker
	jobRuns, err := ec.repo.JobRun().ListAllJobRuns(&repository.ListAllJobRunsOpts{
		TickerId: repository.StringPtr(payload.TickerId),
		Status:   repository.JobRunStatusPtr(db.JobRunStatusRunning),
	})

	if err != nil {
		return fmt.Errorf("could not list step runs: %w", err)
	}

	for i, jobRun := range jobRuns {
		jobRunCp := jobRun
		ticker := tickers[i%numTickers]

		_, err = ec.repo.Ticker().AddJobRun(ticker.ID, &jobRunCp)

		if err != nil {
			return fmt.Errorf("could not update step run: %w", err)
		}

		scheduleTimeoutTask, err := scheduleJobRunTimeoutTask(&ticker, &jobRunCp)

		if err != nil {
			return fmt.Errorf("could not schedule step run timeout task: %w", err)
		}

		// send a task to the ticker
		err = ec.tq.AddTask(
			ctx,
			taskqueue.QueueTypeFromTickerID(ticker.ID),
			scheduleTimeoutTask,
		)

		if err != nil {
			return fmt.Errorf("could not add schedule step run timeout task to task queue: %w", err)
		}
	}

	return nil
}

func (ec *JobsControllerImpl) getValidTickers() ([]db.TickerModel, error) {
	within := time.Now().UTC().Add(-6 * time.Second)

	tickers, err := ec.repo.Ticker().ListTickers(&repository.ListTickerOpts{
		LatestHeartbeatAt: &within,
		Active:            repository.BoolPtr(true),
	})

	if err != nil {
		return nil, fmt.Errorf("could not list tickers: %w", err)
	}

	if len(tickers) == 0 {
		return nil, fmt.Errorf("no tickers available")
	}

	return tickers, nil
}

func stepRunAssignedTask(tenantId, stepRunId, workerId, dispatcherId string) *taskqueue.Task {
	payload, _ := datautils.ToJSONMap(tasktypes.StepRunAssignedTaskPayload{
		StepRunId: stepRunId,
		WorkerId:  workerId,
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.StepRunAssignedTaskMetadata{
		TenantId:     tenantId,
		DispatcherId: dispatcherId,
	})

	return &taskqueue.Task{
		ID:       "step-run-assigned",
		Payload:  payload,
		Metadata: metadata,
	}
}

func scheduleStepRunTimeoutTask(ticker *db.TickerModel, stepRun *db.StepRunModel) (*taskqueue.Task, error) {
	var durationStr string

	if timeout, ok := stepRun.Step().Timeout(); ok {
		durationStr = timeout
	}

	if durationStr == "" {
		durationStr = defaults.DefaultStepRunTimeout
	}

	// get a duration
	duration, err := time.ParseDuration(durationStr)

	if err != nil {
		return nil, fmt.Errorf("could not parse duration: %w", err)
	}

	timeoutAt := time.Now().UTC().Add(duration)

	payload, _ := datautils.ToJSONMap(tasktypes.ScheduleStepRunTimeoutTaskPayload{
		StepRunId: stepRun.ID,
		JobRunId:  stepRun.JobRunID,
		TimeoutAt: timeoutAt.Format(time.RFC3339),
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.ScheduleStepRunTimeoutTaskMetadata{
		TenantId: stepRun.TenantID,
	})

	return &taskqueue.Task{
		ID:       "schedule-step-run-timeout",
		Payload:  payload,
		Metadata: metadata,
	}, nil
}

func scheduleJobRunTimeoutTask(ticker *db.TickerModel, jobRun *db.JobRunModel) (*taskqueue.Task, error) {
	var durationStr string

	if timeout, ok := jobRun.Job().Timeout(); ok {
		durationStr = timeout
	}

	if durationStr == "" {
		durationStr = defaults.DefaultJobRunTimeout
	}

	// get a duration
	duration, err := time.ParseDuration(durationStr)

	if err != nil {
		return nil, fmt.Errorf("could not parse duration: %w", err)
	}

	timeoutAt := time.Now().UTC().Add(duration)

	payload, _ := datautils.ToJSONMap(tasktypes.ScheduleJobRunTimeoutTaskPayload{
		JobRunId:  jobRun.ID,
		TimeoutAt: timeoutAt.Format(time.RFC3339),
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.ScheduleJobRunTimeoutTaskMetadata{
		TenantId: jobRun.TenantID,
	})

	return &taskqueue.Task{
		ID:       "schedule-job-run-timeout",
		Payload:  payload,
		Metadata: metadata,
	}, nil
}

func cancelStepRunTimeoutTask(ticker *db.TickerModel, stepRun *db.StepRunModel) *taskqueue.Task {
	payload, _ := datautils.ToJSONMap(tasktypes.CancelStepRunTimeoutTaskPayload{
		StepRunId: stepRun.ID,
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.CancelStepRunTimeoutTaskMetadata{
		TenantId: stepRun.TenantID,
	})

	return &taskqueue.Task{
		ID:       "cancel-step-run-timeout",
		Payload:  payload,
		Metadata: metadata,
	}
}

func stepRunCancelledTask(tenantId, stepRunId, workerId, dispatcherId, cancelledReason string) *taskqueue.Task {
	payload, _ := datautils.ToJSONMap(tasktypes.StepRunCancelledTaskPayload{
		StepRunId:       stepRunId,
		WorkerId:        workerId,
		CancelledReason: cancelledReason,
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.StepRunCancelledTaskMetadata{
		TenantId:     tenantId,
		DispatcherId: dispatcherId,
	})

	return &taskqueue.Task{
		ID:       "step-run-cancelled",
		Payload:  payload,
		Metadata: metadata,
	}
}
