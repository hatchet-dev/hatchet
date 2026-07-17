package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/controllers/olap/signal"
	"github.com/hatchet-dev/hatchet/internal/services/partition"
	"github.com/hatchet-dev/hatchet/internal/services/shared/durable"
	"github.com/hatchet-dev/hatchet/internal/services/shared/recoveryutils"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/pkg/config/shared"
	hatcheterrors "github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	repov1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	v1 "github.com/hatchet-dev/hatchet/pkg/scheduling/v1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

var (
	eventTypeWaitingForBatch = sqlcv1.V1EventTypeOlapWAITINGFORBATCH
	eventTypeBatchFlushed    = sqlcv1.V1EventTypeOlapBATCHFLUSHED
)

type SchedulerOpt func(*SchedulerOpts)

type SchedulerOpts struct {
	mq          msgqueue.MessageQueue
	l           *zerolog.Logger
	repov1      repov1.Repository
	dv          datautils.DataDecoderValidator
	alerter     hatcheterrors.Alerter
	p           *partition.Partition
	queueLogger *zerolog.Logger
	pool        *v1.SchedulingPool
	promGate    *prometheus.Gate
}

func defaultSchedulerOpts() *SchedulerOpts {
	l := logger.NewDefaultLogger("scheduler")
	alerter := hatcheterrors.NoOpAlerter{}

	queueLogger := logger.NewDefaultLogger("queue")

	return &SchedulerOpts{
		l:           &l,
		dv:          datautils.NewDataDecoderValidator(),
		alerter:     alerter,
		queueLogger: &queueLogger,
	}
}

func WithMessageQueue(mq msgqueue.MessageQueue) SchedulerOpt {
	return func(opts *SchedulerOpts) {
		opts.mq = mq
	}
}

func WithLogger(l *zerolog.Logger) SchedulerOpt {
	return func(opts *SchedulerOpts) {
		opts.l = l
	}
}

func WithQueueLoggerConfig(lc *shared.LoggerConfigFile) SchedulerOpt {
	return func(opts *SchedulerOpts) {
		l := logger.NewStdErr(lc, "queue")
		opts.queueLogger = &l
	}
}

func WithAlerter(a hatcheterrors.Alerter) SchedulerOpt {
	return func(opts *SchedulerOpts) {
		opts.alerter = a
	}
}

func WithRepository(r repov1.Repository) SchedulerOpt {
	return func(opts *SchedulerOpts) {
		opts.repov1 = r
	}
}

func WithPartition(p *partition.Partition) SchedulerOpt {
	return func(opts *SchedulerOpts) {
		opts.p = p
	}
}

func WithDataDecoderValidator(dv datautils.DataDecoderValidator) SchedulerOpt {
	return func(opts *SchedulerOpts) {
		opts.dv = dv
	}
}

func WithSchedulerPool(s *v1.SchedulingPool) SchedulerOpt {
	return func(opts *SchedulerOpts) {
		opts.pool = s
	}
}

func WithPrometheusGate(gate *prometheus.Gate) SchedulerOpt {
	return func(opts *SchedulerOpts) {
		opts.promGate = gate
	}
}

type Scheduler struct {
	mq        msgqueue.MessageQueue
	pubBuffer *msgqueue.MQPubBuffer
	l         *zerolog.Logger
	repov1    repov1.Repository
	dv        datautils.DataDecoderValidator
	s         gocron.Scheduler
	a         *hatcheterrors.Wrapped
	p         *partition.Partition

	// a custom queue logger
	ql *zerolog.Logger

	pool *v1.SchedulingPool

	signaler *signal.OLAPSignaler

	tasksWithNoWorkerCache *expirable.LRU[string, struct{}]
}

func New(
	fs ...SchedulerOpt,
) (*Scheduler, error) {
	opts := defaultSchedulerOpts()

	for _, f := range fs {
		f(opts)
	}

	if opts.mq == nil {
		return nil, fmt.Errorf("task queue is required. use WithMessageQueue")
	}

	if opts.repov1 == nil {
		return nil, fmt.Errorf("v2 repository is required. use WithV2Repository")
	}

	if opts.p == nil {
		return nil, fmt.Errorf("partition is required. use WithPartition")
	}

	if opts.pool == nil {
		return nil, fmt.Errorf("scheduler is required. use WithSchedulerPool")
	}

	s, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))

	if err != nil {
		return nil, fmt.Errorf("could not create scheduler: %w", err)
	}

	a := hatcheterrors.NewWrapped(opts.alerter)
	a.WithData(map[string]interface{}{"service": "scheduler"})

	pubBuffer := msgqueue.NewMQPubBuffer(opts.mq)

	// TODO: replace with config or pull into a constant
	tasksWithNoWorkerCache := expirable.NewLRU(10000, func(string, struct{}) {}, 5*time.Minute)

	signaler := signal.NewOLAPSignaler(opts.mq, opts.repov1, opts.l, pubBuffer, opts.promGate)

	q := &Scheduler{
		mq:                     opts.mq,
		pubBuffer:              pubBuffer,
		l:                      opts.l,
		repov1:                 opts.repov1,
		dv:                     opts.dv,
		s:                      s,
		a:                      a,
		p:                      opts.p,
		ql:                     opts.queueLogger,
		pool:                   opts.pool,
		tasksWithNoWorkerCache: tasksWithNoWorkerCache,
		signaler:               signaler,
	}

	return q, nil
}

