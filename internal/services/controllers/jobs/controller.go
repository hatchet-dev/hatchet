package jobs

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/goccy/go-json"
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
	"github.com/hatchet-dev/hatchet/internal/services/shared/recoveryutils"
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
	repo repository.EngineRepository
	dv   datautils.DataDecoderValidator
	s    gocron.Scheduler
	a    *hatcheterrors.Wrapped
}

type JobsControllerOpt func(*JobsControllerOpts)

type JobsControllerOpts struct {
	mq      msgqueue.MessageQueue
	l       *zerolog.Logger
	repo    repository.EngineRepository
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

func WithRepository(r repository.EngineRepository) JobsControllerOpt {
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

	startedAt := time.Now()

	_, err := jc.s.NewJob(
		gocron.DurationJob(time.Second*1),
		gocron.NewTask(
			jc.runStepRunRequeue(ctx, startedAt),
		),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not schedule step run requeue: %w", err)
	}

	_, err = jc.s.NewJob(
		gocron.DurationJob(time.Second*1),
		gocron.NewTask(
			jc.runStepRunReassign(ctx, startedAt),
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

func (ec *JobsControllerImpl) handleTask(ctx context.Context, task *msgqueue.Message) (err error) {
	defer func() {
		if r := recover(); r != nil {
			recoverErr := recoveryutils.RecoverWithAlert(ec.l, ec.a, r)

			if recoverErr != nil {
				err = recoverErr
			}
		}
	}()

	switch task.ID {
	case "job-run-queued":
		return ec.handleJobRunQueued(ctx, task)
	case "job-run-cancelled":
		return ec.handleJobRunCancelled(ctx, task)
	case "step-run-replay":
		return ec.handleStepRunReplay(ctx, task)
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

	err = ec.repo.JobRun().SetJobRunStatusRunning(ctx, metadata.TenantId, payload.JobRunId)

	if err != nil {
		return fmt.Errorf("could not set job run status to running: %w", err)
	}

	// list the step runs which are startable
	startableStepRuns, err := ec.repo.StepRun().ListStartableStepRuns(ctx, metadata.TenantId, payload.JobRunId, nil)
	if err != nil {
		return fmt.Errorf("could not list startable step runs: %w", err)
	}

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
		ec.l.Err(err).Msg("could not run job run queued")
		return err
	}

	return nil
}

func (ec *JobsControllerImpl) handleJobRunCancelled(ctx context.Context, task *msgqueue.Message) error {
	ctx, span := telemetry.NewSpan(ctx, "handle-job-run-cancelled")
	defer span.End()

	payload := tasktypes.JobRunCancelledTaskPayload{}
	metadata := tasktypes.JobRunCancelledTaskMetadata{}

	err := ec.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode job task payload: %w", err)
	}

	err = ec.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode job task metadata: %w", err)
	}

	stepRuns, err := ec.repo.StepRun().ListStepRuns(ctx, metadata.TenantId, &repository.ListStepRunsOpts{
		JobRunId: &payload.JobRunId,
	})

	if err != nil {
		return fmt.Errorf("could not list step runs: %w", err)
	}

	g := new(errgroup.Group)

	for _, stepRun := range stepRuns {
		stepRunCp := stepRun

		reason := "JOB_RUN_CANCELLED"

		if payload.Reason != nil {
			reason = *payload.Reason
		}

		g.Go(func() error {
			return ec.mq.AddMessage(
				ctx,
				msgqueue.JOB_PROCESSING_QUEUE,
				tasktypes.StepRunCancelToTask(stepRunCp, reason),
			)
		})
	}

	err = g.Wait()

	if err != nil {
		ec.l.Err(err).Msg("could not run job run cancelled")
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

	err = ec.repo.StepRun().ArchiveStepRunResult(ctx, metadata.TenantId, payload.StepRunId)

	if err != nil {
		return fmt.Errorf("could not archive step run result: %w", err)
	}

	stepRun, err := ec.repo.StepRun().GetStepRunForEngine(ctx, metadata.TenantId, payload.StepRunId)

	if err != nil {
		return fmt.Errorf("could not get step run: %w", err)
	}

	inputBytes := stepRun.StepRun.Input
	retryCount := int(stepRun.StepRun.RetryCount) + 1

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
			Event: &repository.CreateStepRunEventOpts{
				EventReason: repository.StepRunEventReasonPtr(dbsqlc.StepRunEventReasonRETRYING),
				EventMessage: repository.StringPtr(
					fmt.Sprintf("Retrying step run. This is retry %d / %d", retryCount, stepRun.StepRetries),
				)},
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

// handleStepRunReplay replays a step run from scratch - it resets the workflow run state, job run state, and
// all cancelled step runs which are children of the step run being replayed.
func (ec *JobsControllerImpl) handleStepRunReplay(ctx context.Context, task *msgqueue.Message) error {
	ctx, span := telemetry.NewSpan(ctx, "handle-step-run-replay")
	defer span.End()

	payload := tasktypes.StepRunReplayTaskPayload{}
	metadata := tasktypes.StepRunReplayTaskMetadata{}

	err := ec.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode job task payload: %w", err)
	}

	err = ec.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode job task metadata: %w", err)
	}

	err = ec.repo.StepRun().ArchiveStepRunResult(ctx, metadata.TenantId, payload.StepRunId)

	if err != nil {
		return fmt.Errorf("could not archive step run result: %w", err)
	}

	// Unlink the step run from its existing worker. This is necessary because automatic retries increment the
	// worker semaphore on failure/cancellation, but in this case we don't want to increment the semaphore.
	// FIXME: this is very far decoupled from the actual worker logic, and should be refactored.
	err = ec.repo.StepRun().UnlinkStepRunFromWorker(ctx, metadata.TenantId, payload.StepRunId)

	if err != nil {
		return fmt.Errorf("could not unlink step run from worker: %w", err)
	}

	stepRun, err := ec.repo.StepRun().GetStepRunForEngine(ctx, metadata.TenantId, payload.StepRunId)

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
	_, err = ec.repo.StepRun().ReplayStepRun(
		ctx,
		metadata.TenantId,
		sqlchelpers.UUIDToStr(stepRun.StepRun.ID),
		&repository.UpdateStepRunOpts{
			Input:      inputBytes,
			Status:     repository.StepRunStatusPtr(db.StepRunStatusPending),
			IsRerun:    true,
			RetryCount: &retryCount,
			Event: &repository.CreateStepRunEventOpts{
				EventReason: repository.StepRunEventReasonPtr(dbsqlc.StepRunEventReasonRETRIEDBYUSER),
				EventMessage: repository.StringPtr(
					"This step was manually replayed by a user",
				)},
		},
	)

	if err != nil {
		return fmt.Errorf("could not update step run for replay: %w", err)
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

func (jc *JobsControllerImpl) runStepRunRequeue(ctx context.Context, startedAt time.Time) func() {
	return func() {
		// if we are within 15 seconds of the started time, then we should not requeue step runs
		if time.Since(startedAt) < 15*time.Second {
			return
		}

		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		jc.l.Debug().Msgf("jobs controller: checking step run requeue")

		// list all tenants
		tenants, err := jc.repo.Tenant().ListTenants(ctx)

		if err != nil {
			jc.l.Err(err).Msg("could not list tenants")
			return
		}

		g := new(errgroup.Group)

		for i := range tenants {
			tenantId := sqlchelpers.UUIDToStr(tenants[i].ID)

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

	stepRuns, err := ec.repo.StepRun().ListStepRunsToRequeue(ctx, tenantId)

	if err != nil {
		return fmt.Errorf("could not list step runs to requeue: %w", err)
	}

	if num := len(stepRuns); num > 0 {
		ec.l.Info().Msgf("requeueing %d step runs", num)
	}

	g := new(errgroup.Group)

	for i := range stepRuns {
		stepRunCp := stepRuns[i]

		// wrap in func to get defer on the span to avoid leaking spans
		g.Go(func() error {
			ctx, span := telemetry.NewSpan(ctx, "handle-step-run-requeue-step-run")
			defer span.End()

			now := time.Now().UTC()
			stepRunId := sqlchelpers.UUIDToStr(stepRunCp.StepRun.ID)

			// if the current time is after the scheduleTimeoutAt, then mark this as timed out
			scheduleTimeoutAt := stepRunCp.StepRun.ScheduleTimeoutAt.Time

			// timed out if there was no scheduleTimeoutAt set and the current time is after the step run created at time plus the default schedule timeout,
			// or if the scheduleTimeoutAt is set and the current time is after the scheduleTimeoutAt
			isTimedOut := !scheduleTimeoutAt.IsZero() && scheduleTimeoutAt.Before(now)

			if isTimedOut {
				return ec.cancelStepRun(ctx, tenantId, stepRunId, "SCHEDULING_TIMED_OUT")
			}

			return ec.scheduleStepRun(ctx, tenantId, stepRunCp)
		})
	}

	return g.Wait()
}

func (jc *JobsControllerImpl) runStepRunReassign(ctx context.Context, startedAt time.Time) func() {
	return func() {
		// if we are within 15 seconds of the started time, then we should not reassign step runs
		if time.Since(startedAt) < 15*time.Second {
			return
		}

		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		jc.l.Debug().Msgf("jobs controller: checking step run reassignment")

		// list all tenants
		tenants, err := jc.repo.Tenant().ListTenants(ctx)

		if err != nil {
			jc.l.Err(err).Msg("could not list tenants")
			return
		}

		g := new(errgroup.Group)

		for i := range tenants {
			tenantId := sqlchelpers.UUIDToStr(tenants[i].ID)

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

	stepRuns, err := ec.repo.StepRun().ListStepRunsToReassign(ctx, tenantId)

	if err != nil {
		return fmt.Errorf("could not list step runs to reassign: %w", err)
	}

	if num := len(stepRuns); num > 0 {
		ec.l.Info().Msgf("reassigning %d step runs", num)
	}

	g := new(errgroup.Group)

	for i := range stepRuns {
		stepRunCp := stepRuns[i]

		// wrap in func to get defer on the span to avoid leaking spans
		g.Go(func() error {
			ctx, span := telemetry.NewSpan(ctx, "handle-step-run-reassign-step-run")
			defer span.End()

			stepRunId := sqlchelpers.UUIDToStr(stepRunCp.StepRun.ID)

			// if the current time is after the scheduleTimeoutAt, then mark this as timed out
			now := time.Now().UTC().UTC()
			scheduleTimeoutAt := stepRunCp.StepRun.ScheduleTimeoutAt.Time

			// timed out if there was no scheduleTimeoutAt set and the current time is after the step run created at time plus the default schedule timeout,
			// or if the scheduleTimeoutAt is set and the current time is after the scheduleTimeoutAt
			isTimedOut := !scheduleTimeoutAt.IsZero() && scheduleTimeoutAt.Before(now)

			if isTimedOut {
				return ec.cancelStepRun(ctx, tenantId, stepRunId, "SCHEDULING_TIMED_OUT")
			}

			eventData := map[string]interface{}{
				"worker_id": sqlchelpers.UUIDToStr(stepRunCp.StepRun.WorkerId),
			}

			err = ec.repo.StepRun().CreateStepRunEvent(ctx, tenantId, stepRunId, repository.CreateStepRunEventOpts{
				EventReason:   repository.StepRunEventReasonPtr(dbsqlc.StepRunEventReasonREASSIGNED),
				EventSeverity: repository.StepRunEventSeverityPtr(dbsqlc.StepRunEventSeverityCRITICAL),
				EventMessage:  repository.StringPtr("Worker has become inactive"),
				EventData:     &eventData,
			})

			if err != nil {
				return fmt.Errorf("could not create step run event: %w", err)
			}

			return ec.scheduleStepRun(ctx, tenantId, stepRunCp)
		})
	}

	return g.Wait()
}

func (ec *JobsControllerImpl) queueStepRun(ctx context.Context, tenantId, stepId, stepRunId string) error {
	ctx, span := telemetry.NewSpan(ctx, "queue-step-run")
	defer span.End()

	// add the rendered data to the step run
	stepRun, err := ec.repo.StepRun().GetStepRunForEngine(ctx, tenantId, stepRunId)

	errData := map[string]interface{}{
		"tenant_id":   tenantId,
		"step_id":     stepId,
		"step_run_id": stepRunId,
	}

	if err != nil {
		return ec.a.WrapErr(fmt.Errorf("could not get step run: %w", err), errData)
	}

	servertel.WithStepRunModel(span, stepRun)

	requeueAfterTime := time.Now().Add(4 * time.Second).UTC()

	updateStepOpts := &repository.UpdateStepRunOpts{
		RequeueAfter: &requeueAfterTime,
	}

	// set scheduling timeout
	if scheduleTimeoutAt := stepRun.StepRun.ScheduleTimeoutAt.Time; scheduleTimeoutAt.IsZero() {
		scheduleTimeoutAt = getScheduleTimeout(stepRun)

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
	stepRun, err = ec.repo.StepRun().QueueStepRun(ctx, tenantId, stepRunId, updateStepOpts)

	if err != nil {
		if errors.Is(err, repository.ErrStepRunIsNotPending) {
			ec.l.Debug().Msgf("step run %s is not pending, skipping scheduling", stepRunId)
			return nil
		}

		return ec.a.WrapErr(fmt.Errorf("could not update step run: %w", err), errData)
	}

	return ec.a.WrapErr(ec.scheduleStepRun(ctx, tenantId, stepRun), errData)
}

func (ec *JobsControllerImpl) scheduleStepRun(ctx context.Context, tenantId string, stepRun *dbsqlc.GetStepRunForEngineRow) error {
	ctx, span := telemetry.NewSpan(ctx, "schedule-step-run")
	defer span.End()

	stepRunId := sqlchelpers.UUIDToStr(stepRun.StepRun.ID)

	selectedWorkerId, dispatcherId, err := ec.repo.StepRun().AssignStepRunToWorker(ctx, stepRun)

	if err != nil {
		if errors.Is(err, repository.ErrNoWorkerAvailable) {
			ec.l.Debug().Msgf("no worker available for step run %s, requeueing", stepRunId)
			return nil
		}

		if errors.Is(err, repository.ErrRateLimitExceeded) {
			ec.l.Debug().Msgf("rate limit exceeded for step run %s, requeueing", stepRunId)
			return nil
		}

		return fmt.Errorf("could not assign step run to worker: %w", err)
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
		Event: &repository.CreateStepRunEventOpts{
			EventReason: repository.StepRunEventReasonPtr(dbsqlc.StepRunEventReasonSTARTED),
			EventMessage: repository.StringPtr(
				fmt.Sprintf("Step run started running on %s", startedAt.Format(time.RFC1123)),
			),
		},
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
		stepOutput = []byte(payload.StepOutputData)
	}

	stepRun, updateInfo, err := ec.repo.StepRun().UpdateStepRun(ctx, metadata.TenantId, payload.StepRunId, &repository.UpdateStepRunOpts{
		FinishedAt: &finishedAt,
		Status:     repository.StepRunStatusPtr(db.StepRunStatusSucceeded),
		Output:     stepOutput,
		Event: &repository.CreateStepRunEventOpts{
			EventReason: repository.StepRunEventReasonPtr(dbsqlc.StepRunEventReasonFINISHED),
			EventMessage: repository.StringPtr(
				fmt.Sprintf("Step run finished on %s", finishedAt.Format(time.RFC1123)),
			)},
	})

	if err != nil {
		return fmt.Errorf("could not update step run: %w", err)
	}

	defer ec.handleStepRunUpdateInfo(stepRun, updateInfo)

	// queue the next step runs
	jobRunId := sqlchelpers.UUIDToStr(stepRun.JobRunId)
	stepRunId := sqlchelpers.UUIDToStr(stepRun.StepRun.ID)

	nextStepRuns, err := ec.repo.StepRun().ListStartableStepRuns(ctx, metadata.TenantId, jobRunId, &stepRunId)

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

	failedAt, err := time.Parse(time.RFC3339, payload.FailedAt)
	if err != nil {
		return fmt.Errorf("could not parse failed at: %w", err)
	}

	return ec.failStepRun(ctx, metadata.TenantId, payload.StepRunId, payload.Error, failedAt)
}

func (ec *JobsControllerImpl) failStepRun(ctx context.Context, tenantId, stepRunId, errorReason string, failedAt time.Time) error {
	stepRun, err := ec.repo.StepRun().GetStepRunForEngine(ctx, tenantId, stepRunId)

	if err != nil {
		return fmt.Errorf("could not get step run: %w", err)
	}

	// determine if step run should be retried or not
	shouldRetry := stepRun.StepRun.RetryCount < stepRun.StepRetries

	status := db.StepRunStatusFailed

	if shouldRetry {
		status = db.StepRunStatusPending
	}

	scheduleTimeoutAt := getScheduleTimeout(stepRun)

	eventMessage := fmt.Sprintf("Step run failed on %s", failedAt.Format(time.RFC1123))

	eventReason := dbsqlc.StepRunEventReasonFAILED

	if errorReason == "TIMED_OUT" {
		eventReason = dbsqlc.StepRunEventReasonTIMEDOUT
		eventMessage = fmt.Sprintf("Step exceeded timeout duration (%s)", stepRun.StepTimeout.String)
	}

	if shouldRetry {
		eventMessage += ", and will be retried."
	} else {
		eventMessage += "."
	}

	stepRun, updateInfo, err := ec.repo.StepRun().UpdateStepRun(ctx, tenantId, stepRunId, &repository.UpdateStepRunOpts{
		FinishedAt:        &failedAt,
		Error:             &errorReason,
		Status:            repository.StepRunStatusPtr(status),
		ScheduleTimeoutAt: &scheduleTimeoutAt,
		Event: &repository.CreateStepRunEventOpts{
			EventReason:   repository.StepRunEventReasonPtr(eventReason),
			EventMessage:  repository.StringPtr(eventMessage),
			EventSeverity: repository.StepRunEventSeverityPtr(dbsqlc.StepRunEventSeverityCRITICAL),
		},
	})

	if err != nil {
		return fmt.Errorf("could not update step run: %w", err)
	}

	defer ec.handleStepRunUpdateInfo(stepRun, updateInfo)

	attemptCancel := false

	if errorReason == "TIMED_OUT" {
		attemptCancel = true
	}

	if !stepRun.StepRun.WorkerId.Valid {
		// this is not a fatal error
		ec.l.Warn().Msgf("step run %s has no worker id, skipping cancellation", stepRunId)
		attemptCancel = false
	}

	// Attempt to cancel the previous running step run
	if attemptCancel {

		workerId := sqlchelpers.UUIDToStr(stepRun.StepRun.WorkerId)

		worker, err := ec.repo.Worker().GetWorkerForEngine(ctx, tenantId, workerId)

		if err != nil {
			return fmt.Errorf("could not get worker: %w", err)
		} else if !worker.DispatcherId.Valid {
			return fmt.Errorf("worker has no dispatcher id")
		}

		dispatcherId := sqlchelpers.UUIDToStr(worker.DispatcherId)

		err = ec.mq.AddMessage(
			ctx,
			msgqueue.QueueTypeFromDispatcherID(dispatcherId),
			stepRunCancelledTask(tenantId, stepRunId, workerId, dispatcherId, *repository.StringPtr(eventMessage)),
		)

		if err != nil {
			return fmt.Errorf("could not add job assigned task to task queue: %w", err)
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

	return ec.failStepRun(ctx, metadata.TenantId, payload.StepRunId, "TIMED_OUT", time.Now().UTC())
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
		Event: &repository.CreateStepRunEventOpts{
			EventReason: repository.StepRunEventReasonPtr(dbsqlc.StepRunEventReasonCANCELLED),
			EventMessage: repository.StringPtr(
				fmt.Sprintf("Step run was cancelled on %s for the following reason: %s", now.Format(time.RFC1123), reason),
			),
			EventSeverity: repository.StepRunEventSeverityPtr(dbsqlc.StepRunEventSeverityWARNING),
		},
	})

	if err != nil {
		return fmt.Errorf("could not update step run: %w", err)
	}

	defer ec.handleStepRunUpdateInfo(stepRun, updateInfo)

	if !stepRun.StepRun.WorkerId.Valid {
		// this is not a fatal error
		ec.l.Debug().Msgf("step run %s has no worker id, skipping cancellation", stepRunId)

		return nil
	}

	workerId := sqlchelpers.UUIDToStr(stepRun.StepRun.WorkerId)

	worker, err := ec.repo.Worker().GetWorkerForEngine(ctx, tenantId, workerId)

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
			recoveryutils.RecoverWithAlert(ec.l, ec.a, r) // nolint:errcheck
		}
	}()

	if updateInfo.WorkflowRunFinalState && stepRun.JobKind == dbsqlc.JobKindDEFAULT {
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

func getScheduleTimeout(stepRun *dbsqlc.GetStepRunForEngineRow) time.Time {
	var timeoutDuration time.Duration

	// get the schedule timeout from the step
	stepScheduleTimeout := stepRun.StepScheduleTimeout

	if stepScheduleTimeout != "" {
		timeoutDuration, _ = time.ParseDuration(stepScheduleTimeout)
	} else {
		timeoutDuration = defaults.DefaultScheduleTimeout
	}

	return time.Now().UTC().Add(timeoutDuration)
}
