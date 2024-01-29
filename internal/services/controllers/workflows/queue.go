package workflows

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"
	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/services/shared/defaults"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/internal/taskqueue"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/internal/telemetry/servertel"
)

func (wc *WorkflowsControllerImpl) handleWorkflowRunQueued(ctx context.Context, task *taskqueue.Task) error {
	ctx, span := telemetry.NewSpan(ctx, "handle-workflow-run-queued")
	defer span.End()

	payload := tasktypes.WorkflowRunQueuedTaskPayload{}
	metadata := tasktypes.WorkflowRunQueuedTaskMetadata{}

	err := wc.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode job task payload: %w", err)
	}

	err = wc.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode job task metadata: %w", err)
	}

	// get the workflow run in the database
	workflowRun, err := wc.repo.WorkflowRun().GetWorkflowRunById(metadata.TenantId, payload.WorkflowRunId)

	if err != nil {
		return fmt.Errorf("could not get job run: %w", err)
	}

	servertel.WithWorkflowRunModel(span, workflowRun)

	wc.l.Info().Msgf("starting workflow run %s", workflowRun.ID)

	// determine if we should start this workflow run or we need to limit its concurrency
	// if the workflow has concurrency settings, then we need to check if we can start it
	if _, hasConcurrency := workflowRun.WorkflowVersion().Concurrency(); hasConcurrency {
		wc.l.Info().Msgf("workflow %s has concurrency settings", workflowRun.ID)

		groupKeyRun, ok := workflowRun.GetGroupKeyRun()

		if !ok {
			return fmt.Errorf("could not get group key run")
		}

		err = wc.scheduleGetGroupAction(ctx, groupKeyRun)

		if err != nil {
			return fmt.Errorf("could not trigger get group action: %w", err)
		}

		return nil
	}

	err = wc.queueWorkflowRunJobs(ctx, workflowRun)

	if err != nil {
		return fmt.Errorf("could not start workflow run: %w", err)
	}

	return nil
}

func (wc *WorkflowsControllerImpl) scheduleGetGroupAction(
	ctx context.Context,
	getGroupKeyRun *db.GetGroupKeyRunModel,
) error {
	ctx, span := telemetry.NewSpan(ctx, "trigger-get-group-action")
	defer span.End()

	getGroupKeyRun, err := wc.repo.GetGroupKeyRun().UpdateGetGroupKeyRun(getGroupKeyRun.TenantID, getGroupKeyRun.ID, &repository.UpdateGetGroupKeyRunOpts{
		Status: repository.StepRunStatusPtr(db.StepRunStatusPendingAssignment),
	})

	if err != nil {
		return fmt.Errorf("could not update get group key run: %w", err)
	}

	workflowRun := getGroupKeyRun.WorkflowRun()
	concurrency, hasConcurrency := workflowRun.WorkflowVersion().Concurrency()

	if !hasConcurrency {
		return fmt.Errorf("workflow run does not have concurrency settings")
	}

	tenantId := workflowRun.TenantID

	// Assign the get group action to a worker.
	//
	// 1. Get a list of workers that can run this step. If there are no workers available, then return an error.
	// 2. Pick a worker to run the step and get the dispatcher currently connected to this worker.
	// 3. Update the step run's designated worker.
	//
	// After creating the worker, send a task to the taskqueue, which will be picked up by the dispatcher.
	after := time.Now().UTC().Add(-6 * time.Second)

	getAction, ok := concurrency.GetConcurrencyGroup()

	if !ok {
		return fmt.Errorf("could not get concurrency group")
	}

	workers, err := wc.repo.Worker().ListWorkers(workflowRun.TenantID, &repository.ListWorkersOpts{
		Action:             &getAction.ActionID,
		LastHeartbeatAfter: &after,
	})

	if err != nil {
		return fmt.Errorf("could not list workers for step: %w", err)
	}

	if len(workers) == 0 {
		wc.l.Debug().Msgf("no workers available for action %s, requeueing", getAction.ActionID)
		return nil
	}

	// pick the worker with the least jobs currently assigned (this heuristic can and should change)
	selectedWorker := workers[0]

	for _, worker := range workers {
		if worker.StepRunCount < selectedWorker.StepRunCount {
			selectedWorker = worker
		}
	}

	telemetry.WithAttributes(span, servertel.WorkerId(selectedWorker.Worker.ID))

	// update the job run's designated worker
	err = wc.repo.Worker().AddGetGroupKeyRun(tenantId, selectedWorker.Worker.ID, getGroupKeyRun.ID)

	if err != nil {
		return fmt.Errorf("could not add step run to worker: %w", err)
	}

	// pick a ticker to use for timeout
	tickers, err := wc.getValidTickers()

	if err != nil {
		return err
	}

	ticker := &tickers[0]

	ticker, err = wc.repo.Ticker().AddGetGroupKeyRun(ticker.ID, getGroupKeyRun.ID)

	if err != nil {
		return fmt.Errorf("could not add step run to ticker: %w", err)
	}

	scheduleTimeoutTask, err := scheduleGetGroupKeyRunTimeoutTask(ticker, getGroupKeyRun)

	if err != nil {
		return fmt.Errorf("could not schedule step run timeout task: %w", err)
	}

	// send a task to the dispatcher
	err = wc.tq.AddTask(
		ctx,
		taskqueue.QueueTypeFromDispatcherID(selectedWorker.Worker.Dispatcher().ID),
		getGroupActionTask(workflowRun.TenantID, workflowRun.ID, selectedWorker.Worker),
	)

	if err != nil {
		return fmt.Errorf("could not add job assigned task to task queue: %w", err)
	}

	// send a task to the ticker
	err = wc.tq.AddTask(
		ctx,
		taskqueue.QueueTypeFromTickerID(ticker.ID),
		scheduleTimeoutTask,
	)

	if err != nil {
		return fmt.Errorf("could not add schedule step run timeout task to task queue: %w", err)
	}

	return nil
}

