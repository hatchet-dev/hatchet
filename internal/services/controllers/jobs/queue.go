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

	tenantOperations sync.Map
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
	lastSchedule   time.Time
}

func (o *operation) run(l *zerolog.Logger, ql *zerolog.Logger, scheduler func(context.Context, string) (bool, error)) {
	if o.getRunning() {
		return
	}

	ql.Info().Str("tenant_id", o.tenantId).TimeDiff("last_run", time.Now(), o.lastRun).Msg("running tenant queue")
	o.setRunning(true)

	go func() {
		defer func() {
			o.setRunning(false)
		}()

		f := func() {
			ql.Info().Str("tenant_id", o.tenantId).TimeDiff("last_schedule", time.Now(), o.lastSchedule).Msg("running scheduling")
			o.lastSchedule = time.Now()

			o.setContinue(false)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			shouldContinue, err := scheduler(ctx, o.tenantId)

			if err != nil {
				l.Err(err).Msgf("could not schedule step runs for tenant")
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

func (o *operation) setRunning(isRunning bool) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if isRunning {
		o.lastRun = time.Now()
	}

	o.isRunning = isRunning
}

func (o *operation) getRunning() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return o.isRunning
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
	if opInt, ok := q.tenantOperations.Load(metadata.TenantId); ok {
		op := opInt.(*operation)

		op.setContinue(true)
		op.run(q.l, q.ql, q.scheduleStepRuns)
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

			var op *operation

			opInt, ok := q.tenantOperations.Load(tenantId)

			if !ok {
				op = &operation{
					tenantId:     tenantId,
					lastRun:      time.Now(),
					lastSchedule: time.Now(),
				}

				q.tenantOperations.Store(tenantId, op)
			} else {
				op = opInt.(*operation)
			}

			op.run(q.l, q.ql, q.scheduleStepRuns)
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

func getStepRunCancelTask(tenantId, stepRunId, reason string) *msgqueue.Message {
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
