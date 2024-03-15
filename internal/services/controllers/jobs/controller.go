package jobs

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/goccy/go-json"

	"github.com/go-co-op/gocron/v2"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/datautils/merge"
	"github.com/hatchet-dev/hatchet/internal/logger"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/internal/services/shared/defaults"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/internal/telemetry/servertel"

	hatcheterrors "github.com/hatchet-dev/hatchet/pkg/errors"
)

type JobsController interface {
	Start(ctx context.Context) error
}

type JobsControllerImpl struct {
	mq   msgqueue.MessageQueue
	l    *zerolog.Logger
	repo repository.Repository
	dv   datautils.DataDecoderValidator
	s    gocron.Scheduler
	a    *hatcheterrors.Wrapped
}

type JobsControllerOpt func(*JobsControllerOpts)

type JobsControllerOpts struct {
	mq      msgqueue.MessageQueue
	l       *zerolog.Logger
	repo    repository.Repository
	dv      datautils.DataDecoderValidator
	alerter hatcheterrors.Alerter
}

func defaultJobsControllerOpts() *JobsControllerOpts {
	logger := logger.NewDefaultLogger("jobs-controller")
	alerter := hatcheterrors.NoOpAlerter{}

	return &JobsControllerOpts{
		l:       &logger,
		dv:      datautils.NewDataDecoderValidator(),
		alerter: alerter,
	}
}

func WithMessageQueue(mq msgqueue.MessageQueue) JobsControllerOpt {
	return func(opts *JobsControllerOpts) {
		opts.mq = mq
	}
}

func WithLogger(l *zerolog.Logger) JobsControllerOpt {
	return func(opts *JobsControllerOpts) {
		opts.l = l
	}
}

func WithAlerter(a hatcheterrors.Alerter) JobsControllerOpt {
	return func(opts *JobsControllerOpts) {
		opts.alerter = a
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

	if opts.mq == nil {
		return nil, fmt.Errorf("task queue is required. use WithMessageQueue")
	}

	if opts.repo == nil {
		return nil, fmt.Errorf("repository is required. use WithRepository")
	}

	newLogger := opts.l.With().Str("service", "jobs-controller").Logger()
	opts.l = &newLogger

	s, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))

	if err != nil {
		return nil, fmt.Errorf("could not create scheduler: %w", err)
	}

	a := hatcheterrors.NewWrapped(opts.alerter)
	a.WithData(map[string]interface{}{"service": "jobs-controller"})

	return &JobsControllerImpl{
		mq:   opts.mq,
		l:    opts.l,
		repo: opts.repo,
		dv:   opts.dv,
		s:    s,
		a:    a,
	}, nil
}

func (jc *JobsControllerImpl) Start() (func() error, error) {
	ctx, cancel := context.WithCancel(context.Background())

	wg := sync.WaitGroup{}

	_, err := jc.s.NewJob(
		gocron.DurationJob(time.Second*5),
		gocron.NewTask(
			jc.runStepRunRequeue(ctx),
		),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not schedule step run requeue: %w", err)
	}

	_, err = jc.s.NewJob(
		gocron.DurationJob(time.Second*5),
		gocron.NewTask(
			jc.runStepRunReassign(ctx),
		),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not schedule step run reassign: %w", err)
	}

	jc.s.Start()

	f := func(task *msgqueue.Message) error {
		wg.Add(1)
		defer wg.Done()

		err := jc.handleTask(context.Background(), task)
		if err != nil {
			jc.l.Error().Err(err).Msg("could not handle job task")
			return jc.a.WrapErr(fmt.Errorf("could not handle job task: %w", err), map[string]interface{}{"task_id": task.ID}) // nolint: errcheck
		}

		return nil
	}

	cleanupQueue, err := jc.mq.Subscribe(msgqueue.JOB_PROCESSING_QUEUE, f, msgqueue.NoOpHook)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not subscribe to job processing queue: %w", err)
	}

	cleanup := func() error {
		cancel()

		if err := cleanupQueue(); err != nil {
			return fmt.Errorf("could not cleanup job processing queue: %w", err)
		}

		if err := jc.s.Shutdown(); err != nil {
			return fmt.Errorf("could not shutdown scheduler: %w", err)
		}

		wg.Wait()

		return nil
	}

	return cleanup, nil
}

