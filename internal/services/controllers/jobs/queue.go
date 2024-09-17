package jobs

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/controllers/partition"
	"github.com/hatchet-dev/hatchet/internal/services/shared/recoveryutils"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	hatcheterrors "github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

type queue struct {
	mq   msgqueue.MessageQueue
	l    *zerolog.Logger
	repo repository.EngineRepository
	dv   datautils.DataDecoderValidator
	s    gocron.Scheduler
	a    *hatcheterrors.Wrapped
	p    *partition.Partition

	// a custom queue logger
	ql *zerolog.Logger

	tenantQueueOperations   sync.Map
	updateStepRunOperations sync.Map
}

func newQueue(
	mq msgqueue.MessageQueue,
	l *zerolog.Logger,
	ql *zerolog.Logger,
	repo repository.EngineRepository,
	dv datautils.DataDecoderValidator,
	a *hatcheterrors.Wrapped,
	p *partition.Partition,
) (*queue, error) {
	s, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))

	if err != nil {
		return nil, fmt.Errorf("could not create scheduler: %w", err)
	}

	return &queue{
		mq:   mq,
		l:    l,
		repo: repo,
		dv:   dv,
		s:    s,
		a:    a,
		p:    p,
		ql:   ql,
	}, nil
}

type operation struct {
	mu             sync.RWMutex
	shouldContinue bool
	isRunning      bool
	tenantId       string
	lastRun        time.Time
	description    string
	timeout        time.Duration
}

func (o *operation) runOrContinue(l *zerolog.Logger, ql *zerolog.Logger, scheduler func(context.Context, string) (bool, error)) {
	o.setContinue(true)
	o.run(l, ql, scheduler)
}

func (o *operation) run(l *zerolog.Logger, ql *zerolog.Logger, scheduler func(context.Context, string) (bool, error)) {
	if !o.setRunning(true, ql) {
		return
	}

	go func() {
		defer func() {
			o.setRunning(false, ql)
		}()

		f := func() {
			o.setContinue(false)

			ctx, cancel := context.WithTimeout(context.Background(), o.timeout)
			defer cancel()

			shouldContinue, err := scheduler(ctx, o.tenantId)

			if err != nil {
				l.Err(err).Msgf("could not %s", o.description)
				return
			}

			// if a continue was set during execution of the scheduler, we'd like to continue no matter what.
			// if a continue was not set, we'd like to set it to the value returned by the scheduler.
			if !o.getContinue() {
				o.setContinue(shouldContinue)
			}
		}

		f()

		for o.getContinue() {
			f()
		}
	}()
}

// setRunning sets the running state of the operation and returns true if the state was changed,
// false if the state was not changed.
func (o *operation) setRunning(isRunning bool, ql *zerolog.Logger) bool {
	o.mu.Lock()
	defer o.mu.Unlock()

	if isRunning == o.isRunning {
		return false
	}

	if isRunning {
		ql.Info().Str("tenant_id", o.tenantId).TimeDiff("last_run", time.Now(), o.lastRun).Msg(o.description)

		o.lastRun = time.Now()
	}

	o.isRunning = isRunning

	return true
}

func (o *operation) setContinue(shouldContinue bool) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.shouldContinue = shouldContinue
}

func (o *operation) getContinue() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return o.shouldContinue
}

func (q *queue) Start() (func() error, error) {
	ctx, cancel := context.WithCancel(context.Background())

	wg := sync.WaitGroup{}

	_, err := q.s.NewJob(
		gocron.DurationJob(time.Second*1),
		gocron.NewTask(
			q.runTenantQueues(ctx),
		),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not schedule step run reassign: %w", err)
	}

	_, err = q.s.NewJob(
		gocron.DurationJob(time.Second*1),
		gocron.NewTask(
			q.runTenantUpdateStepRuns(ctx),
		),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not schedule step run update: %w", err)
	}

	q.s.Start()

	f := func(task *msgqueue.Message) error {
		wg.Add(1)
		defer wg.Done()

		err := q.handleTask(context.Background(), task)
		if err != nil {
			q.l.Error().Err(err).Msg("could not handle job task")
			return q.a.WrapErr(fmt.Errorf("could not handle job task: %w", err), map[string]interface{}{"task_id": task.ID}) // nolint: errcheck
		}

		return nil
	}

	cleanupQueue, err := q.mq.Subscribe(msgqueue.QueueTypeFromPartitionID(q.p.GetControllerPartitionId()), f, msgqueue.NoOpHook)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not subscribe to job processing queue: %w", err)
	}

	cleanup := func() error {
		cancel()

		if err := cleanupQueue(); err != nil {
			return fmt.Errorf("could not cleanup job processing queue: %w", err)
		}

		if err := q.s.Shutdown(); err != nil {
			return fmt.Errorf("could not shutdown scheduler: %w", err)
		}

		wg.Wait()

		return nil
	}

	return cleanup, nil
}

