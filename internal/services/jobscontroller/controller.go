package jobscontroller

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/internal/taskqueue"
	"github.com/rs/zerolog"
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
	logger := zerolog.New(os.Stderr)
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

	return &JobsControllerImpl{
		tq:   opts.tq,
		l:    opts.l,
		repo: opts.repo,
		dv:   opts.dv,
	}, nil
}

func (jc *JobsControllerImpl) Start(ctx context.Context) error {
	taskChan, err := jc.tq.Subscribe(ctx, taskqueue.JOB_PROCESSING_QUEUE)

	if err != nil {
		return err
	}

	for {
		select {
		case task := <-taskChan:
			err = jc.handleTask(ctx, task)

			if err != nil {
				jc.l.Error().Err(err).Msg("could not handle job task")
			}
		}
	}
}

func (ec *JobsControllerImpl) handleTask(ctx context.Context, task *taskqueue.Task) error {
	switch task.ID {
	case "job-run-queued":
		return ec.handleJobRunQueued(ctx, task)
	case "job-run-timed-out":
		return ec.handleJobRunTimedOut(ctx, task)
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
	case "step-run-timed-out":
		return ec.handleStepRunTimedOut(ctx, task)
	case "ticker-removed":
		return ec.handleTickerRemoved(ctx, task)
	}

	return fmt.Errorf("unknown task: %s in queue %s", task.ID, string(task.Queue))
}

func (ec *JobsControllerImpl) handleJobRunQueued(ctx context.Context, task *taskqueue.Task) error {
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

	// schedule the first step in the job run
	stepRuns := jobRun.StepRuns()

	if len(stepRuns) == 0 {
		return fmt.Errorf("job run has no step runs")
	}

	stepRun := jobRun.StepRuns()[0]

	// find the step run without a previous
	for _, sr := range stepRuns {
		if prev, ok := sr.Step().Prev(); !ok || prev == nil {
			stepRun = sr
			break
		}
	}

	// send a task to the taskqueue
	err = ec.tq.AddTask(
		ctx,
		taskqueue.JOB_PROCESSING_QUEUE,
		tasktypes.StepRunQueuedToTask(
			jobRun.Job(),
			&stepRun,
		),
	)

	if err != nil {
		return fmt.Errorf("could not add job queued task to task queue: %w", err)
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
		taskqueue.QueueTypeFromTicker(ticker),
		scheduleTimeoutTask,
	)

	// if err != nil {
	// 	return fmt.Errorf("could not add schedule job run timeout task to task queue: %w", err)
	// }

	// // update the job run
	// _, err = ec.repo.JobRun().UpdateJobRun(metadata.TenantId, jobRun.ID, &repository.UpdateJobRunOpts{
	// 	Status: repository.JobRunStatusPtr(db.JobRunStatusRUNNING),
	// })

	return nil
}

