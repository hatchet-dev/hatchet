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
	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	"github.com/hatchet-dev/hatchet/internal/services/partition"
	"github.com/hatchet-dev/hatchet/internal/services/shared/recoveryutils"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/pkg/config/shared"
	hatcheterrors "github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	repov1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
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
	repo        repository.EngineRepository
	repov1      repov1.Repository
	dv          datautils.DataDecoderValidator
	alerter     hatcheterrors.Alerter
	p           *partition.Partition
	queueLogger *zerolog.Logger
	pool        *v1.SchedulingPool
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

func WithRepository(r repository.EngineRepository) SchedulerOpt {
	return func(opts *SchedulerOpts) {
		opts.repo = r
	}
}

func WithV2Repository(r repov1.Repository) SchedulerOpt {
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

type Scheduler struct {
	mq        msgqueue.MessageQueue
	pubBuffer *msgqueue.MQPubBuffer
	l         *zerolog.Logger
	repo      repository.EngineRepository
	repov1    repov1.Repository
	dv        datautils.DataDecoderValidator
	s         gocron.Scheduler
	a         *hatcheterrors.Wrapped
	p         *partition.Partition

	// a custom queue logger
	ql *zerolog.Logger

	pool *v1.SchedulingPool

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

	if opts.repo == nil {
		return nil, fmt.Errorf("repository is required. use WithRepository")
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

	q := &Scheduler{
		mq:                     opts.mq,
		pubBuffer:              pubBuffer,
		l:                      opts.l,
		repo:                   opts.repo,
		repov1:                 opts.repov1,
		dv:                     opts.dv,
		s:                      s,
		a:                      a,
		p:                      opts.p,
		ql:                     opts.queueLogger,
		pool:                   opts.pool,
		tasksWithNoWorkerCache: tasksWithNoWorkerCache,
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
			s.l.Error().Err(err).Msg("could not handle job task")
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
				s.l.Debug().Msgf("partition: received queue results")

				if res == nil {
					continue
				}

				go func(results *v1.QueueResults) {
					err := s.scheduleStepRuns(ctx, sqlchelpers.UUIDToStr(results.TenantId), results)

					if err != nil {
						s.l.Error().Err(err).Msg("could not schedule step runs")
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
				s.l.Debug().Msgf("partition: received concurrency results")

				if res == nil {
					continue
				}

				go s.notifyAfterConcurrency(ctx, sqlchelpers.UUIDToStr(res.TenantId), res)
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

	if task.ID == "check-tenant-queue" {
		return s.handleCheckQueue(ctx, task)
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

func (s *Scheduler) runSetTenants(ctx context.Context) func() {
	return func() {
		s.l.Debug().Msgf("partition: checking step run requeue")

		// list all tenants
		tenants, err := s.repo.Tenant().ListTenantsBySchedulerPartition(ctx, s.p.GetSchedulerPartitionId(), dbsqlc.TenantMajorEngineVersionV1)

		if err != nil {
			s.l.Err(err).Msg("could not list tenants")
			return
		}

		s.pool.SetTenants(tenants)
	}
}

func (s *Scheduler) scheduleStepRuns(ctx context.Context, tenantId string, res *v1.QueueResults) error {
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
		dispatcherIdToWorkerIdsToStepRuns := make(map[string]map[string][]int64)

		workerIds := make([]string, 0)

		for _, assigned := range res.Assigned {
			workerIds = append(workerIds, sqlchelpers.UUIDToStr(assigned.WorkerId))
		}

		var dispatcherIdWorkerIds map[string][]string

		dispatcherIdWorkerIds, err := s.repo.Worker().GetDispatcherIdsForWorkers(ctx, tenantId, workerIds)

		if err != nil {
			s.internalRetry(ctx, tenantId, res.Assigned...)

			return fmt.Errorf("could not list dispatcher ids for workers: %w. attempting internal retry", err)
		}

		workerIdToDispatcherId := make(map[string]string)

		for dispatcherId, workerIds := range dispatcherIdWorkerIds {
			for _, workerId := range workerIds {
				workerIdToDispatcherId[workerId] = dispatcherId
			}
		}

		assignedMsgs := make([]*msgqueue.Message, 0)
		batchAssignments := make([]*repov1.AssignedItem, 0)

		for _, bulkAssigned := range res.Assigned {
			if bulkAssigned != nil && bulkAssigned.Batch != nil && bulkAssigned.QueueItem != nil && bulkAssigned.QueueItem.BatchKey.Valid && strings.TrimSpace(bulkAssigned.QueueItem.BatchKey.String) != "" {
				batchAssignments = append(batchAssignments, bulkAssigned)
				continue
			}

			dispatcherId, ok := workerIdToDispatcherId[sqlchelpers.UUIDToStr(bulkAssigned.WorkerId)]

			if !ok {
				s.l.Error().Msg("could not assign step run to worker: no dispatcher id. attempting internal retry.")

				s.internalRetry(ctx, tenantId, bulkAssigned)

				continue
			}

			if _, ok := dispatcherIdToWorkerIdsToStepRuns[dispatcherId]; !ok {
				dispatcherIdToWorkerIdsToStepRuns[dispatcherId] = make(map[string][]int64)
			}

			workerId := sqlchelpers.UUIDToStr(bulkAssigned.WorkerId)

			if _, ok := dispatcherIdToWorkerIdsToStepRuns[dispatcherId][workerId]; !ok {
				dispatcherIdToWorkerIdsToStepRuns[dispatcherId][workerId] = make([]int64, 0)
			}

			dispatcherIdToWorkerIdsToStepRuns[dispatcherId][workerId] = append(dispatcherIdToWorkerIdsToStepRuns[dispatcherId][workerId], bulkAssigned.QueueItem.TaskID)

			taskId := bulkAssigned.QueueItem.TaskID

			assignedMsg, err := tasktypes.MonitoringEventMessageFromInternal(
				tenantId,
				tasktypes.CreateMonitoringEventPayload{
					TaskId:         taskId,
					RetryCount:     bulkAssigned.QueueItem.RetryCount,
					WorkerId:       &workerId,
					EventType:      sqlcv1.V1EventTypeOlapASSIGNED,
					EventTimestamp: time.Now(),
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
				sqlchelpers.UUIDToStr(schedulingTimedOut.ExternalID),
				sqlchelpers.UUIDToStr(schedulingTimedOut.WorkflowRunID),
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
			taskExternalId := sqlchelpers.UUIDToStr(unassigned.ExternalID)

			// if we have seen this task recently, don't send it again
			if _, ok := s.tasksWithNoWorkerCache.Get(taskExternalId); ok {
				s.l.Debug().Msgf("skipping unassigned task %s as it was recently unassigned", taskExternalId)
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

func (s *Scheduler) emitBatchWaitingEvents(ctx context.Context, tenantId string, buffered []*repov1.AssignedItem) error {
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

		expected := int(meta.ConfiguredBatchSize)
		if expected <= 0 {
			expected = pending
		}

		var builder strings.Builder
		fmt.Fprintf(&builder, "Waiting for batch (%d/%d).", pending, expected)

		batchKey := ""
		if queueItem.BatchKey.Valid {
			batchKey = strings.TrimSpace(queueItem.BatchKey.String)
			if batchKey != "" {
				fmt.Fprintf(&builder, " Batch key: %s.", batchKey)
			}
		}

		if meta.ConfiguredFlushIntervalMs > 0 {
			interval := time.Duration(meta.ConfiguredFlushIntervalMs) * time.Millisecond
			if meta.NextFlushAt != nil {
				fmt.Fprintf(&builder, " Flush at %s (interval %s).", meta.NextFlushAt.UTC().Format(time.RFC3339), interval)
			} else {
				fmt.Fprintf(&builder, " Flush after %s interval.", interval)
			}
		}

		if meta.MaxRuns > 0 {
			fmt.Fprintf(&builder, " Max concurrent batches per key: %d.", meta.MaxRuns)
		}

		if meta.Reason != "" {
			fmt.Fprintf(&builder, " Reason: %s.", meta.Reason)
		}

		eventPayload := map[string]interface{}{
			"status":       "waiting_for_batch",
			"batchKey":     batchKey,
			"pending":      pending,
			"expectedSize": expected,
			"maxRuns":      meta.MaxRuns,
		}

		if meta.NextFlushAt != nil {
			eventPayload["nextFlushAt"] = meta.NextFlushAt.UTC().Format(time.RFC3339)
		}

		if meta.ConfiguredFlushIntervalMs > 0 {
			eventPayload["flushIntervalMs"] = meta.ConfiguredFlushIntervalMs
		}

		if queueItem.StepID.Valid {
			eventPayload["stepId"] = sqlchelpers.UUIDToStr(queueItem.StepID)
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
		"create-monitoring-event",
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

func (s *Scheduler) internalRetry(ctx context.Context, tenantId string, assigned ...*repov1.AssignedItem) {
	for _, a := range assigned {
		msg, err := tasktypes.FailedTaskMessage(
			tenantId,
			a.QueueItem.TaskID,
			a.QueueItem.TaskInsertedAt,
			sqlchelpers.UUIDToStr(a.QueueItem.ExternalID),
			sqlchelpers.UUIDToStr(a.QueueItem.WorkflowRunID),
			a.QueueItem.RetryCount,
			false,
			"could not assign step run to worker",
			false,
		)

		if err != nil {
			s.l.Error().Err(err).Msg("could not create failed task")
			continue
		}

		err = s.mq.SendMessage(
			ctx,
			msgqueue.TASK_PROCESSING_QUEUE,
			msg,
		)

		if err != nil {
			s.l.Error().Err(err).Msg("could not send failed task")
			continue
		}
	}
}

func (s *Scheduler) notifyAfterConcurrency(ctx context.Context, tenantId string, res *v1.ConcurrencyResults) {
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
			s.l.Error().Err(err).Msg("could not create cancelled task")
			continue
		}

		err = s.mq.SendMessage(
			ctx,
			msgqueue.TASK_PROCESSING_QUEUE,
			msg,
		)

		if err != nil {
			s.l.Error().Err(err).Msg("could not send cancelled task")
			continue
		}
	}
}

func (s *Scheduler) handleBatchAssignments(ctx context.Context, tenantId string, assignments []*repov1.AssignedItem, workerIdToDispatcherId map[string]string) error {
	if len(assignments) == 0 {
		return nil
	}

	type batchGroupKey struct {
		WorkerID string
		StepID   string
		ActionID string
		BatchKey string
	}

	groups := make(map[batchGroupKey][]*repov1.AssignedItem)

	for _, assignment := range assignments {
		if assignment == nil || assignment.QueueItem == nil {
			continue
		}

		workerID := sqlchelpers.UUIDToStr(assignment.WorkerId)
		stepID := sqlchelpers.UUIDToStr(assignment.QueueItem.StepID)
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
			if strings.TrimSpace(meta.BatchKey) != "" {
				batchKey = strings.TrimSpace(meta.BatchKey)
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

		// FIXME: It is not clear why we're ending up in this state, but we should investigate why and fix it.
		// Deduplicate tasks within a batch group by (task_id, task_inserted_at).
		// In some edge cases we can end up with multiple AssignedItems that
		// reference the same underlying task, which would cause duplicate
		// batch indexes on the worker side.
		type taskKey struct {
			id         int64
			insertedAt time.Time
		}

		seen := make(map[taskKey]bool, len(group))
		dedupedGroup := make([]*repov1.AssignedItem, 0, len(group))

		for _, item := range group {
			if item == nil || item.QueueItem == nil || !item.QueueItem.TaskInsertedAt.Valid {
				continue
			}

			k := taskKey{
				id:         item.QueueItem.TaskID,
				insertedAt: item.QueueItem.TaskInsertedAt.Time,
			}

			if seen[k] {
				s.l.Warn().
					Int64("task_id", item.QueueItem.TaskID).
					Str("step_id", sqlchelpers.UUIDToStr(item.QueueItem.StepID)).
					Str("action_id", item.QueueItem.ActionID).
					Str("batch_key", key.BatchKey).
					Msg("skipping duplicate task in batch dispatcher group")
				continue
			}

			seen[k] = true
			dedupedGroup = append(dedupedGroup, item)
		}

		s.l.Debug().
			Str("tenant_id", tenantId).
			Str("worker_id", key.WorkerID).
			Str("step_id", key.StepID).
			Str("action_id", key.ActionID).
			Str("batch_key", key.BatchKey).
			Int("original_group_size", len(group)).
			Int("deduped_group_size", len(dedupedGroup)).
			Msg("prepared batch dispatcher group")

		group = dedupedGroup

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

		shouldRelease := meta.MaxRuns > 0 && strings.TrimSpace(key.BatchKey) != "" && batchID != ""
		releaseOnError := func() {
			if !shouldRelease {
				return
			}

			if err := s.repov1.Tasks().DeleteTaskBatchRun(ctx, tenantId, batchID); err != nil {
				s.l.Error().
					Err(err).
					Str("tenant_id", tenantId).
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
		configuredSize := int(meta.ConfiguredBatchSize)
		if configuredSize <= 0 {
			configuredSize = batchSize
		}

		flushReason := describeBatchFlushReason(meta.Reason, configuredSize, time.Duration(meta.ConfiguredFlushIntervalMs)*time.Millisecond)

		assignmentsPayload := make([]repov1.TaskBatchAssignment, 0, len(group))
		taskIds := make([]int64, 0, len(group))

		for idx, item := range group {
			if item == nil || item.QueueItem == nil || !item.QueueItem.TaskInsertedAt.Valid {
				continue
			}

			assignmentsPayload = append(assignmentsPayload, repov1.TaskBatchAssignment{
				TaskID:         item.QueueItem.TaskID,
				TaskInsertedAt: item.QueueItem.TaskInsertedAt.Time,
				BatchIndex:     idx,
			})

			taskIds = append(taskIds, item.QueueItem.TaskID)
		}

		if len(assignmentsPayload) == 0 {
			releaseOnError()
			continue
		}
		if err := s.repov1.Tasks().UpdateTaskBatchMetadata(ctx, tenantId, batchID, key.WorkerID, key.BatchKey, batchSize, assignmentsPayload); err != nil {
			s.internalRetry(ctx, tenantId, group...)
			releaseOnError()
			result = multierror.Append(result, fmt.Errorf("could not persist batch metadata: %w", err))
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
		}

		if meta.MaxRuns > 0 {
			maxRuns := int(meta.MaxRuns)
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

		workerBatches := map[string][]tasktypes.TaskAssignedBatch{
			key.WorkerID: {
				{
					BatchID:    batchID,
					BatchSize:  batchSize,
					TaskIds:    taskIds,
					StartBatch: &startPayload,
				},
			},
		}

		assignedMsg, err := taskAssignedBatchMessage(tenantId, workerBatches)
		if err != nil {
			s.internalRetry(ctx, tenantId, group...)
			releaseOnError()
			result = multierror.Append(result, fmt.Errorf("could not create bulk assigned batch message: %w", err))
			continue
		}

		if err := s.mq.SendMessage(ctx, msgqueue.QueueTypeFromDispatcherID(dispatcherID), assignedMsg); err != nil {
			s.internalRetry(ctx, tenantId, group...)
			releaseOnError()
			result = multierror.Append(result, fmt.Errorf("could not send bulk assigned batch message: %w", err))
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
			batchMessage += fmt.Sprintf(" Batch key: %s.", key.BatchKey)
		}

		if meta.MaxRuns > 0 {
			batchMessage += fmt.Sprintf(" Max concurrent batches per key: %d.", meta.MaxRuns)
		}

		if meta.ConfiguredFlushIntervalMs > 0 {
			interval := time.Duration(meta.ConfiguredFlushIntervalMs) * time.Millisecond
			batchMessage += fmt.Sprintf(" Configured flush interval: %s.", interval)
		}

		batchPayloadFields := map[string]interface{}{
			"status":         "flushed",
			"batchId":        batchID,
			"batchKey":       key.BatchKey,
			"batchSize":      batchSize,
			"configuredSize": configuredSize,
			"triggerReason":  flushReason,
			"triggeredAt":    triggeredAt.UTC().Format(time.RFC3339),
			"maxRuns":        meta.MaxRuns,
		}

		if key.StepID != "" {
			batchPayloadFields["stepId"] = key.StepID
		}

		if meta.ActionID != "" {
			batchPayloadFields["actionId"] = meta.ActionID
		} else if key.ActionID != "" {
			batchPayloadFields["actionId"] = key.ActionID
		}

		if meta.ConfiguredFlushIntervalMs > 0 {
			batchPayloadFields["flushIntervalMs"] = meta.ConfiguredFlushIntervalMs
		}

		batchPayload := buildBatchEventPayload(batchPayloadFields)

		workerPtr := &key.WorkerID
		if key.WorkerID == "" {
			workerPtr = nil
		}

		monitoringPayloads := make([]tasktypes.CreateMonitoringEventPayload, 0, len(group))
		for _, item := range group {
			if item == nil || item.QueueItem == nil {
				continue
			}

			monitoringPayloads = append(monitoringPayloads, tasktypes.CreateMonitoringEventPayload{
				TaskId:         item.QueueItem.TaskID,
				RetryCount:     item.QueueItem.RetryCount,
				WorkerId:       workerPtr,
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

func describeBatchFlushReason(reason string, batchSize int, interval time.Duration) string {
	switch reason {
	case "batch_size_reached":
		if batchSize > 0 {
			return fmt.Sprintf("batch size threshold %d reached", batchSize)
		}
		return "batch size threshold reached"
	case "worker_changed":
		return "assigned worker changed"
	case "dispatcher_changed":
		return "dispatcher changed"
	case "interval_elapsed":
		if interval > 0 {
			return fmt.Sprintf("flush interval %s elapsed", interval)
		}
		return "flush interval elapsed"
	case "buffer_drained":
		return "buffer drained during shutdown"
	default:
		return reason
	}
}

func taskAssignedBatchMessage(tenantId string, workerBatches map[string][]tasktypes.TaskAssignedBatch) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		"task-assigned-bulk",
		false,
		true,
		tasktypes.TaskAssignedBulkTaskPayload{
			WorkerBatches: workerBatches,
		},
	)
}

func taskBulkAssignedTask(tenantId string, workerIdsToTaskIds map[string][]int64) (*msgqueue.Message, error) {
	workerBatches := make(map[string][]tasktypes.TaskAssignedBatch, len(workerIdsToTaskIds))

	for workerId, taskIds := range workerIdsToTaskIds {
		if len(taskIds) == 0 {
			continue
		}

		copied := make([]int64, len(taskIds))
		copy(copied, taskIds)

		workerBatches[workerId] = append(workerBatches[workerId], tasktypes.TaskAssignedBatch{
			BatchID:   "",
			BatchSize: len(copied),
			TaskIds:   copied,
		})
	}

	return taskAssignedBatchMessage(tenantId, workerBatches)
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
	case "task-assigned-bulk":
		err = s.handleDeadLetteredTaskBulkAssigned(ctx, msg)
	case "task-cancelled":
		err = s.handleDeadLetteredTaskCancelled(ctx, msg)
	default:
		err = fmt.Errorf("unknown task: %s", msg.ID)
	}

	return err
}

func (s *Scheduler) handleDeadLetteredTaskBulkAssigned(ctx context.Context, msg *msgqueue.Message) error {
	msgs := msgqueue.JSONConvert[tasktypes.TaskAssignedBulkTaskPayload](msg.Payloads)

	taskIds := make([]int64, 0)

	for _, innerMsg := range msgs {
		for workerID, batches := range innerMsg.WorkerBatches {
			for _, batch := range batches {
				s.l.Error().Msgf("handling dead-lettered task assignments for tenant %s, worker %s, batch %s tasks: %v. This indicates an abrupt shutdown of a dispatcher and should be investigated.", msg.TenantID, workerID, batch.BatchID, batch.TaskIds)
				taskIds = append(taskIds, batch.TaskIds...)
			}
		}
	}

	toFail, err := s.repov1.Tasks().ListTasks(ctx, msg.TenantID, taskIds)

	if err != nil {
		return fmt.Errorf("could not list tasks for dead lettered bulk assigned message: %w", err)
	}

	for _, _task := range toFail {
		tenantId := msg.TenantID
		task := _task.Task

		msg, err := tasktypes.FailedTaskMessage(
			tenantId,
			task.ID,
			task.InsertedAt,
			sqlchelpers.UUIDToStr(task.ExternalID),
			sqlchelpers.UUIDToStr(task.WorkflowRunID),
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

func (s *Scheduler) handleDeadLetteredTaskCancelled(ctx context.Context, msg *msgqueue.Message) error {
	payloads := msgqueue.JSONConvert[tasktypes.SignalTaskCancelledPayload](msg.Payloads)

	// try to resend the cancellation signal to the impacted worker.
	workerIds := make([]string, 0)

	for _, p := range payloads {
		s.l.Error().Msgf("handling dead-lettered task cancellations for tenant %s, task %d. This indicates an abrupt shutdown of a dispatcher and should be investigated.", msg.TenantID, p.TaskId)
		workerIds = append(workerIds, p.WorkerId)
	}

	// since the dispatcher IDs may have changed since the previous send, we need to query them again
	dispatcherIdWorkerIds, err := s.repo.Worker().GetDispatcherIdsForWorkers(ctx, msg.TenantID, workerIds)

	if err != nil {
		return fmt.Errorf("could not list dispatcher ids for workers: %w", err)
	}

	workerIdToDispatcherId := make(map[string]string)

	for dispatcherId, workerIds := range dispatcherIdWorkerIds {
		for _, workerId := range workerIds {
			workerIdToDispatcherId[workerId] = dispatcherId
		}
	}

	dispatcherIdsToPayloads := make(map[string][]tasktypes.SignalTaskCancelledPayload)

	for _, p := range payloads {
		// if we no longer have the worker attached to a dispatcher, discard the message
		if _, ok := workerIdToDispatcherId[p.WorkerId]; !ok {
			continue
		}

		pcp := *p
		dispatcherId := workerIdToDispatcherId[pcp.WorkerId]

		dispatcherIdsToPayloads[dispatcherId] = append(dispatcherIdsToPayloads[dispatcherId], pcp)
	}

	for dispatcherId, payloads := range dispatcherIdsToPayloads {
		msg, err := msgqueue.NewTenantMessage(
			msg.TenantID,
			"task-cancelled",
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