func (wc *WorkflowsControllerImpl) queueWorkflowRunJobs(ctx context.Context, workflowRun *db.WorkflowRunModel) error {
	ctx, span := telemetry.NewSpan(ctx, "process-event")
	defer span.End()

	jobRuns := workflowRun.JobRuns()

	var err error

	for i := range jobRuns {
		err := wc.tq.AddTask(
			context.Background(),
			taskqueue.JOB_PROCESSING_QUEUE,
			tasktypes.JobRunQueuedToTask(jobRuns[i].Job(), &jobRuns[i]),
		)

		if err != nil {
			err = multierror.Append(err, fmt.Errorf("could not add job run to task queue: %w", err))
		}
	}

	return err
}

func (wc *WorkflowsControllerImpl) handleGroupKeyActionRequeue(ctx context.Context, task *taskqueue.Task) error {
	ctx, span := telemetry.NewSpan(ctx, "handle-group-key-action-requeue-ticker")
	defer span.End()

	payload := tasktypes.GroupKeyActionRequeueTaskPayload{}
	metadata := tasktypes.GroupKeyActionRequeueTaskMetadata{}

	err := wc.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode job task payload: %w", err)
	}

	err = wc.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode job task metadata: %w", err)
	}

	getGroupKeyRuns, err := wc.repo.GetGroupKeyRun().ListGetGroupKeyRuns(payload.TenantId, &repository.ListGetGroupKeyRunsOpts{
		Requeuable: repository.BoolPtr(true),
	})

	if err != nil {
		return fmt.Errorf("could not list step runs: %w", err)
	}

	g := new(errgroup.Group)

	for _, getGroupKeyRun := range getGroupKeyRuns {
		getGroupKeyRunCp := getGroupKeyRun

		// wrap in func to get defer on the span to avoid leaking spans
		g.Go(func() error {

			ctx, span := telemetry.NewSpan(ctx, "handle-step-run-requeue-step-run")
			defer span.End()

			wc.l.Debug().Msgf("requeueing step run %s", getGroupKeyRunCp.ID)

			now := time.Now().UTC().UTC()

			// if the current time is after the scheduleTimeoutAt, then mark this as timed out
			if getGroupKeyRunCp.CreatedAt.Add(30 * time.Second).Before(now) {
				_, err = wc.repo.GetGroupKeyRun().UpdateGetGroupKeyRun(payload.TenantId, getGroupKeyRunCp.ID, &repository.UpdateGetGroupKeyRunOpts{
					CancelledAt:     &now,
					CancelledReason: repository.StringPtr("SCHEDULING_TIMED_OUT"),
					Status:          repository.StepRunStatusPtr(db.StepRunStatusCancelled),
				})

				if err != nil {
					return fmt.Errorf("could not update step run %s: %w", getGroupKeyRunCp.ID, err)
				}

				return nil
			}

			requeueAfter := time.Now().UTC().Add(time.Second * 5)

			getGroupKeyRunP, err := wc.repo.GetGroupKeyRun().UpdateGetGroupKeyRun(payload.TenantId, getGroupKeyRunCp.ID, &repository.UpdateGetGroupKeyRunOpts{
				RequeueAfter: &requeueAfter,
			})

			if err != nil {
				return fmt.Errorf("could not update get group key run %s: %w", getGroupKeyRunP.ID, err)
			}

			return wc.scheduleGetGroupAction(ctx, getGroupKeyRunP)
		})
	}

	return nil
}