func (ec *JobsControllerImpl) handleJobRunTimedOut(ctx context.Context, task *taskqueue.Task) error {
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

	stepRuns, err := ec.repo.StepRun().ListStepRuns(metadata.TenantId, &repository.ListStepRunsOpts{
		JobRunId: &jobRun.ID,
		Status:   repository.StepRunStatusPtr(db.StepRunStatusRUNNING),
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
		now := time.Now()

		// cancel current step run
		stepRun, err := ec.repo.StepRun().UpdateStepRun(metadata.TenantId, currStepRun.ID, &repository.UpdateStepRunOpts{
			CancelledAt:     &now,
			CancelledReason: repository.StringPtr("JOB_RUN_TIMED_OUT"),
			Status:          repository.StepRunStatusPtr(db.StepRunStatusCANCELLED),
		})

		if err != nil {
			return fmt.Errorf("could not update step run: %w", err)
		}

		workerId, ok := stepRun.WorkerID()

		if !ok {
			return fmt.Errorf("step run has no worker id")
		}

		worker, err := ec.repo.Worker().GetWorkerById(workerId)

		if err != nil {
			return fmt.Errorf("could not get worker: %w", err)
		}

		// send a task to the taskqueue
		err = ec.tq.AddTask(
			ctx,
			taskqueue.QueueTypeFromDispatcher(worker.Dispatcher()),
			stepRunCancelledTask(metadata.TenantId, currStepRun.ID, "JOB_RUN_TIMED_OUT", worker),
		)

		if err != nil {
			return fmt.Errorf("could not add job assigned task to task queue: %w", err)
		}

		// cancel the ticker for the step run
		stepRunTicker, ok := stepRun.Ticker()

		if ok {
			err = ec.tq.AddTask(
				ctx,
				taskqueue.QueueTypeFromTicker(stepRunTicker),
				cancelStepRunTimeoutTask(stepRunTicker, stepRun),
			)

			if err != nil {
				return fmt.Errorf("could not add cancel step run timeout task to task queue: %w", err)
			}
		}
	}

	// // cancel all step runs
	// err = ec.repo.StepRun().CancelPendingStepRuns(metadata.TenantId, jobRun.ID, "JOB_RUN_TIMED_OUT")

	// if err != nil {
	// 	return fmt.Errorf("could not cancel pending step runs: %w", err)
	// }

	// // update the job run in the database
	// _, err = ec.repo.JobRun().UpdateJobRun(metadata.TenantId, jobRun.ID, &repository.UpdateJobRunOpts{
	// 	Status: repository.JobRunStatusPtr(db.JobRunStatusCANCELLED),
	// })

	// if err != nil {
	// 	return fmt.Errorf("could not update job run: %w", err)
	// }

	return nil
}

func (ec *JobsControllerImpl) handleStepRunQueued(ctx context.Context, task *taskqueue.Task) error {
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

	stepRuns, err := ec.repo.StepRun().ListStepRuns(payload.TenantId, &repository.ListStepRunsOpts{
		Requeuable: repository.BoolPtr(true),
	})

	if err != nil {
		return fmt.Errorf("could not list step runs: %w", err)
	}

	var allErrs error

	for _, stepRun := range stepRuns {
		ec.l.Debug().Msgf("requeueing step run %s", stepRun.ID)

		stepRunCp := stepRun

		// TODO: update the job run and send a task to the taskqueue
		requeueAfter := time.Now().Add(time.Second * 5)

		_, err = ec.repo.StepRun().UpdateStepRun(payload.TenantId, stepRunCp.ID, &repository.UpdateStepRunOpts{
			RequeueAfter: &requeueAfter,
		})

		if err != nil {
			allErrs = multierror.Append(allErrs, fmt.Errorf("could not update step run %s: %w", stepRun.ID, err))
		}

		err = ec.tq.AddTask(
			ctx,
			taskqueue.JOB_PROCESSING_QUEUE,
			tasktypes.StepRunQueuedToTask(
				stepRunCp.JobRun().Job(),
				&stepRunCp,
			),
		)

		if err != nil {
			allErrs = multierror.Append(allErrs, fmt.Errorf("could not add job queued task to task queue: %w", err))
		}
	}

	return allErrs
}

func (ec *JobsControllerImpl) queueStepRun(ctx context.Context, tenantId, stepId, stepRunId string) error {
	// add the rendered data to the step run
	stepRun, err := ec.repo.StepRun().GetStepRunById(tenantId, stepRunId)

	if err != nil {
		return fmt.Errorf("could not get step run: %w", err)
	}

	lookupDataModel, ok := stepRun.JobRun().LookupData()

	if ok && lookupDataModel != nil {
		data, ok := lookupDataModel.Data()

		lookupDataMap := map[string]interface{}{}
		inputDataMap := map[string]interface{}{}

		err := datautils.FromJSONType(&data, &lookupDataMap)

		if err != nil {
			return fmt.Errorf("could not get job run lookup data: %w", err)
		}

		// input data is the step data
		inputData, ok := stepRun.Step().Inputs()

		if ok {
			ec.l.Debug().Msgf("rendering template fields for step run %s", stepRun.ID)

			err := datautils.FromJSONType(&inputData, &inputDataMap)

			if err != nil {
				return fmt.Errorf("could not get step inputs: %w", err)
			}

			err = datautils.RenderTemplateFields(lookupDataMap, inputDataMap)

			if err != nil {
				return fmt.Errorf("could not render template fields: %w", err)
			}

			newInput, err := datautils.ToJSONType(inputDataMap)

			if err != nil {
				return fmt.Errorf("could not convert input data to json: %w", err)
			}

			// update the step's input data
			_, err = ec.repo.StepRun().UpdateStepRun(tenantId, stepRunId, &repository.UpdateStepRunOpts{
				Input: newInput,
			})

			if err != nil {
				return fmt.Errorf("could not update step run: %w", err)
			}
		}
	}

	return ec.scheduleStepRun(ctx, tenantId, stepId, stepRunId)
}

func (ec *JobsControllerImpl) scheduleStepRun(ctx context.Context, tenantId, stepId, stepRunId string) error {
	// indicate that the step run is pending assignment
	stepRun, err := ec.repo.StepRun().UpdateStepRun(tenantId, stepRunId, &repository.UpdateStepRunOpts{
		Status: repository.StepRunStatusPtr(db.StepRunStatusPENDINGASSIGNMENT),
	})

	if err != nil {
		return fmt.Errorf("could not update step run: %w", err)
	}

	// Assign the step run to a worker.
	//
	// 1. Get a list of workers that can run this step. If there are no workers available, then simply return with
	//    no additional transactions and this step run will be requeued.
	// 2. Pick a worker to run the step and get the dispatcher currently connected to this worker.
	// 3. Update the step run's designated worker.
	//
	// After creating the worker, send a task to the taskqueue, which will be picked up by the dispatcher.
	workers, err := ec.repo.Worker().ListWorkers(tenantId, &repository.ListWorkersOpts{
		Action: &stepRun.Step().ActionID,
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
		if worker.StepRunCount < selectedWorker.StepRunCount {
			selectedWorker = worker
		}
	}

	// update the job run's designated worker
	err = ec.repo.Worker().AddStepRun(tenantId, selectedWorker.Worker.ID, stepRunId)

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

	// send a task to the dispatcher
	err = ec.tq.AddTask(
		ctx,
		taskqueue.QueueTypeFromDispatcher(selectedWorker.Worker.Dispatcher()),
		stepRunAssignedTask(tenantId, stepRunId, selectedWorker.Worker),
	)

	if err != nil {
		return fmt.Errorf("could not add job assigned task to task queue: %w", err)
	}

	// send a task to the ticker
	err = ec.tq.AddTask(
		ctx,
		taskqueue.QueueTypeFromTicker(ticker),
		scheduleTimeoutTask,
	)

	if err != nil {
		return fmt.Errorf("could not add schedule step run timeout task to task queue: %w", err)
	}

	return nil
}

func (ec *JobsControllerImpl) handleStepRunStarted(ctx context.Context, task *taskqueue.Task) error {
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

	_, err = ec.repo.StepRun().UpdateStepRun(metadata.TenantId, payload.StepRunId, &repository.UpdateStepRunOpts{
		StartedAt: &startedAt,
		Status:    repository.StepRunStatusPtr(db.StepRunStatusRUNNING),
	})

	return err
}

func (ec *JobsControllerImpl) handleStepRunFinished(ctx context.Context, task *taskqueue.Task) error {
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

	stepRun, err := ec.repo.StepRun().UpdateStepRun(metadata.TenantId, payload.StepRunId, &repository.UpdateStepRunOpts{
		FinishedAt: &finishedAt,
		Status:     repository.StepRunStatusPtr(db.StepRunStatusSUCCEEDED),
	})

	if err != nil {
		return fmt.Errorf("could not update step run: %w", err)
	}

	jobRun, err := ec.repo.JobRun().GetJobRunById(metadata.TenantId, stepRun.JobRunID)

	if err != nil {
		return fmt.Errorf("could not get job run: %w", err)
	}

	lookupData, ok := jobRun.LookupData()

	if !ok {
		return fmt.Errorf("job run has no lookup data")
	}

	stepReadableId, ok := stepRun.Step().ReadableID()

	// only update the job lookup data if the step has a readable id to key
	if ok && stepReadableId != "" {
		data, _ := lookupData.Data()

		newData, err := datautils.AddStepOutput(&data, stepReadableId, []byte(payload.StepOutputData))

		if err != nil {
			return fmt.Errorf("could not add step output to lookup data: %w", err)
		}

		// update the job run lookup data
		_, err = ec.repo.JobRun().UpdateJobRunLookupData(metadata.TenantId, stepRun.JobRunID, &repository.UpdateJobRunLookupDataOpts{
			LookupData: newData,
		})

		if err != nil {
			return fmt.Errorf("could not update job run lookup data: %w", err)
		}
	}

	// queue the next step run if there is one
	if next, ok := stepRun.Next(); ok && next != nil {
		err := ec.queueStepRun(ctx, metadata.TenantId, next.StepID, next.ID)

		if err != nil {
			return fmt.Errorf("could not queue next step run: %w", err)
		}
	}

	// else {
	// 	ec.l.Debug().Msgf("no next step for step run %s", stepRun.ID)

	// 	// if there is no next step, then the job run is complete
	// 	_, err := ec.repo.JobRun().UpdateJobRun(metadata.TenantId, stepRun.JobRunID, &repository.UpdateJobRunOpts{
	// 		Status: repository.RunStatusPtr(db.RunStatusFINISHED),
	// 	})

	// 	if err != nil {
	// 		return fmt.Errorf("could not update job run: %w", err)
	// 	}
	// }

	// cancel the timeout task
	stepRunTicker, ok := stepRun.Ticker()

	if ok {
		err = ec.tq.AddTask(
			ctx,
			taskqueue.QueueTypeFromTicker(stepRunTicker),
			cancelStepRunTimeoutTask(stepRunTicker, stepRun),
		)

		if err != nil {
			return fmt.Errorf("could not add cancel step run timeout task to task queue: %w", err)
		}
	}

	return nil
}

func (ec *JobsControllerImpl) handleStepRunFailed(ctx context.Context, task *taskqueue.Task) error {
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

	stepRun, err := ec.repo.StepRun().UpdateStepRun(metadata.TenantId, payload.StepRunId, &repository.UpdateStepRunOpts{
		FinishedAt: &failedAt,
		Error:      &payload.Error,
		Status:     repository.StepRunStatusPtr(db.StepRunStatusFAILED),
	})

	if err != nil {
		return fmt.Errorf("could not update step run: %w", err)
	}

	// cancel the ticker for the step run
	stepRunTicker, ok := stepRun.Ticker()

	if ok {
		err = ec.tq.AddTask(
			ctx,
			taskqueue.QueueTypeFromTicker(stepRunTicker),
			cancelStepRunTimeoutTask(stepRunTicker, stepRun),
		)

		if err != nil {
			return fmt.Errorf("could not add cancel step run timeout task to task queue: %w", err)
		}
	}

	// // cancel any pending steps for this job run in the database
	// err = ec.repo.StepRun().CancelPendingStepRuns(metadata.TenantId, stepRun.JobRunID, "PREVIOUS_STEP_FAILED")

	// if err != nil {
	// 	return fmt.Errorf("could not cancel pending step runs: %w", err)
	// }

	// // update the job run in the database
	// _, err = ec.repo.JobRun().UpdateJobRun(metadata.TenantId, stepRun.JobRunID, &repository.UpdateJobRunOpts{
	// 	Status: repository.JobRunStatusPtr(db.JobRunStatusFAILED),
	// })

	// if err != nil {
	// 	return fmt.Errorf("could not update job run: %w", err)
	// }

	return nil
}

func (ec *JobsControllerImpl) handleStepRunTimedOut(ctx context.Context, task *taskqueue.Task) error {
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

	now := time.Now()

	// cancel current step run
	stepRun, err := ec.repo.StepRun().UpdateStepRun(metadata.TenantId, payload.StepRunId, &repository.UpdateStepRunOpts{
		CancelledAt:     &now,
		CancelledReason: repository.StringPtr("TIMED_OUT"),
		Status:          repository.StepRunStatusPtr(db.StepRunStatusCANCELLED),
	})

	if err != nil {
		return fmt.Errorf("could not update step run: %w", err)
	}

	workerId, ok := stepRun.WorkerID()

	if !ok {
		return fmt.Errorf("step run has no worker id")
	}

	worker, err := ec.repo.Worker().GetWorkerById(workerId)

	if err != nil {
		return fmt.Errorf("could not get worker: %w", err)
	}

	// send a task to the taskqueue
	err = ec.tq.AddTask(
		ctx,
		taskqueue.QueueTypeFromDispatcher(worker.Dispatcher()),
		stepRunCancelledTask(metadata.TenantId, payload.StepRunId, "TIMED_OUT", worker),
	)

	if err != nil {
		return fmt.Errorf("could not add job assigned task to task queue: %w", err)
	}

	// // cancel any pending steps for this job run in the database
	// err = ec.repo.StepRun().CancelPendingStepRuns(metadata.TenantId, payload.JobRunId, "PREVIOUS_STEP_TIMED_OUT")

	// if err != nil {
	// 	return fmt.Errorf("could not cancel pending step runs: %w", err)
	// }

	return nil
}

func (ec *JobsControllerImpl) handleTickerRemoved(ctx context.Context, task *taskqueue.Task) error {
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

	// reassign all step runs to a different ticker
	tickers, err := ec.getValidTickers()

	if err != nil {
		return err
	}

	// reassign all step runs randomly to tickers
	numTickers := len(tickers)

	// get all step runs assigned to the ticker
	stepRuns, err := ec.repo.StepRun().ListAllStepRuns(&repository.ListAllStepRunsOpts{
		NoTickerId: repository.BoolPtr(true),
		Status:     repository.StepRunStatusPtr(db.StepRunStatusRUNNING),
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
			taskqueue.QueueTypeFromTicker(&ticker),
			scheduleTimeoutTask,
		)

		if err != nil {
			return fmt.Errorf("could not add schedule step run timeout task to task queue: %w", err)
		}
	}

	// get all step runs assigned to the ticker
	jobRuns, err := ec.repo.JobRun().ListAllJobRuns(&repository.ListAllJobRunsOpts{
		NoTickerId: repository.BoolPtr(true),
		Status:     repository.JobRunStatusPtr(db.JobRunStatusRUNNING),
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
			taskqueue.QueueTypeFromTicker(&ticker),
			scheduleTimeoutTask,
		)

		if err != nil {
			return fmt.Errorf("could not add schedule step run timeout task to task queue: %w", err)
		}
	}

	return nil
}

func (ec *JobsControllerImpl) getValidTickers() ([]db.TickerModel, error) {
	within := time.Now().Add(-6 * time.Second)

	tickers, err := ec.repo.Ticker().ListTickers(&repository.ListTickerOpts{
		LatestHeartbeatAt: &within,
	})

	if err != nil {
		return nil, fmt.Errorf("could not list tickers: %w", err)
	}

	if len(tickers) == 0 {
		return nil, fmt.Errorf("no tickers available")
	}

	return tickers, nil
}

func stepRunAssignedTask(tenantId, stepRunId string, worker *db.WorkerModel) *taskqueue.Task {
	dispatcher := worker.Dispatcher()

	payload, _ := datautils.ToJSONMap(tasktypes.StepRunAssignedTaskPayload{
		StepRunId: stepRunId,
		WorkerId:  worker.ID,
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.StepRunAssignedTaskMetadata{
		TenantId:     tenantId,
		DispatcherId: dispatcher.ID,
	})

	return &taskqueue.Task{
		ID:       "step-run-assigned",
		Queue:    taskqueue.QueueTypeFromDispatcher(dispatcher),
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
		durationStr = "300s"
	}

	// get a duration
	duration, err := time.ParseDuration(durationStr)

	if err != nil {
		return nil, fmt.Errorf("could not parse duration: %w", err)
	}

	timeoutAt := time.Now().Add(duration)

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
		Queue:    taskqueue.QueueTypeFromTicker(ticker),
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
		durationStr = "300s"
	}

	// get a duration
	duration, err := time.ParseDuration(durationStr)

	if err != nil {
		return nil, fmt.Errorf("could not parse duration: %w", err)
	}

	timeoutAt := time.Now().Add(duration)

	payload, _ := datautils.ToJSONMap(tasktypes.ScheduleJobRunTimeoutTaskPayload{
		JobRunId:  jobRun.ID,
		TimeoutAt: timeoutAt.Format(time.RFC3339),
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.ScheduleJobRunTimeoutTaskMetadata{
		TenantId: jobRun.TenantID,
	})

	return &taskqueue.Task{
		ID:       "schedule-job-run-timeout",
		Queue:    taskqueue.QueueTypeFromTicker(ticker),
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
		Queue:    taskqueue.QueueTypeFromTicker(ticker),
		Payload:  payload,
		Metadata: metadata,
	}
}

func cancelJobRunTimeoutTask(ticker *db.TickerModel, jobRun *db.JobRunModel) *taskqueue.Task {
	payload, _ := datautils.ToJSONMap(tasktypes.CancelJobRunTimeoutTaskPayload{
		JobRunId: jobRun.ID,
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.CancelJobRunTimeoutTaskMetadata{
		TenantId: jobRun.TenantID,
	})

	return &taskqueue.Task{
		ID:       "cancel-job-run-timeout",
		Queue:    taskqueue.QueueTypeFromTicker(ticker),
		Payload:  payload,
		Metadata: metadata,
	}
}

func stepRunCancelledTask(tenantId, stepRunId, cancelledReason string, worker *db.WorkerModel) *taskqueue.Task {
	dispatcher := worker.Dispatcher()

	payload, _ := datautils.ToJSONMap(tasktypes.StepRunCancelledTaskPayload{
		StepRunId:       stepRunId,
		WorkerId:        worker.ID,
		CancelledReason: cancelledReason,
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.StepRunCancelledTaskMetadata{
		TenantId:     tenantId,
		DispatcherId: dispatcher.ID,
	})

	return &taskqueue.Task{
		ID:       "step-run-cancelled",
		Queue:    taskqueue.QueueTypeFromDispatcher(dispatcher),
		Payload:  payload,
		Metadata: metadata,
	}
}