func (s *Scheduler) Start() (func() error, error) {
	cleanupDLQ, err := s.mq.Subscribe(msgqueue.DISPATCHER_DEAD_LETTER_QUEUE, s.handleDeadLetteredMessages, msgqueue.NoOpHook)

	if err != nil {
		return nil, fmt.Errorf("could not start subscribe to dispatcher dead letter queue: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	wg := sync.WaitGroup{}

	_, err = s.s.NewJob(
		gocron.DurationJob(time.Second*1),
		gocron.NewTask(
			s.runSetTenants(ctx),
		),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not schedule tenant set queues: %w", err)
	}

	s.s.Start()

	postAck := func(task *msgqueue.Message) error {
		wg.Add(1)
		defer wg.Done()

		err := s.handleTask(context.Background(), task)
		if err != nil {
			s.l.Error().Ctx(ctx).Err(err).Msg("could not handle job task")
			return s.a.WrapErr(fmt.Errorf("could not handle job task: %w", err), map[string]interface{}{"task_id": task.ID}) // nolint: errcheck
		}

		return nil
	}

	cleanupQueue, err := s.mq.Subscribe(
		msgqueue.QueueTypeFromPartitionIDAndController(s.p.GetSchedulerPartitionId(), msgqueue.Scheduler),
		msgqueue.NoOpHook, // the only handler is to check the queue, so we acknowledge immediately with the NoOpHook
		postAck,
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not subscribe to job processing queue: %w", err)
	}

	queueResults := s.pool.GetResultsCh()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case res := <-queueResults:
				s.l.Debug().Ctx(ctx).Msgf("partition: received queue results")

				if res == nil {
					continue
				}

				go func(results *v1.QueueResults) {
					err := s.scheduleStepRuns(ctx, results.TenantId, results)

					if err != nil {
						s.l.Error().Ctx(ctx).Err(err).Msg("could not schedule step runs")
					}
				}(res)
			}
		}
	}()

	concurrencyResults := s.pool.GetConcurrencyResultsCh()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case res := <-concurrencyResults:
				s.l.Debug().Ctx(ctx).Msgf("partition: received concurrency results")

				if res == nil {
					continue
				}

				go s.notifyAfterConcurrency(ctx, res.TenantId, res)
			}
		}
	}()

	cleanup := func() error {
		cancel()

		if err := cleanupQueue(); err != nil {
			return fmt.Errorf("could not cleanup job processing queue: %w", err)
		}

		if err := cleanupDLQ(); err != nil {
			return fmt.Errorf("could not cleanup message queue buffer: %w", err)
		}

		if err := s.s.Shutdown(); err != nil {
			return fmt.Errorf("could not shutdown scheduler: %w", err)
		}

		wg.Wait()

		s.pubBuffer.Stop()

		return nil
	}

	return cleanup, nil
}

func (s *Scheduler) handleTask(ctx context.Context, task *msgqueue.Message) (err error) {
	defer func() {
		if r := recover(); r != nil {
			recoverErr := recoveryutils.RecoverWithAlert(s.l, s.a, r)

			if recoverErr != nil {
				err = recoverErr
			}
		}
	}()

	switch task.ID {
	case msgqueue.MsgIDCheckTenantQueue:
		return s.handleCheckQueue(ctx, task)
	case msgqueue.MsgIDNewWorker:
		return s.handleNewWorker(ctx, task)
	case msgqueue.MsgIDNewQueue:
		return s.handleNewQueue(ctx, task)
	case msgqueue.MsgIDNewConcurrencyStrategy:
		return s.handleNewConcurrencyStrategy(ctx, task)
	}

	return fmt.Errorf("unknown task: %s", task.ID)
}

func (s *Scheduler) handleCheckQueue(ctx context.Context, msg *msgqueue.Message) error {
	ctx, span := telemetry.NewSpanWithCarrier(ctx, "handle-check-queue", msg.OtelCarrier)
	defer span.End()

	payloads := msgqueue.JSONConvert[tasktypes.CheckTenantQueuesPayload](msg.Payloads)

	for _, payload := range payloads {
		if len(payload.StrategyIds) > 0 {
			s.pool.NotifyConcurrency(ctx, msg.TenantID, payload.StrategyIds)
		}

		if len(payload.QueueNames) > 0 {
			s.pool.NotifyQueues(ctx, msg.TenantID, payload.QueueNames)
		}

		if payload.SlotsReleased {
			s.pool.Replenish(ctx, msg.TenantID)
		}
	}

	return nil
}

func (s *Scheduler) handleNewWorker(ctx context.Context, msg *msgqueue.Message) error {
	ctx, span := telemetry.NewSpanWithCarrier(ctx, "handle-new-worker", msg.OtelCarrier)
	defer span.End()

	payloads := msgqueue.JSONConvert[tasktypes.NewWorkerPayload](msg.Payloads)

	for _, payload := range payloads {
		s.pool.NotifyNewWorker(ctx, msg.TenantID, payload.WorkerId)
	}

	return nil
}

func (s *Scheduler) handleNewQueue(ctx context.Context, msg *msgqueue.Message) error {
	ctx, span := telemetry.NewSpanWithCarrier(ctx, "handle-new-queue", msg.OtelCarrier)
	defer span.End()

	payloads := msgqueue.JSONConvert[tasktypes.NewQueuePayload](msg.Payloads)

	for _, payload := range payloads {
		s.pool.NotifyNewQueue(ctx, msg.TenantID, payload.QueueName)
	}

	return nil
}

func (s *Scheduler) handleNewConcurrencyStrategy(ctx context.Context, msg *msgqueue.Message) error {
	ctx, span := telemetry.NewSpanWithCarrier(ctx, "handle-new-concurrency-strategy", msg.OtelCarrier)
	defer span.End()

	payloads := msgqueue.JSONConvert[tasktypes.NewConcurrencyStrategyPayload](msg.Payloads)

	for _, payload := range payloads {
		s.pool.NotifyNewConcurrencyStrategy(ctx, msg.TenantID, payload.StrategyId)
	}

	return nil
}

func (s *Scheduler) runSetTenants(ctx context.Context) func() {
	return func() {
		s.l.Debug().Ctx(ctx).Msgf("partition: checking step run requeue")

		// list all tenants
		tenants, err := s.repov1.Tenant().ListTenantsBySchedulerPartition(ctx, s.p.GetSchedulerPartitionId())

		if err != nil {
			s.l.Err(err).Ctx(ctx).Msg("could not list tenants")
			return
		}

		s.pool.SetTenants(tenants)
	}
}