func (ec *JobsControllerImpl) handleTask(ctx context.Context, task *msgqueue.Message) error {
	switch task.ID {
	case "job-run-queued":
		return ec.handleJobRunQueued(ctx, task)
	case "step-run-retry":
		return ec.handleStepRunRetry(ctx, task)
	case "step-run-queued":
		return ec.handleStepRunQueued(ctx, task)
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

func (ec *JobsControllerImpl) handleJobRunQueued(ctx context.Context, task *msgqueue.Message) error {
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

	err = ec.repo.JobRun().SetJobRunStatusRunning(metadata.TenantId, payload.JobRunId)

	if err != nil {
		return fmt.Errorf("could not set job run status to running: %w", err)
	}

	// list the step runs which are startable
	startableStepRuns, err := ec.repo.StepRun().ListStartableStepRuns(metadata.TenantId, payload.JobRunId, nil)

	g := new(errgroup.Group)

	for _, stepRun := range startableStepRuns {
		stepRunCp := stepRun

		g.Go(func() error {
			return ec.mq.AddMessage(
				ctx,
				msgqueue.JOB_PROCESSING_QUEUE,
				tasktypes.StepRunQueuedToTask(stepRunCp),
			)
		})
	}

	err = g.Wait()

	if err != nil {
		ec.l.Err(err).Msg("could not run step run requeue")
		return err
	}

	return nil
}

func (ec *JobsControllerImpl) handleStepRunRetry(ctx context.Context, task *msgqueue.Message) error {
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

	stepRun, err := ec.repo.StepRun().GetStepRunForEngine(metadata.TenantId, payload.StepRunId)

	if err != nil {
		return fmt.Errorf("could not get step run: %w", err)
	}

	var inputBytes []byte
	retryCount := int(stepRun.StepRun.RetryCount) + 1

	// update the input schema for the step run based on the new input
	if payload.InputData != "" {
		inputBytes = []byte(payload.InputData)

		// Merge the existing input data with the new input data. We don't blindly trust the
		// input data because the user could have deleted fields that are required by the step.
		// A better solution would be to validate the user input ahead of time.
		// NOTE: this is an expensive operation.
		if currentInput := stepRun.StepRun.Input; len(currentInput) > 0 {
			inputMap, err := datautils.JSONBytesToMap([]byte(payload.InputData))

			if err != nil {
				return fmt.Errorf("could not convert input data to map: %w", err)
			}

			currentInputMap, err := datautils.JSONBytesToMap(currentInput)

			if err != nil {
				return fmt.Errorf("could not convert current input to map: %w", err)
			}

			currentInputOverridesMap, ok1 := currentInputMap["overrides"].(map[string]interface{})
			inputOverridesMap, ok2 := inputMap["overrides"].(map[string]interface{})

			if ok1 && ok2 {
				mergedInputOverrides := merge.MergeMaps(currentInputOverridesMap, inputOverridesMap)

				inputMap["overrides"] = mergedInputOverrides

				mergedInputBytes, err := json.Marshal(inputMap)

				if err != nil {
					return fmt.Errorf("could not marshal merged input: %w", err)
				}

				inputBytes = mergedInputBytes
			}
		}

		// if the input data has been manually set, we reset the retry count as this is a user-triggered retry
		retryCount = 0
	} else {
		inputBytes = stepRun.StepRun.Input
	}

	// update step run
	_, _, err = ec.repo.StepRun().UpdateStepRun(
		ctx,
		metadata.TenantId,
		sqlchelpers.UUIDToStr(stepRun.StepRun.ID),
		&repository.UpdateStepRunOpts{
			Input:      inputBytes,
			Status:     repository.StepRunStatusPtr(db.StepRunStatusPending),
			IsRerun:    true,
			RetryCount: &retryCount,
		},
	)

	if err != nil {
		return fmt.Errorf("could not update step run: %w", err)
	}

	// send a task to the taskqueue
	return ec.mq.AddMessage(
		ctx,
		msgqueue.JOB_PROCESSING_QUEUE,
		tasktypes.StepRunQueuedToTask(stepRun),
	)
}

func (ec *JobsControllerImpl) handleStepRunQueued(ctx context.Context, task *msgqueue.Message) error {
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

func (jc *JobsControllerImpl) runStepRunRequeue(ctx context.Context) func() {
	return func() {
		jc.l.Debug().Msgf("jobs controller: checking step run requeue")

		// list all tenants
		tenants, err := jc.repo.Tenant().ListTenants()

		if err != nil {
			jc.l.Err(err).Msg("could not list tenants")
			return
		}

		g := new(errgroup.Group)

		for i := range tenants {
			tenantId := tenants[i].ID

			g.Go(func() error {
				return jc.runStepRunRequeueTenant(ctx, tenantId)
			})
		}

		err = g.Wait()

		if err != nil {
			jc.l.Err(err).Msg("could not run step run requeue")
		}
	}
}

// handleStepRunRequeue looks for any step runs that haven't been assigned that are past their requeue time
func (ec *JobsControllerImpl) runStepRunRequeueTenant(ctx context.Context, tenantId string) error {
	ctx, span := telemetry.NewSpan(ctx, "handle-step-run-requeue")
	defer span.End()

	stepRuns, err := ec.repo.StepRun().ListStepRunsToRequeue(tenantId)

	if err != nil {
		return fmt.Errorf("could not list step runs: %w", err)
	}

	g := new(errgroup.Group)

	for i := range stepRuns {
		stepRunCp := stepRuns[i]

		// wrap in func to get defer on the span to avoid leaking spans
		g.Go(func() error {
			var innerStepRun *dbsqlc.GetStepRunForEngineRow

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

				innerStepRun, updateInfo, err = ec.repo.StepRun().UpdateStepRun(ctx, tenantId, stepRunId, &repository.UpdateStepRunOpts{
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

			innerStepRun, _, err := ec.repo.StepRun().UpdateStepRun(ctx, tenantId, stepRunId, &repository.UpdateStepRunOpts{
				RequeueAfter: &requeueAfter,
			})

			if err != nil {
				return fmt.Errorf("could not update step run %s: %w", stepRunId, err)
			}

			stepId := sqlchelpers.UUIDToStr(innerStepRun.StepId)

			return ec.scheduleStepRun(ctx, tenantId, stepId, stepRunId)
		})
	}

	return g.Wait()
}

func (jc *JobsControllerImpl) runStepRunReassign(ctx context.Context) func() {
	return func() {
		jc.l.Debug().Msgf("jobs controller: checking step run reassignment")

		// list all tenants
		tenants, err := jc.repo.Tenant().ListTenants()

		if err != nil {
			jc.l.Err(err).Msg("could not list tenants")
			return
		}

		g := new(errgroup.Group)

		for i := range tenants {
			tenantId := tenants[i].ID

			g.Go(func() error {
				return jc.runStepRunReassignTenant(ctx, tenantId)
			})
		}

		err = g.Wait()

		if err != nil {
			jc.l.Err(err).Msg("could not run step run requeue")
		}
	}
}

// runStepRunReassignTenant looks for step runs that have been assigned to a worker but have not started,
// or have been running but the worker has become inactive.
func (ec *JobsControllerImpl) runStepRunReassignTenant(ctx context.Context, tenantId string) error {
	ctx, span := telemetry.NewSpan(ctx, "handle-step-run-reassign")
	defer span.End()

	stepRuns, err := ec.repo.StepRun().ListStepRunsToReassign(tenantId)

	if err != nil {
		return fmt.Errorf("could not list step runs: %w", err)
	}

	g := new(errgroup.Group)

	for i := range stepRuns {
		stepRunCp := stepRuns[i]

		// wrap in func to get defer on the span to avoid leaking spans
		g.Go(func() error {
			var innerStepRun *dbsqlc.GetStepRunForEngineRow

			ctx, span := telemetry.NewSpan(ctx, "handle-step-run-reassign-step-run")
			defer span.End()

			stepRunId := sqlchelpers.UUIDToStr(stepRunCp.ID)

			ec.l.Info().Msgf("reassigning step run %s", stepRunId)

			requeueAfter := time.Now().UTC().Add(time.Second * 5)

			// update the step to a pending assignment state
			innerStepRun, _, err := ec.repo.StepRun().UpdateStepRun(ctx, tenantId, stepRunId, &repository.UpdateStepRunOpts{
				Status:       repository.StepRunStatusPtr(db.StepRunStatusPendingAssignment),
				RequeueAfter: &requeueAfter,
			})

			if err != nil {
				return fmt.Errorf("could not update step run %s: %w", stepRunId, err)
			}

			stepId := sqlchelpers.UUIDToStr(innerStepRun.StepId)

			return ec.scheduleStepRun(ctx, tenantId, stepId, stepRunId)
		})
	}

	return g.Wait()
}

func (ec *JobsControllerImpl) queueStepRun(ctx context.Context, tenantId, stepId, stepRunId string) error {
	ctx, span := telemetry.NewSpan(ctx, "queue-step-run")
	defer span.End()

	// add the rendered data to the step run
	stepRun, err := ec.repo.StepRun().GetStepRunForEngine(tenantId, stepRunId)

	errData := map[string]interface{}{
		"tenant_id":   tenantId,
		"step_id":     stepId,
		"step_run_id": stepRunId,
	}

	if err != nil {
		return ec.a.WrapErr(fmt.Errorf("could not get step run: %w", err), errData)
	}

	// servertel.WithStepRunModel(span, stepRun)

	updateStepOpts := &repository.UpdateStepRunOpts{}

	// set scheduling timeout
	if scheduleTimeoutAt := stepRun.StepRun.ScheduleTimeoutAt.Time; scheduleTimeoutAt.IsZero() {
		var timeoutDuration time.Duration

		// get the schedule timeout from the step
		stepScheduleTimeout := stepRun.StepScheduleTimeout

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
	if in := stepRun.StepRun.Input; len(in) == 0 || string(in) == "{}" {
		lookupDataBytes := stepRun.JobRunLookupData

		if lookupDataBytes != nil {
			lookupData := &datautils.JobRunLookupData{}

			err := json.Unmarshal(lookupDataBytes, lookupData)

			if err != nil {
				return ec.a.WrapErr(fmt.Errorf("could not get job run lookup data: %w", err), errData)
			}

			userData := map[string]interface{}{}

			if setUserData := stepRun.StepCustomUserData; len(setUserData) > 0 {
				err := json.Unmarshal(setUserData, &userData)

				if err != nil {
					return fmt.Errorf("could not unmarshal custom user data: %w", err)
				}
			}

			// input data is the triggering event data and any parent step data
			inputData := datautils.StepRunData{
				Input:       lookupData.Input,
				TriggeredBy: lookupData.TriggeredBy,
				Parents:     lookupData.Steps,
				UserData:    userData,
				Overrides:   map[string]interface{}{},
			}

			inputDataBytes, err := json.Marshal(inputData)

			if err != nil {
				return ec.a.WrapErr(fmt.Errorf("could not convert input data to json: %w", err), errData)
			}

			updateStepOpts.Input = inputDataBytes
		}
	}

	// begin transaction and make sure step run is in a pending status
	// if the step run is no longer is a pending status, we should return with no error
	updateStepOpts.Status = repository.StepRunStatusPtr(db.StepRunStatusPendingAssignment)

	// indicate that the step run is pending assignment
	_, err = ec.repo.StepRun().QueueStepRun(ctx, tenantId, stepRunId, updateStepOpts)

	if err != nil {
		if errors.Is(err, repository.ErrStepRunIsNotPending) {
			ec.l.Debug().Msgf("step run %s is not pending, skipping scheduling", stepRunId)
			return nil
		}

		return ec.a.WrapErr(fmt.Errorf("could not update step run: %w", err), errData)
	}

	return ec.a.WrapErr(ec.scheduleStepRun(ctx, tenantId, stepId, stepRunId), errData)
}

func (ec *JobsControllerImpl) scheduleStepRun(ctx context.Context, tenantId, stepId, stepRunId string) error {
	ctx, span := telemetry.NewSpan(ctx, "schedule-step-run")
	defer span.End()

	stepRun, err := ec.repo.StepRun().GetStepRunById(tenantId, stepRunId)

	if err != nil {
		return fmt.Errorf("could not get step run: %w", err)
	}

	servertel.WithStepRunModel(span, stepRun)

	selectedWorkerId, dispatcherId, err := ec.repo.StepRun().AssignStepRunToWorker(tenantId, stepRunId)

	if err != nil {
		if errors.Is(err, repository.ErrNoWorkerAvailable) {
			ec.l.Debug().Msgf("no worker available for step run %s, requeueing", stepRunId)
			return nil
		}

		return fmt.Errorf("could not assign step run to worker: %w", err)
	}

	telemetry.WithAttributes(span, servertel.WorkerId(selectedWorkerId))

	tickerId, err := ec.repo.StepRun().AssignStepRunToTicker(tenantId, stepRunId)

	if err != nil {
		return fmt.Errorf("could not assign step run to ticker: %w", err)
	}

	scheduleTimeoutTask, err := scheduleStepRunTimeoutTask(stepRun)

	if err != nil {
		return fmt.Errorf("could not schedule step run timeout task: %w", err)
	}

	// send a task to the dispatcher
	err = ec.mq.AddMessage(
		ctx,
		msgqueue.QueueTypeFromDispatcherID(dispatcherId),
		stepRunAssignedTask(tenantId, stepRunId, selectedWorkerId, dispatcherId),
	)

	if err != nil {
		return fmt.Errorf("could not add job assigned task to task queue: %w", err)
	}

	// send a task to the ticker
	err = ec.mq.AddMessage(
		ctx,
		msgqueue.QueueTypeFromTickerID(tickerId),
		scheduleTimeoutTask,
	)

	if err != nil {
		return fmt.Errorf("could not add schedule step run timeout task to task queue: %w", err)
	}

	return nil
}

func (ec *JobsControllerImpl) handleStepRunStarted(ctx context.Context, task *msgqueue.Message) error {
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

	stepRun, updateInfo, err := ec.repo.StepRun().UpdateStepRun(ctx, metadata.TenantId, payload.StepRunId, &repository.UpdateStepRunOpts{
		StartedAt: &startedAt,
		Status:    repository.StepRunStatusPtr(db.StepRunStatusRunning),
	})

	if err != nil {
		return fmt.Errorf("could not update step run: %w", err)
	}

	defer ec.handleStepRunUpdateInfo(stepRun, updateInfo)

	return nil
}

func (ec *JobsControllerImpl) handleStepRunFinished(ctx context.Context, task *msgqueue.Message) error {
	ctx, span := telemetry.NewSpan(ctx, "handle-step-run-finished")
	defer span.End()

	payload := tasktypes.StepRunFinishedTaskPayload{}
	metadata := tasktypes.StepRunFinishedTaskMetadata{}

	err := ec.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode step run finished task payload: %w", err)
	}

	err = ec.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode step run finished task metadata: %w", err)
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

	stepRun, updateInfo, err := ec.repo.StepRun().UpdateStepRun(ctx, metadata.TenantId, payload.StepRunId, &repository.UpdateStepRunOpts{
		FinishedAt: &finishedAt,
		Status:     repository.StepRunStatusPtr(db.StepRunStatusSucceeded),
		Output:     stepOutput,
	})

	if err != nil {
		return fmt.Errorf("could not update step run: %w", err)
	}

	defer ec.handleStepRunUpdateInfo(stepRun, updateInfo)

	// queue the next step runs
	jobRunId := sqlchelpers.UUIDToStr(stepRun.JobRunId)
	stepRunId := sqlchelpers.UUIDToStr(stepRun.StepRun.ID)

	nextStepRuns, err := ec.repo.StepRun().ListStartableStepRuns(metadata.TenantId, jobRunId, &stepRunId)

	if err != nil {
		return fmt.Errorf("could not list startable step runs: %w", err)
	}

	for _, nextStepRun := range nextStepRuns {
		nextStepId := sqlchelpers.UUIDToStr(nextStepRun.StepId)
		nextStepRunId := sqlchelpers.UUIDToStr(nextStepRun.StepRun.ID)

		err = ec.queueStepRun(ctx, metadata.TenantId, nextStepId, nextStepRunId)

		if err != nil {
			return fmt.Errorf("could not queue next step run: %w", err)
		}
	}

	// cancel the timeout task
	if stepRun.StepRun.TickerId.Valid {
		tickerId := sqlchelpers.UUIDToStr(stepRun.StepRun.TickerId)
		tenantId := sqlchelpers.UUIDToStr(stepRun.StepRun.TenantId)
		stepRunId := sqlchelpers.UUIDToStr(stepRun.StepRun.ID)

		err = ec.mq.AddMessage(
			ctx,
			msgqueue.QueueTypeFromTickerID(tickerId),
			cancelStepRunTimeoutTask(tenantId, stepRunId),
		)

		if err != nil {
			return fmt.Errorf("could not add cancel step run timeout task to task queue: %w", err)
		}
	}

	return nil
}

func (ec *JobsControllerImpl) handleStepRunFailed(ctx context.Context, task *msgqueue.Message) error {
	ctx, span := telemetry.NewSpan(ctx, "handle-step-run-failed")
	defer span.End()

	payload := tasktypes.StepRunFailedTaskPayload{}
	metadata := tasktypes.StepRunFailedTaskMetadata{}

	err := ec.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode step run failed task payload: %w", err)
	}

	err = ec.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode step run failed task metadata: %w", err)
	}

	// update the step run in the database
	failedAt, err := time.Parse(time.RFC3339, payload.FailedAt)
	if err != nil {
		return fmt.Errorf("could not parse failed at: %w", err)
	}

	stepRun, err := ec.repo.StepRun().GetStepRunForEngine(metadata.TenantId, payload.StepRunId)

	if err != nil {
		return fmt.Errorf("could not get step run: %w", err)
	}

	// determine if step run should be retried or not
	shouldRetry := stepRun.StepRun.RetryCount < stepRun.StepRetries

	status := db.StepRunStatusFailed

	if shouldRetry {
		status = db.StepRunStatusPending
	}

	stepRun, updateInfo, err := ec.repo.StepRun().UpdateStepRun(ctx, metadata.TenantId, payload.StepRunId, &repository.UpdateStepRunOpts{
		FinishedAt: &failedAt,
		Error:      &payload.Error,
		Status:     repository.StepRunStatusPtr(status),
	})

	if err != nil {
		return fmt.Errorf("could not update step run: %w", err)
	}

	defer ec.handleStepRunUpdateInfo(stepRun, updateInfo)

	// servertel.WithStepRunModel(span, stepRun)

	// cancel the ticker for the step run
	if stepRun.StepRun.TickerId.Valid {
		tickerId := sqlchelpers.UUIDToStr(stepRun.StepRun.TickerId)
		tenantId := sqlchelpers.UUIDToStr(stepRun.StepRun.TenantId)
		stepRunId := sqlchelpers.UUIDToStr(stepRun.StepRun.ID)

		err = ec.mq.AddMessage(
			ctx,
			msgqueue.QueueTypeFromTickerID(tickerId),
			cancelStepRunTimeoutTask(tenantId, stepRunId),
		)

		if err != nil {
			return fmt.Errorf("could not add cancel step run timeout task to task queue: %w", err)
		}
	}

	if shouldRetry {
		// send a task to the taskqueue
		return ec.mq.AddMessage(
			ctx,
			msgqueue.JOB_PROCESSING_QUEUE,
			tasktypes.StepRunRetryToTask(stepRun, nil),
		)
	}

	return nil
}

func (ec *JobsControllerImpl) handleStepRunTimedOut(ctx context.Context, task *msgqueue.Message) error {
	ctx, span := telemetry.NewSpan(ctx, "handle-step-run-timed-out")
	defer span.End()

	payload := tasktypes.StepRunTimedOutTaskPayload{}
	metadata := tasktypes.StepRunTimedOutTaskMetadata{}

	err := ec.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode step run timed out task payload: %w", err)
	}

	err = ec.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode step run timed out task metadata: %w", err)
	}

	return ec.cancelStepRun(ctx, metadata.TenantId, payload.StepRunId, "TIMED_OUT")
}

func (ec *JobsControllerImpl) handleStepRunCancelled(ctx context.Context, task *msgqueue.Message) error {
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

	stepRun, updateInfo, err := ec.repo.StepRun().UpdateStepRun(ctx, tenantId, stepRunId, &repository.UpdateStepRunOpts{
		CancelledAt:     &now,
		CancelledReason: repository.StringPtr(reason),
		Status:          repository.StepRunStatusPtr(db.StepRunStatusCancelled),
	})

	if err != nil {
		return fmt.Errorf("could not update step run: %w", err)
	}

	defer ec.handleStepRunUpdateInfo(stepRun, updateInfo)

	// servertel.WithStepRunModel(span, stepRun)

	if !stepRun.StepRun.WorkerId.Valid {
		return fmt.Errorf("step run has no worker id")
	}

	workerId := sqlchelpers.UUIDToStr(stepRun.StepRun.WorkerId)

	worker, err := ec.repo.Worker().GetWorkerForEngine(tenantId, workerId)

	if err != nil {
		return fmt.Errorf("could not get worker: %w", err)
	} else if !worker.DispatcherId.Valid {
		return fmt.Errorf("worker has no dispatcher id")
	}

	dispatcherId := sqlchelpers.UUIDToStr(worker.DispatcherId)

	err = ec.mq.AddMessage(
		ctx,
		msgqueue.QueueTypeFromDispatcherID(dispatcherId),
		stepRunCancelledTask(tenantId, stepRunId, workerId, dispatcherId, reason),
	)

	if err != nil {
		return fmt.Errorf("could not add job assigned task to task queue: %w", err)
	}

	return nil
}

func (ec *JobsControllerImpl) handleStepRunUpdateInfo(stepRun *dbsqlc.GetStepRunForEngineRow, updateInfo *repository.StepRunUpdateInfo) {
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
		err := ec.mq.AddMessage(
			context.Background(),
			msgqueue.WORKFLOW_PROCESSING_QUEUE,
			tasktypes.WorkflowRunFinishedToTask(
				sqlchelpers.UUIDToStr(stepRun.StepRun.TenantId),
				updateInfo.WorkflowRunId,
				updateInfo.WorkflowRunStatus,
			),
		)

		if err != nil {
			ec.l.Error().Err(err).Msg("could not add workflow run finished task to task queue")
		}
	}
}