func (q *queue) handleTask(ctx context.Context, task *msgqueue.Message) (err error) {
	defer func() {
		if r := recover(); r != nil {
			recoverErr := recoveryutils.RecoverWithAlert(q.l, q.a, r)

			if recoverErr != nil {
				err = recoverErr
			}
		}
	}()

	if task.ID == "check-tenant-queue" {
		return q.handleCheckQueue(ctx, task)
	}

	return fmt.Errorf("unknown task: %s", task.ID)
}

func (q *queue) handleCheckQueue(ctx context.Context, task *msgqueue.Message) error {
	_, span := telemetry.NewSpanWithCarrier(ctx, "handle-check-queue", task.OtelCarrier)
	defer span.End()

	metadata := tasktypes.CheckTenantQueueMetadata{}

	err := q.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode check queue metadata: %w", err)
	}

	// if this tenant is registered, then we should check the queue
	if opInt, ok := q.tenantQueueOperations.Load(metadata.TenantId); ok {
		op := opInt.(*operation)

		op.runOrContinue(q.l, q.ql, q.scheduleStepRuns)
	}

	if opInt, ok := q.updateStepRunOperations.Load(metadata.TenantId); ok {
		op := opInt.(*operation)

		op.runOrContinue(q.l, q.ql, q.processStepRunUpdates)
	}

	return nil
}

func (q *queue) runTenantQueues(ctx context.Context) func() {
	return func() {
		q.l.Debug().Msgf("partition: checking step run requeue")

		// list all tenants
		tenants, err := q.repo.Tenant().ListTenantsByControllerPartition(ctx, q.p.GetControllerPartitionId())

		if err != nil {
			q.l.Err(err).Msg("could not list tenants")
			return
		}

		for i := range tenants {
			tenantId := sqlchelpers.UUIDToStr(tenants[i].ID)

			q.getQueueOperation(tenantId).run(q.l, q.ql, q.scheduleStepRuns)
		}
	}
}

func (q *queue) scheduleStepRuns(ctx context.Context, tenantId string) (bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "schedule-step-runs")
	defer span.End()

	dbCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	queueResults, err := q.repo.StepRun().QueueStepRuns(dbCtx, q.ql, tenantId)

	if err != nil {
		return false, fmt.Errorf("could not queue step runs: %w", err)
	}

	for _, queueResult := range queueResults.Queued {
		// send a task to the dispatcher
		innerErr := q.mq.AddMessage(
			ctx,
			msgqueue.QueueTypeFromDispatcherID(queueResult.DispatcherId),
			stepRunAssignedTask(tenantId, queueResult.StepRunId, queueResult.WorkerId, queueResult.DispatcherId),
		)

		if innerErr != nil {
			err = multierror.Append(err, fmt.Errorf("could not send queued step run: %w", innerErr))
		}
	}

	for _, toCancel := range queueResults.SchedulingTimedOut {
		innerErr := q.mq.AddMessage(
			ctx,
			msgqueue.JOB_PROCESSING_QUEUE,
			getStepRunCancelTask(
				tenantId,
				toCancel,
				"SCHEDULING_TIMED_OUT",
			),
		)

		if innerErr != nil {
			err = multierror.Append(err, fmt.Errorf("could not send cancel step run event: %w", innerErr))
		}
	}

	return queueResults.Continue, err
}