func (s *Scheduler) scheduleStepRuns(ctx context.Context, tenantId uuid.UUID, res *v1.QueueResults) error {
	ctx, span := telemetry.NewSpan(ctx, "schedule-step-runs")
	defer span.End()

	var outerErr error

	if len(res.Buffered) > 0 {
		if err := s.emitBatchWaitingEvents(ctx, tenantId, res.Buffered); err != nil {
			outerErr = multierror.Append(outerErr, err)
		}
	}

	// bulk assign step runs
	if len(res.Assigned) > 0 {
		dispatcherIdToWorkerIdsToStepRuns := make(map[uuid.UUID]map[uuid.UUID][]int64)

		workerIds := make([]uuid.UUID, 0)

		for _, assigned := range res.Assigned {
			workerIds = append(workerIds, assigned.WorkerId)
		}

		workerIdToDispatcherId, workersWithoutDispatchers, err := s.repov1.Workers().GetDispatcherIdsForWorkers(ctx, tenantId, workerIds)

		if err != nil {
			s.internalRetry(ctx, tenantId, res.Assigned...)

			return fmt.Errorf("could not list dispatcher ids for workers: %w. attempting internal retry", err)
		}

		assignedMsgs := make([]*msgqueue.Message, 0)
		batchAssignments := make([]*repov1.AssignedItem, 0)

		invCountOpts := make([]repov1.IdInsertedAt, 0, len(res.Assigned))

		for _, a := range res.Assigned {
			if a.IsDurable {
				invCountOpts = append(invCountOpts, repov1.IdInsertedAt{
					ID:         a.QueueItem.TaskID,
					InsertedAt: a.QueueItem.TaskInsertedAt,
				})
			}
		}

		invocationCounts := make(map[repov1.IdInsertedAt]*int32, len(invCountOpts))

		if len(invCountOpts) > 0 {
			invocationCounts, err = s.repov1.DurableEvents().GetDurableTaskInvocationCounts(ctx, tenantId, invCountOpts)
			if err != nil {
				s.internalRetry(ctx, tenantId, res.Assigned...)

				return fmt.Errorf("could not get durable task invocation counts for assigned tasks: %w", err)
			}
		}

		for _, bulkAssigned := range res.Assigned {
			if bulkAssigned != nil && bulkAssigned.Batch != nil && bulkAssigned.QueueItem != nil {
				batchAssignments = append(batchAssignments, bulkAssigned)
				continue
			}

			_, hasNoDispatcher := workersWithoutDispatchers[bulkAssigned.WorkerId]
			dispatcherId, ok := workerIdToDispatcherId[bulkAssigned.WorkerId]

			if hasNoDispatcher || !ok {
				s.l.Error().Ctx(ctx).Msg("could not assign step run to worker: no dispatcher id. attempting internal retry.")

				s.internalRetry(ctx, tenantId, bulkAssigned)

				continue
			}

			if _, ok := dispatcherIdToWorkerIdsToStepRuns[dispatcherId]; !ok {
				dispatcherIdToWorkerIdsToStepRuns[dispatcherId] = make(map[uuid.UUID][]int64)
			}

			workerId := bulkAssigned.WorkerId

			if _, ok := dispatcherIdToWorkerIdsToStepRuns[dispatcherId][workerId]; !ok {
				dispatcherIdToWorkerIdsToStepRuns[dispatcherId][workerId] = make([]int64, 0)
			}

			if !bulkAssigned.IsAssignedLocally {
				dispatcherIdToWorkerIdsToStepRuns[dispatcherId][workerId] = append(dispatcherIdToWorkerIdsToStepRuns[dispatcherId][workerId], bulkAssigned.QueueItem.TaskID)
			}

			taskId := bulkAssigned.QueueItem.TaskID

			var durableInvCount int32
			if count, ok := invocationCounts[repov1.IdInsertedAt{ID: taskId, InsertedAt: bulkAssigned.QueueItem.TaskInsertedAt}]; ok && count != nil {
				durableInvCount = *count
			}

			assignedMsg, err := tasktypes.MonitoringEventMessageFromInternal(
				tenantId,
				tasktypes.CreateMonitoringEventPayload{
					TaskId:                 taskId,
					RetryCount:             bulkAssigned.QueueItem.RetryCount,
					DurableInvocationCount: durableInvCount,
					WorkerId:               &workerId,
					EventType:              sqlcv1.V1EventTypeOlapASSIGNED,
					EventTimestamp:         time.Now(),
				},
			)

			if err != nil {
				outerErr = multierror.Append(outerErr, fmt.Errorf("could not create monitoring event message: %w", err))
				continue
			}

			assignedMsgs = append(assignedMsgs, assignedMsg)
		}

		if len(batchAssignments) > 0 {
			if err := s.handleBatchAssignments(ctx, tenantId, batchAssignments, workerIdToDispatcherId); err != nil {
				outerErr = multierror.Append(outerErr, err)
			}
		}

		// for each dispatcher, send a bulk assigned task
		for dispatcherId, workerIdsToStepRuns := range dispatcherIdToWorkerIdsToStepRuns {
			msg, err := taskBulkAssignedTask(tenantId, workerIdsToStepRuns)

			if err != nil {
				outerErr = multierror.Append(outerErr, fmt.Errorf("could not create bulk assigned task: %w", err))
				continue
			}

			err = s.mq.SendMessage(
				ctx,
				msgqueue.QueueTypeFromDispatcherID(dispatcherId),
				msg,
			)

			if err != nil {
				outerErr = multierror.Append(outerErr, fmt.Errorf("could not send bulk assigned task: %w", err))
			}
		}

		for _, assignedMsg := range assignedMsgs {
			err = s.pubBuffer.Pub(
				ctx,
				msgqueue.OLAP_QUEUE,
				assignedMsg,
				false,
			)

			if err != nil {
				outerErr = multierror.Append(outerErr, fmt.Errorf("could not send monitoring event message: %w", err))
			}
		}
	}

	if len(res.RateLimited) > 0 {
		for _, rateLimited := range res.RateLimited {
			message := fmt.Sprintf(
				"Rate limit exceeded for key %s, attempting to consume %d units, but only had %d remaining",
				rateLimited.ExceededKey,
				rateLimited.ExceededUnits,
				rateLimited.ExceededVal,
			)

			msg, err := tasktypes.MonitoringEventMessageFromInternal(
				tenantId,
				tasktypes.CreateMonitoringEventPayload{
					TaskId:         rateLimited.TaskId,
					RetryCount:     rateLimited.RetryCount,
					EventType:      sqlcv1.V1EventTypeOlapREQUEUEDRATELIMIT,
					EventTimestamp: time.Now(),
					EventMessage:   message,
				},
			)

			if err != nil {
				outerErr = multierror.Append(outerErr, fmt.Errorf("could not create cancelled task: %w", err))
				continue
			}

			err = s.pubBuffer.Pub(
				ctx,
				msgqueue.OLAP_QUEUE,
				msg,
				false,
			)

			if err != nil {
				outerErr = multierror.Append(outerErr, fmt.Errorf("could not send cancelled task: %w", err))
			}
		}
	}

	if len(res.SchedulingTimedOut) > 0 {
		for _, schedulingTimedOut := range res.SchedulingTimedOut {
			msg, err := tasktypes.CancelledTaskMessage(
				tenantId,
				schedulingTimedOut.TaskID,
				schedulingTimedOut.TaskInsertedAt,
				schedulingTimedOut.ExternalID,
				schedulingTimedOut.WorkflowRunID,
				schedulingTimedOut.RetryCount,
				sqlcv1.V1EventTypeOlapSCHEDULINGTIMEDOUT,
				"",
				false,
			)

			if err != nil {
				outerErr = multierror.Append(outerErr, fmt.Errorf("could not create cancelled task: %w", err))
				continue
			}

			err = s.mq.SendMessage(
				ctx,
				msgqueue.TASK_PROCESSING_QUEUE,
				msg,
			)

			if err != nil {
				outerErr = multierror.Append(outerErr, fmt.Errorf("could not send cancelled task: %w", err))
			}
		}
	}

	if len(res.Unassigned) > 0 {
		for _, unassigned := range res.Unassigned {
			taskExternalId := unassigned.ExternalID.String()

			// if we have seen this task recently, don't send it again
			if _, ok := s.tasksWithNoWorkerCache.Get(taskExternalId); ok {
				s.l.Debug().Ctx(ctx).Msgf("skipping unassigned task %s as it was recently unassigned", taskExternalId)
				continue
			}

			taskId := unassigned.TaskID

			msg, err := tasktypes.MonitoringEventMessageFromInternal(
				tenantId,
				tasktypes.CreateMonitoringEventPayload{
					TaskId:         taskId,
					RetryCount:     unassigned.RetryCount,
					EventType:      sqlcv1.V1EventTypeOlapREQUEUEDNOWORKER,
					EventTimestamp: time.Now(),
				},
			)

			if err != nil {
				outerErr = multierror.Append(outerErr, fmt.Errorf("could not create cancelled task: %w", err))
				continue
			}

			err = s.pubBuffer.Pub(
				ctx,
				msgqueue.OLAP_QUEUE,
				msg,
				false,
			)

			if err != nil {
				outerErr = multierror.Append(outerErr, fmt.Errorf("could not send cancelled task: %w", err))
			}

			s.tasksWithNoWorkerCache.Add(taskExternalId, struct{}{})
		}
	}

	return outerErr
}

