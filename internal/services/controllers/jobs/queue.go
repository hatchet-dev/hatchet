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
	"github.com/hatchet-dev/hatchet/internal/queueutils"
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

	tenantQueueOperations   *queueutils.OperationPool
	updateStepRunOperations *queueutils.OperationPool
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

	q := &queue{
		mq:   mq,
		l:    l,
		repo: repo,
		dv:   dv,
		s:    s,
		a:    a,
		p:    p,
		ql:   ql,
	}

	q.tenantQueueOperations = queueutils.NewOperationPool(ql, time.Second*5, "check tenant queue", q.scheduleStepRuns)
	q.updateStepRunOperations = queueutils.NewOperationPool(ql, time.Second*30, "update step runs", q.processStepRunUpdates)

	return q, nil
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

	cleanupQueue, err := q.mq.Subscribe(msgqueue.QueueTypeFromPartitionIDAndController(q.p.GetControllerPartitionId(), msgqueue.JobController), f, msgqueue.NoOpHook)

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
	q.tenantQueueOperations.RunOrContinue(metadata.TenantId)
	q.updateStepRunOperations.RunOrContinue(metadata.TenantId)

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

			q.tenantQueueOperations.RunOrContinue(tenantId)
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

			q.updateStepRunOperations.RunOrContinue(tenantId)
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
			stepRunId := sqlchelpers.UUIDToStr(stepRun.SRID)

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