func (ec *WorkflowsControllerImpl) getValidTickers() ([]db.TickerModel, error) {
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

func (wc *WorkflowsControllerImpl) queueByCancelInProgress(ctx context.Context, tenantId, groupKey string, workflowVersion *db.WorkflowVersionModel) error {
	ctx, span := telemetry.NewSpan(ctx, "cancel-by-cancel-in-progress")
	defer span.End()

	wc.l.Info().Msgf("handling queue with strategy CANCEL_IN_PROGRESS for %s", groupKey)

	concurrency, hasConcurrency := workflowVersion.Concurrency()

	if !hasConcurrency {
		return nil
	}

	// list all workflow runs that are running for this group key
	running := db.WorkflowRunStatusRunning

	runningWorkflowRuns, err := wc.repo.WorkflowRun().ListWorkflowRuns(tenantId, &repository.ListWorkflowRunsOpts{
		WorkflowVersionId: &concurrency.WorkflowVersionID,
		GroupKey:          &groupKey,
		Status:            &running,
		// order from oldest to newest
		OrderBy:        repository.StringPtr("createdAt"),
		OrderDirection: repository.StringPtr("ASC"),
	})

	if err != nil {
		return fmt.Errorf("could not list running workflow runs: %w", err)
	}

	// get workflow runs which are queued for this group key
	queued := db.WorkflowRunStatusQueued

	queuedWorkflowRuns, err := wc.repo.WorkflowRun().ListWorkflowRuns(tenantId, &repository.ListWorkflowRunsOpts{
		WorkflowVersionId: &concurrency.WorkflowVersionID,
		GroupKey:          &groupKey,
		Status:            &queued,
		// order from oldest to newest
		OrderBy:        repository.StringPtr("createdAt"),
		OrderDirection: repository.StringPtr("ASC"),
		Limit:          &concurrency.MaxRuns,
	})

	if err != nil {
		return fmt.Errorf("could not list queued workflow runs: %w", err)
	}

	// cancel up to maxRuns - queued runs
	maxRuns := concurrency.MaxRuns
	maxToQueue := min(maxRuns, len(queuedWorkflowRuns.Rows))
	errGroup := new(errgroup.Group)

	for i := range runningWorkflowRuns.Rows {
		// in this strategy we need to make room for all of the queued runs
		if i >= len(queuedWorkflowRuns.Rows) {
			break
		}

		row := runningWorkflowRuns.Rows[i]

		errGroup.Go(func() error {
			return wc.cancelWorkflowRun(tenantId, row.WorkflowRun.ID)
		})
	}

	if err := errGroup.Wait(); err != nil {
		return fmt.Errorf("could not cancel workflow runs: %w", err)
	}

	errGroup = new(errgroup.Group)

	for i := range queuedWorkflowRuns.Rows {
		if i >= maxToQueue {
			break
		}

		row := queuedWorkflowRuns.Rows[i]

		errGroup.Go(func() error {
			workflowRun, err := wc.repo.WorkflowRun().GetWorkflowRunById(tenantId, row.WorkflowRun.ID)

			if err != nil {
				return fmt.Errorf("could not get workflow run: %w", err)
			}

			return wc.queueWorkflowRunJobs(ctx, workflowRun)
		})
	}

	if err := errGroup.Wait(); err != nil {
		return fmt.Errorf("could not queue workflow runs: %w", err)
	}

	return nil
}

func (wc *WorkflowsControllerImpl) cancelWorkflowRun(tenantId, workflowRunId string) error {
	// get the workflow run in the database
	workflowRun, err := wc.repo.WorkflowRun().GetWorkflowRunById(tenantId, workflowRunId)

	if err != nil {
		return fmt.Errorf("could not get workflow run: %w", err)
	}

	// cancel all running step runs
	stepRuns, err := wc.repo.StepRun().ListStepRuns(tenantId, &repository.ListStepRunsOpts{
		WorkflowRunId: &workflowRun.ID,
		Status:        repository.StepRunStatusPtr(db.StepRunStatusRunning),
	})

	if err != nil {
		return fmt.Errorf("could not list step runs: %w", err)
	}

	errGroup := new(errgroup.Group)

	for i := range stepRuns {
		stepRunCp := stepRuns[i]
		errGroup.Go(func() error {
			return wc.tq.AddTask(
				context.Background(),
				taskqueue.JOB_PROCESSING_QUEUE,
				getStepRunNotifyCancelTask(tenantId, stepRunCp.ID, "CANCELLED_BY_CONCURRENCY_LIMIT"),
			)
		})
	}

	return errGroup.Wait()
}

func getGroupActionTask(tenantId, workflowRunId string, worker *db.WorkerModel) *taskqueue.Task {
	dispatcher := worker.Dispatcher()

	payload, _ := datautils.ToJSONMap(tasktypes.GroupKeyActionAssignedTaskPayload{
		WorkflowRunId: workflowRunId,
		WorkerId:      worker.ID,
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.GroupKeyActionAssignedTaskMetadata{
		TenantId:     tenantId,
		DispatcherId: dispatcher.ID,
	})

	return &taskqueue.Task{
		ID:       "group-key-action-assigned",
		Queue:    taskqueue.QueueTypeFromDispatcherID(dispatcher.ID),
		Payload:  payload,
		Metadata: metadata,
	}
}

func getStepRunNotifyCancelTask(tenantId, stepRunId, reason string) *taskqueue.Task {
	payload, _ := datautils.ToJSONMap(tasktypes.StepRunNotifyCancelTaskPayload{
		StepRunId:       stepRunId,
		CancelledReason: reason,
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.StepRunNotifyCancelTaskMetadata{
		TenantId: tenantId,
	})

	return &taskqueue.Task{
		ID:       "step-run-cancelled",
		Queue:    taskqueue.JOB_PROCESSING_QUEUE,
		Payload:  payload,
		Metadata: metadata,
	}
}

func scheduleGetGroupKeyRunTimeoutTask(ticker *db.TickerModel, getGroupKeyRun *db.GetGroupKeyRunModel) (*taskqueue.Task, error) {
	durationStr := defaults.DefaultStepRunTimeout

	// get a duration
	duration, err := time.ParseDuration(durationStr)

	if err != nil {
		return nil, fmt.Errorf("could not parse duration: %w", err)
	}

	timeoutAt := time.Now().UTC().Add(duration)

	payload, _ := datautils.ToJSONMap(tasktypes.ScheduleGetGroupKeyRunTimeoutTaskPayload{
		GetGroupKeyRunId: getGroupKeyRun.ID,
		WorkflowRunId:    getGroupKeyRun.WorkflowRunID,
		TimeoutAt:        timeoutAt.Format(time.RFC3339),
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.ScheduleGetGroupKeyRunTimeoutTaskMetadata{
		TenantId: getGroupKeyRun.TenantID,
	})

	return &taskqueue.Task{
		ID:       "schedule-get-group-key-run-timeout",
		Queue:    taskqueue.QueueTypeFromTickerID(ticker.ID),
		Payload:  payload,
		Metadata: metadata,
	}, nil
}