func (s *Scheduler) emitBatchWaitingEvents(ctx context.Context, tenantId uuid.UUID, buffered []*repov1.AssignedItem) error {
	payloads := make([]tasktypes.CreateMonitoringEventPayload, 0, len(buffered))

	for _, item := range buffered {
		if item == nil || item.QueueItem == nil || item.Batch == nil {
			continue
		}

		queueItem := item.QueueItem
		meta := item.Batch

		pending := int(meta.Pending)
		if pending <= 0 {
			pending = 1
		}

		expected := int(meta.ConfiguredBatchMaxSize)
		if expected <= 0 {
			expected = pending
		}

		var builder strings.Builder
		fmt.Fprintf(&builder, "Waiting for batch (%d/%d).", pending, expected)

		batchKey := ""
		if queueItem.BatchKey.Valid {
			batchKey = strings.TrimSpace(queueItem.BatchKey.String)
			if batchKey != "" {
				fmt.Fprintf(&builder, " Batch group key: %s.", batchKey)
			}
		}

		if meta.ConfiguredBatchMaxIntervalMs > 0 {
			interval := time.Duration(meta.ConfiguredBatchMaxIntervalMs) * time.Millisecond
			if meta.NextFlushAt != nil {
				fmt.Fprintf(&builder, " Flush at %s (interval %s).", meta.NextFlushAt.UTC().Format(time.RFC3339), interval)
			} else {
				fmt.Fprintf(&builder, " Flush after %s interval.", interval)
			}
		}

		if meta.ConfiguredBatchGroupMaxRuns > 0 {
			fmt.Fprintf(&builder, " Max concurrent batches per key: %d.", meta.ConfiguredBatchGroupMaxRuns)
		}

		if meta.Reason != "" {
			fmt.Fprintf(&builder, " Reason: %s.", meta.Reason)
		}

		eventPayload := map[string]interface{}{
			"status":            "waiting_for_batch",
			"batchGroupKey":     batchKey,
			"pending":           pending,
			"expectedSize":      expected,
			"batchGroupMaxRuns": meta.ConfiguredBatchGroupMaxRuns,
		}

		if meta.NextFlushAt != nil {
			eventPayload["nextFlushAt"] = meta.NextFlushAt.UTC().Format(time.RFC3339)
		}

		if meta.ConfiguredBatchMaxIntervalMs > 0 {
			eventPayload["batchMaxIntervalMs"] = meta.ConfiguredBatchMaxIntervalMs
		}

		if queueItem.StepID != uuid.Nil {
			eventPayload["stepId"] = queueItem.StepID.String()
		}

		if item.Batch != nil && item.Batch.ActionID != "" {
			eventPayload["actionId"] = item.Batch.ActionID
		} else if queueItem.ActionID != "" {
			eventPayload["actionId"] = queueItem.ActionID
		}

		payloads = append(payloads, tasktypes.CreateMonitoringEventPayload{
			TaskId:         queueItem.TaskID,
			RetryCount:     queueItem.RetryCount,
			EventType:      eventTypeWaitingForBatch,
			EventTimestamp: meta.TriggeredAt,
			EventMessage:   builder.String(),
			EventPayload:   buildBatchEventPayload(eventPayload),
		})
	}

	if len(payloads) == 0 {
		return nil
	}

	msg, err := msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDCreateMonitoringEvent,
		false,
		true,
		payloads...,
	)

	if err != nil {
		return fmt.Errorf("could not create waiting-for-batch monitoring events: %w", err)
	}

	if err := s.pubBuffer.Pub(ctx, msgqueue.OLAP_QUEUE, msg, false); err != nil {
		return fmt.Errorf("could not send waiting-for-batch monitoring events: %w", err)
	}

	return nil
}