func (q *queue) getQueueOperation(tenantId string) *operation {
	op, ok := q.tenantQueueOperations.Load(tenantId)

	if !ok {
		op = &operation{
			tenantId:    tenantId,
			lastRun:     time.Now(),
			description: "scheduling step runs",
			timeout:     30 * time.Second,
		}

		q.tenantQueueOperations.Store(tenantId, op)
	}

	return op.(*operation)
}

func (q *queue) getUpdateStepRunOperation(tenantId string) *operation {
	op, ok := q.updateStepRunOperations.Load(tenantId)

	if !ok {
		op = &operation{
			tenantId:    tenantId,
			lastRun:     time.Now(),
			description: "updating step runs",
			timeout:     300 * time.Second,
		}

		q.updateStepRunOperations.Store(tenantId, op)
	}

	return op.(*operation)
}

func (q *queue) runTenantUpdateStepRuns(ctx context.Context) func() {
	return func() {
		q.l.Debug().Msgf("partition: updating step run statuses")

		// list all tenants
		tenants, err := q.repo.Tenant().ListTenantsByControllerPartition(ctx, q.p.GetControllerPartitionId())

		if err != nil {
			q.l.Err(err).Msg("could not list tenants")
			return
		}

		for i := range tenants {
			tenantId := sqlchelpers.UUIDToStr(tenants[i].ID)

			q.getUpdateStepRunOperation(tenantId).run(q.l, q.ql, q.processStepRunUpdates)
		}
	}
}

func (q *queue) processStepRunUpdates(ctx context.Context, tenantId string) (bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "process-worker-semaphores")
	defer span.End()

	dbCtx, cancel := context.WithTimeout(ctx, 300*time.Second)
	defer cancel()

	res, err := q.repo.StepRun().ProcessStepRunUpdates(dbCtx, q.ql, tenantId)

	if err != nil {
		return false, fmt.Errorf("could not process step run updates: %w", err)
	}

	// for all succeeded step runs, check for startable child step runs
	err = MakeBatched(20, res.SucceededStepRuns, func(group []*dbsqlc.GetStepRunForEngineRow) error {
		for _, stepRun := range group {
			if stepRun.SRChildCount == 0 {
				continue
			}

			// queue the next step runs
			tenantId := sqlchelpers.UUIDToStr(stepRun.SRTenantId)
			jobRunId := sqlchelpers.UUIDToStr(stepRun.JobRunId)
			stepRunId := stepRun.SRID

			nextStepRuns, err := q.repo.StepRun().ListStartableStepRuns(ctx, tenantId, jobRunId, &stepRunId)

			if err != nil {
				q.l.Error().Err(err).Msg("could not list startable step runs")
				continue
			}

			for _, nextStepRun := range nextStepRuns {
				err := q.mq.AddMessage(
					context.Background(),
					msgqueue.JOB_PROCESSING_QUEUE,
					tasktypes.StepRunQueuedToTask(nextStepRun),
				)

				if err != nil {
					q.l.Error().Err(err).Msg("could not queue next step run")
				}
			}
		}

		return nil
	})

	if err != nil {
		return false, fmt.Errorf("could not process succeeded step runs: %w", err)
	}

	// for all finished workflow runs, send a message
	for _, finished := range res.CompletedWorkflowRuns {
		workflowRunId := sqlchelpers.UUIDToStr(finished.ID)
		status := string(finished.Status)

		err := q.mq.AddMessage(
			context.Background(),
			msgqueue.WORKFLOW_PROCESSING_QUEUE,
			tasktypes.WorkflowRunFinishedToTask(
				tenantId,
				workflowRunId,
				status,
			),
		)

		if err != nil {
			q.l.Error().Err(err).Msg("could not add workflow run finished task to task queue")
		}
	}

	return res.Continue, nil
}

func getStepRunCancelTask(tenantId string, stepRunId int64, reason string) *msgqueue.Message {
	payload, _ := datautils.ToJSONMap(tasktypes.StepRunCancelTaskPayload{
		StepRunId:       stepRunId,
		CancelledReason: reason,
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.StepRunCancelTaskMetadata{
		TenantId: tenantId,
	})

	return &msgqueue.Message{
		ID:       "step-run-cancel",
		Payload:  payload,
		Metadata: metadata,
		Retries:  3,
	}
}