func (ec *JobsControllerImpl) handleTickerRemoved(ctx context.Context, task *msgqueue.Message) error {
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

		scheduleTimeoutTask, err := scheduleStepRunTimeoutTask(&stepRunCp)

		if err != nil {
			return fmt.Errorf("could not schedule step run timeout task: %w", err)
		}

		// send a task to the ticker
		err = ec.mq.AddMessage(
			ctx,
			msgqueue.QueueTypeFromTickerID(ticker.ID),
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

func stepRunAssignedTask(tenantId, stepRunId, workerId, dispatcherId string) *msgqueue.Message {
	payload, _ := datautils.ToJSONMap(tasktypes.StepRunAssignedTaskPayload{
		StepRunId: stepRunId,
		WorkerId:  workerId,
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.StepRunAssignedTaskMetadata{
		TenantId:     tenantId,
		DispatcherId: dispatcherId,
	})

	return &msgqueue.Message{
		ID:       "step-run-assigned",
		Payload:  payload,
		Metadata: metadata,
		Retries:  3,
	}
}

func scheduleStepRunTimeoutTask(stepRun *db.StepRunModel) (*msgqueue.Message, error) {
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

	return &msgqueue.Message{
		ID:       "schedule-step-run-timeout",
		Payload:  payload,
		Metadata: metadata,
		Retries:  3,
	}, nil
}

func cancelStepRunTimeoutTask(tenantId, stepRunId string) *msgqueue.Message {
	payload, _ := datautils.ToJSONMap(tasktypes.CancelStepRunTimeoutTaskPayload{
		StepRunId: stepRunId,
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.CancelStepRunTimeoutTaskMetadata{
		TenantId: tenantId,
	})

	return &msgqueue.Message{
		ID:       "cancel-step-run-timeout",
		Payload:  payload,
		Metadata: metadata,
		Retries:  3,
	}
}

func stepRunCancelledTask(tenantId, stepRunId, workerId, dispatcherId, cancelledReason string) *msgqueue.Message {
	payload, _ := datautils.ToJSONMap(tasktypes.StepRunCancelledTaskPayload{
		StepRunId:       stepRunId,
		WorkerId:        workerId,
		CancelledReason: cancelledReason,
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.StepRunCancelledTaskMetadata{
		TenantId:     tenantId,
		DispatcherId: dispatcherId,
	})

	return &msgqueue.Message{
		ID:       "step-run-cancelled",
		Payload:  payload,
		Metadata: metadata,
		Retries:  3,
	}
}