func (s *Scheduler) internalRetry(ctx context.Context, tenantId uuid.UUID, assigned ...*repov1.AssignedItem) {
	for _, a := range assigned {
		msg, err := tasktypes.FailedTaskMessage(
			tenantId,
			a.QueueItem.TaskID,
			a.QueueItem.TaskInsertedAt,
			a.QueueItem.ExternalID,
			a.QueueItem.WorkflowRunID,
			a.QueueItem.RetryCount,
			false,
			"could not assign step run to worker",
			false,
		)

		if err != nil {
			s.l.Error().Ctx(ctx).Err(err).Msg("could not create failed task")
			continue
		}

		err = s.mq.SendMessage(
			ctx,
			msgqueue.TASK_PROCESSING_QUEUE,
			msg,
		)

		if err != nil {
			s.l.Error().Ctx(ctx).Err(err).Msg("could not send failed task")
			continue
		}
	}
}

func (s *Scheduler) notifyAfterConcurrency(ctx context.Context, tenantId uuid.UUID, res *v1.ConcurrencyResults) {
	uniqueQueueNames := make(map[string]struct{}, 0)

	for _, task := range res.Queued {
		uniqueQueueNames[task.Queue] = struct{}{}
	}

	uniqueNextStrategies := make(map[int64]struct{}, len(res.NextConcurrencyStrategies))

	for _, id := range res.NextConcurrencyStrategies {
		uniqueNextStrategies[id] = struct{}{}
	}

	strategies := make([]int64, 0, len(uniqueNextStrategies))

	for stratId := range uniqueNextStrategies {
		strategies = append(strategies, stratId)
	}

	s.pool.NotifyConcurrency(ctx, tenantId, strategies)

	queues := make([]string, 0, len(uniqueQueueNames))

	for queueName := range uniqueQueueNames {
		queues = append(queues, queueName)
	}

	s.pool.NotifyQueues(ctx, tenantId, queues)

	// The in-memory concurrency flush releases the task runtime in-transaction, so the
	// MsgIDTaskCancelled handler below (handleTaskCancelled -> CancelTasks) can no longer resolve the
	// worker for a task that was running when it got cancelled. Signal those workers' dispatchers
	// directly using the worker id captured during the flush, mirroring sendTaskCancellationsToDispatcher.
	s.signalWorkersToCancelInProgress(ctx, tenantId, res.Cancelled)

	// handle cancellations
	for _, cancelled := range res.Cancelled {
		eventType := sqlcv1.V1EventTypeOlapCANCELLED
		eventMessage := ""
		shouldNotify := true

		if cancelled.CancelledReason == "SCHEDULING_TIMED_OUT" {
			eventType = sqlcv1.V1EventTypeOlapSCHEDULINGTIMEDOUT
			shouldNotify = false
		} else {
			eventMessage = "Cancelled due to concurrency strategy"
		}

		msg, err := tasktypes.CancelledTaskMessage(
			tenantId,
			cancelled.TaskIdInsertedAtRetryCount.Id,
			cancelled.TaskIdInsertedAtRetryCount.InsertedAt,
			cancelled.TaskExternalId,
			cancelled.WorkflowRunId,
			cancelled.TaskIdInsertedAtRetryCount.RetryCount,
			eventType,
			eventMessage,
			shouldNotify,
		)

		if err != nil {
			s.l.Error().Ctx(ctx).Err(err).Msg("could not create cancelled task")
			continue
		}

		err = s.mq.SendMessage(
			ctx,
			msgqueue.TASK_PROCESSING_QUEUE,
			msg,
		)

		if err != nil {
			s.l.Error().Ctx(ctx).Err(err).Msg("could not send cancelled task")
			continue
		}
	}
}

// signalWorkersToCancelInProgress tells the relevant dispatchers to cancel tasks that were running on
// a worker when a concurrency strategy cancelled them. The in-memory concurrency flush releases the
// task runtime in its transaction, so the MsgIDTaskCancelled handler (handleTaskCancelled) can't map
// these tasks back to a worker; we do it here from the worker id captured during the flush. Tasks that
// were only queued (no runtime) carry a nil worker id and are skipped - their cancellation still flows
// through the MsgIDTaskCancelled handler. This mirrors sendTaskCancellationsToDispatcher in the tasks
// controller.
func (s *Scheduler) signalWorkersToCancelInProgress(ctx context.Context, tenantId uuid.UUID, cancelled []repov1.TaskWithCancelledReason) {
	signals := make([]tasktypes.SignalTaskCancelledPayload, 0, len(cancelled))
	workerIds := make([]uuid.UUID, 0, len(cancelled))

	for _, c := range cancelled {
		if c.WorkerId == uuid.Nil {
			continue
		}

		signals = append(signals, tasktypes.SignalTaskCancelledPayload{
			TaskId:     c.Id,
			InsertedAt: c.InsertedAt,
			RetryCount: c.RetryCount,
			WorkerId:   c.WorkerId,
		})
		workerIds = append(workerIds, c.WorkerId)
	}

	if len(signals) == 0 {
		return
	}

	workerIdToDispatcherId, _, err := s.repov1.Workers().GetDispatcherIdsForWorkers(ctx, tenantId, workerIds)

	if err != nil {
		s.l.Error().Ctx(ctx).Err(err).Msg("could not list dispatcher ids for cancelled in-progress tasks")
		return
	}

	dispatcherIdsToPayloads := make(map[uuid.UUID][]tasktypes.SignalTaskCancelledPayload)

	for _, sig := range signals {
		dispatcherId, ok := workerIdToDispatcherId[sig.WorkerId]

		if !ok {
			continue
		}

		dispatcherIdsToPayloads[dispatcherId] = append(dispatcherIdsToPayloads[dispatcherId], sig)
	}

	for dispatcherId, payloads := range dispatcherIdsToPayloads {
		msg, err := msgqueue.NewTenantMessage(
			tenantId,
			msgqueue.MsgIDTaskCancelled,
			false,
			true,
			payloads...,
		)

		if err != nil {
			s.l.Error().Ctx(ctx).Err(err).Msg("could not create task cancellation signal for dispatcher")
			continue
		}

		if err := s.mq.SendMessage(ctx, msgqueue.QueueTypeFromDispatcherID(dispatcherId), msg); err != nil {
			s.l.Error().Ctx(ctx).Err(err).Msg("could not send task cancellation signal to dispatcher")
		}
	}
}

func (s *Scheduler) handleBatchAssignments(ctx context.Context, tenantId uuid.UUID, assignments []*repov1.AssignedItem, workerIdToDispatcherId map[uuid.UUID]uuid.UUID) error {
	if len(assignments) == 0 {
		return nil
	}

	type batchGroupKey struct {
		WorkerID uuid.UUID
		StepID   string
		ActionID string
		BatchKey string
	}

	groups := make(map[batchGroupKey][]*repov1.AssignedItem)

	for _, assignment := range assignments {
		if assignment == nil || assignment.QueueItem == nil {
			continue
		}

		workerID := assignment.WorkerId
		stepID := assignment.QueueItem.StepID.String()
		actionID := assignment.QueueItem.ActionID
		batchKey := ""
		if assignment.QueueItem.BatchKey.Valid {
			batchKey = strings.TrimSpace(assignment.QueueItem.BatchKey.String)
		}

		if meta := assignment.Batch; meta != nil {
			if meta.StepID != "" {
				stepID = meta.StepID
			}
			if meta.ActionID != "" {
				actionID = meta.ActionID
			}
			if strings.TrimSpace(meta.BatchGroupKey) != "" {
				batchKey = strings.TrimSpace(meta.BatchGroupKey)
			}
		}

		key := batchGroupKey{
			WorkerID: workerID,
			StepID:   stepID,
			ActionID: actionID,
			BatchKey: batchKey,
		}

		groups[key] = append(groups[key], assignment)
	}

	var result error

	for key, group := range groups {
		if len(group) == 0 {
			continue
		}

		s.l.Debug().
			Str("tenant_id", tenantId.String()).
			Str("worker_id", key.WorkerID.String()).
			Str("step_id", key.StepID).
			Str("action_id", key.ActionID).
			Str("batch_key", key.BatchKey).
			Int("original_group_size", len(group)).
			Msg("prepared batch dispatcher group")

		if len(group) == 0 {
			continue
		}

		meta := group[0].Batch
		if meta == nil {
			meta = &repov1.BatchAssignmentMetadata{}
		}

		batchID := strings.TrimSpace(meta.BatchID)
		if batchID == "" {
			batchID = uuid.NewString()
		}

		shouldRelease := meta.ConfiguredBatchGroupMaxRuns > 0 && strings.TrimSpace(key.BatchKey) != "" && batchID != ""
		releaseOnError := func() {
			if !shouldRelease {
				return
			}

			if err := s.repov1.Tasks().DeleteTaskBatchRun(ctx, tenantId.String(), batchID); err != nil {
				s.l.Error().
					Err(err).
					Str("tenant_id", tenantId.String()).
					Str("batch_id", batchID).
					Msg("failed to release batch reservation after error")
			}
		}

		dispatcherID, ok := workerIdToDispatcherId[key.WorkerID]
		if !ok {
			s.internalRetry(ctx, tenantId, group...)
			releaseOnError()
			result = multierror.Append(result, fmt.Errorf("could not assign batch to worker %s: dispatcher id missing", key.WorkerID))
			continue
		}

		triggeredAt := meta.TriggeredAt
		if triggeredAt.IsZero() {
			triggeredAt = time.Now().UTC()
		}

		batchSize := len(group)
		configuredSize := int(meta.ConfiguredBatchMaxSize)
		if configuredSize <= 0 {
			configuredSize = batchSize
		}

		flushReason := describeBatchFlushReason(meta.Reason, configuredSize, time.Duration(meta.ConfiguredBatchMaxIntervalMs)*time.Millisecond)

		// batch_id/batch_size/batch_index/worker_id/batch_key were already set on these
		// v1_task_runtime rows by ReserveAndCommitBatchRun, atomically with the reservation itself
		// - no separate persistence step is needed here.
		batchItems := make([]tasktypes.BatchTaskItem, 0, len(group))

		for _, item := range group {
			if item == nil || item.QueueItem == nil || !item.QueueItem.TaskInsertedAt.Valid {
				continue
			}

			batchItems = append(batchItems, tasktypes.BatchTaskItem{
				TaskID:        item.QueueItem.TaskID,
				ExternalID:    item.QueueItem.ExternalID,
				WorkflowRunID: item.QueueItem.WorkflowRunID,
				InsertedAt:    item.QueueItem.TaskInsertedAt.Time,
			})
		}

		if len(batchItems) == 0 {
			releaseOnError()
			continue
		}

		startPayload := tasktypes.StartBatchTaskPayload{
			TenantId:      tenantId,
			WorkerId:      key.WorkerID,
			ActionId:      key.ActionID,
			BatchId:       batchID,
			ExpectedSize:  batchSize,
			BatchKey:      key.BatchKey,
			TriggerReason: flushReason,
			TriggerTime:   triggeredAt,
			Items:         batchItems,
		}

		if meta.ConfiguredBatchGroupMaxRuns > 0 {
			maxRuns := int(meta.ConfiguredBatchGroupMaxRuns)
			startPayload.MaxRuns = &maxRuns
		}

		startMsg, err := tasktypes.StartBatchMessage(tenantId, startPayload)
		if err != nil {
			s.internalRetry(ctx, tenantId, group...)
			releaseOnError()
			result = multierror.Append(result, fmt.Errorf("could not create batch start message: %w", err))
			continue
		}

		if err := s.mq.SendMessage(ctx, msgqueue.QueueTypeFromDispatcherID(dispatcherID), startMsg); err != nil {
			s.internalRetry(ctx, tenantId, group...)
			releaseOnError()
			result = multierror.Append(result, fmt.Errorf("could not send batch start message: %w", err))
			continue
		}

		batchMessage := fmt.Sprintf(
			"Batch %s flushed (%d/%d tasks) at %s because %s.",
			batchID,
			batchSize,
			configuredSize,
			triggeredAt.UTC().Format(time.RFC3339),
			flushReason,
		)

		if key.BatchKey != "" {
			batchMessage += fmt.Sprintf(" Batch group key: %s.", key.BatchKey)
		}

		if meta.ConfiguredBatchGroupMaxRuns > 0 {
			batchMessage += fmt.Sprintf(" Max concurrent batches per key: %d.", meta.ConfiguredBatchGroupMaxRuns)
		}

		if meta.ConfiguredBatchMaxIntervalMs > 0 {
			interval := time.Duration(meta.ConfiguredBatchMaxIntervalMs) * time.Millisecond
			batchMessage += fmt.Sprintf(" Configured batch max interval: %s.", interval)
		}

		batchPayloadFields := map[string]interface{}{
			"status":            "flushed",
			"batchId":           batchID,
			"batchGroupKey":     key.BatchKey,
			"batchMaxSize":      batchSize,
			"configuredSize":    configuredSize,
			"triggerReason":     flushReason,
			"triggeredAt":       triggeredAt.UTC().Format(time.RFC3339),
			"batchGroupMaxRuns": meta.ConfiguredBatchGroupMaxRuns,
		}

		if key.StepID != "" {
			batchPayloadFields["stepId"] = key.StepID
		}

		if meta.ActionID != "" {
			batchPayloadFields["actionId"] = meta.ActionID
		} else if key.ActionID != "" {
			batchPayloadFields["actionId"] = key.ActionID
		}

		if meta.ConfiguredBatchMaxIntervalMs > 0 {
			batchPayloadFields["batchMaxIntervalMs"] = meta.ConfiguredBatchMaxIntervalMs
		}

		batchPayload := buildBatchEventPayload(batchPayloadFields)

		monitoringPayloads := make([]tasktypes.CreateMonitoringEventPayload, 0, len(group))
		for _, item := range group {
			if item == nil || item.QueueItem == nil {
				continue
			}

			monitoringPayloads = append(monitoringPayloads, tasktypes.CreateMonitoringEventPayload{
				TaskId:         item.QueueItem.TaskID,
				RetryCount:     item.QueueItem.RetryCount,
				WorkerId:       &key.WorkerID,
				EventType:      eventTypeBatchFlushed,
				EventTimestamp: triggeredAt,
				EventMessage:   batchMessage,
				EventPayload:   batchPayload,
			})
		}

		if len(monitoringPayloads) > 0 {
			flushMsg, err := msgqueue.NewTenantMessage(
				tenantId,
				"create-monitoring-event",
				false,
				true,
				monitoringPayloads...,
			)

			if err != nil {
				result = multierror.Append(result, fmt.Errorf("could not create batch flushed monitoring events: %w", err))
			} else if err := s.pubBuffer.Pub(ctx, msgqueue.OLAP_QUEUE, flushMsg, false); err != nil {
				result = multierror.Append(result, fmt.Errorf("could not send batch flushed monitoring events: %w", err))
			}
		}
	}

	return result
}

func describeBatchFlushReason(reason repov1.BatchFlushReason, batchSize int, interval time.Duration) string {
	switch reason {
	case repov1.FlushReasonBatchSizeReached:
		if batchSize > 0 {
			return fmt.Sprintf("batch size threshold %d reached", batchSize)
		}
		return "batch size threshold reached"
	case repov1.FlushReasonWorkerChanged:
		return "assigned worker changed"
	case repov1.FlushReasonDispatcherChanged:
		return "dispatcher changed"
	case repov1.FlushReasonIntervalElapsed:
		if interval > 0 {
			return fmt.Sprintf("batch max interval %s elapsed", interval)
		}
		return "batch max interval elapsed"
	case repov1.FlushReasonBufferDrained:
		return "buffer drained during shutdown"
	case repov1.FlushReasonBufferMemorySizeReached:
		return "batch max memory size reached"
	default:
		return string(reason)
	}
}

func taskBulkAssignedTask(tenantId uuid.UUID, workerIdsToTaskIds map[uuid.UUID][]int64) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDTaskAssignedBulk,
		false,
		true,
		tasktypes.TaskAssignedBulkTaskPayload{
			WorkerIdToTaskIds: workerIdsToTaskIds,
		},
	)
}

func buildBatchEventPayload(fields map[string]interface{}) string {
	data := make(map[string]interface{}, len(fields))

	for key, value := range fields {
		switch v := value.(type) {
		case string:
			if v != "" {
				data[key] = v
			}
		case int, int32, int64:
			if fmt.Sprintf("%v", v) != "0" {
				data[key] = v
			}
		case float32, float64:
			if fmt.Sprintf("%v", v) != "0" {
				data[key] = v
			}
		default:
			if value != nil {
				data[key] = value
			}
		}
	}

	if len(data) == 0 {
		return ""
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		return ""
	}

	return string(bytes)
}

func (s *Scheduler) handleDeadLetteredMessages(msg *msgqueue.Message) (err error) {
	defer func() {
		if r := recover(); r != nil {
			recoverErr := recoveryutils.RecoverWithAlert(s.l, s.a, r)

			if recoverErr != nil {
				err = recoverErr
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	switch msg.ID {
	case msgqueue.MsgIDTaskAssignedBulk:
		err = s.handleDeadLetteredTaskBulkAssigned(ctx, msg)
	case msgqueue.MsgIDTaskCancelled:
		err = s.handleDeadLetteredTaskCancelled(ctx, msg)
	case msgqueue.MsgIDDurableCallbackCompleted:
		err = s.handleDeadLetteredDurableCallbackCompleted(ctx, msg)
	default:
		err = fmt.Errorf("unknown task: %s", msg.ID)
	}

	return err
}

func (s *Scheduler) handleDeadLetteredTaskBulkAssigned(ctx context.Context, msg *msgqueue.Message) error {
	msgs := msgqueue.JSONConvert[tasktypes.TaskAssignedBulkTaskPayload](msg.Payloads)

	taskIds := make([]int64, 0)

	for _, innerMsg := range msgs {
		for workerID, ids := range innerMsg.WorkerIdToTaskIds {
			s.l.Error().Ctx(ctx).Msgf("handling dead-lettered task assignments for tenant %s, worker %s, tasks: %v. This indicates an abrupt shutdown of a dispatcher and should be investigated.", msg.TenantID, workerID, ids)
			taskIds = append(taskIds, ids...)
		}
	}

	toFail, err := s.repov1.Tasks().ListTasks(ctx, msg.TenantID, taskIds)

	if err != nil {
		return fmt.Errorf("could not list tasks for dead lettered bulk assigned message: %w", err)
	}

	for _, task := range toFail {
		tenantId := msg.TenantID

		msg, err := tasktypes.FailedTaskMessage(
			tenantId,
			task.ID,
			task.InsertedAt,
			task.ExternalID,
			task.WorkflowRunID,
			task.RetryCount,
			false,
			"Could not send task to worker",
			false,
		)

		if err != nil {
			return fmt.Errorf("could not create failed task message: %w", err)
		}

		err = s.mq.SendMessage(ctx, msgqueue.TASK_PROCESSING_QUEUE, msg)

		if err != nil {
			// NOTE: failure to send on the MQ is likely not transient; ideally we could only retry individual
			// tasks but since this message has the tasks in a batch, we retry all of them instead. we're banking
			// on the downstream `task-failed` processing to be idempotent.
			return fmt.Errorf("could not send failed task message: %w", err)
		}
	}

	return nil
}

func (s *Scheduler) handleDeadLetteredDurableCallbackCompleted(ctx context.Context, msg *msgqueue.Message) error {
	payloads := msgqueue.JSONConvert[tasktypes.DurableCallbackCompletedPayload](msg.Payloads)

	externalIds := make([]uuid.UUID, 0, len(payloads))
	for _, p := range payloads {
		externalIds = append(externalIds, p.TaskExternalId)
	}

	flatTasks, err := s.repov1.Tasks().FlattenExternalIds(ctx, msg.TenantID, externalIds)
	if err != nil {
		return fmt.Errorf("could not resolve tasks for undelivered durable callbacks: %w", err)
	}

	flatByExternalId := make(map[uuid.UUID]*sqlcv1.FlattenExternalIdsRow, len(flatTasks))
	for _, t := range flatTasks {
		flatByExternalId[t.ExternalID] = t
	}

	callbacks := make([]repov1.SatisfiedEntry, 0, len(payloads))

	for _, p := range payloads {
		t, ok := flatByExternalId[p.TaskExternalId]
		if !ok {
			s.l.Warn().Ctx(ctx).Msgf("no task found for undelivered durable callback %s, skipping", p.TaskExternalId)
			continue
		}

		callbacks = append(callbacks, repov1.SatisfiedEntry{
			DurableTaskId:         t.ID,
			DurableTaskInsertedAt: t.InsertedAt,
			DurableTaskExternalId: p.TaskExternalId,
			InvocationCount:       p.InvocationCount,
			BranchId:              p.BranchId,
			NodeId:                p.NodeId,
			Data:                  p.Payload,
			ChildTaskIsFailure:    p.ChildTaskIsFailure,
			ChildTaskErrorMessage: p.ChildTaskErrorMessage,
		})
	}

	return durable.DispatchCallbacks(ctx, s.l, s.mq, s.repov1, msg.TenantID, callbacks)
}

func (s *Scheduler) handleDeadLetteredTaskCancelled(ctx context.Context, msg *msgqueue.Message) error {
	payloads := msgqueue.JSONConvert[tasktypes.SignalTaskCancelledPayload](msg.Payloads)

	// try to resend the cancellation signal to the impacted worker.
	workerIds := make([]uuid.UUID, 0)

	for _, p := range payloads {
		s.l.Error().Ctx(ctx).Msgf("handling dead-lettered task cancellations for tenant %s, task %d. This indicates an abrupt shutdown of a dispatcher and should be investigated.", msg.TenantID, p.TaskId)
		workerIds = append(workerIds, p.WorkerId)
	}

	// since the dispatcher IDs may have changed since the previous send, we need to query them again
	workerIdToDispatcherId, workersWithoutDispatchers, err := s.repov1.Workers().GetDispatcherIdsForWorkers(ctx, msg.TenantID, workerIds)

	if err != nil {
		return fmt.Errorf("could not list dispatcher ids for workers: %w", err)
	}

	dispatcherIdsToPayloads := make(map[uuid.UUID][]tasktypes.SignalTaskCancelledPayload)

	for _, p := range payloads {
		// if we no longer have the worker attached to a dispatcher, discard the message
		dispatcherId, ok := workerIdToDispatcherId[p.WorkerId]
		_, hasNoDispatcher := workersWithoutDispatchers[p.WorkerId]

		if hasNoDispatcher || !ok {
			continue
		}

		dispatcherIdsToPayloads[dispatcherId] = append(dispatcherIdsToPayloads[dispatcherId], *p)
	}

	for dispatcherId, payloads := range dispatcherIdsToPayloads {
		msg, err := msgqueue.NewTenantMessage(
			msg.TenantID,
			msgqueue.MsgIDTaskCancelled,
			false,
			true,
			payloads...,
		)

		if err != nil {
			return fmt.Errorf("could not create message for task cancellation: %w", err)
		}

		err = s.mq.SendMessage(
			ctx,
			msgqueue.QueueTypeFromDispatcherID(dispatcherId),
			msg,
		)

		if err != nil {
			return fmt.Errorf("could not send message for task cancellation: %w", err)
		}
	}

	return nil
}
