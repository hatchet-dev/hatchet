package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/codes"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

type CheckTenantQueuesPayload struct {
	QueueNames    []string `json:"queue_name"`
	StrategyIds   []int64  `json:"strategy_ids"`
	SlotsReleased bool     `json:"slots_released"`
}

func NotifyTaskCreated(tenantId uuid.UUID, tasks []*V1TaskWithPayload) (*msgqueue.Message, error) {
	uniqueQueueNames := make(map[string]struct{})
	uniqueStrategies := make(map[int64]struct{})

	for _, task := range tasks {
		uniqueQueueNames[task.Queue] = struct{}{}

		for _, strategyId := range task.ConcurrencyStrategyIds {
			uniqueStrategies[strategyId] = struct{}{}
		}
	}

	payload := CheckTenantQueuesPayload{
		QueueNames:  make([]string, 0, len(uniqueQueueNames)),
		StrategyIds: make([]int64, 0, len(uniqueStrategies)),
	}

	for queueName := range uniqueQueueNames {
		payload.QueueNames = append(payload.QueueNames, queueName)
	}

	for strategyId := range uniqueStrategies {
		payload.StrategyIds = append(payload.StrategyIds, strategyId)
	}

	return msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDCheckTenantQueue,
		true,
		false,
		payload,
	)
}

func NewInternalEventMessage(tenantId uuid.UUID, events ...InternalTaskEvent) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDInternalEvent,
		false,
		true,
		events...,
	)
}

// postCommitTimeout bounds the non-transactional side effects which run after a
// staging transaction commits.
const postCommitTimeout = 30 * time.Second

// OLAPSignaler orchestrates the messaging side effects of task lifecycle changes.
// OLAP messages are staged in the OLAP outbox on the caller's transaction, so they
// commit atomically with the caller's writes; non-transactional side effects
// (scheduler-partition notifies, internal events, prometheus counters) are returned
// as a post-commit closure which repository methods run after their commit — either
// inline, or async via OptimisticTx.AddPostCommit.
type OLAPSignaler struct {
	shared *sharedRepository

	// tenant is set in NewRepository (the cached tenant repository is constructed
	// after the shared repository).
	tenant TenantRepository

	// mq, pubsub and promGate are wired at startup via Repository.SetMessagePublisher.
	mq       msgqueue.MessageQueue
	pubsub   msgqueue.PubSub
	promGate *prometheus.Gate
}

func newOLAPSignaler(shared *sharedRepository) *OLAPSignaler {
	return &OLAPSignaler{
		shared: shared,
	}
}

// SendInternalEvents publishes internal task lifecycle events to the task processing
// queue. Unlike OLAP staging, this drives workflow progression, so an unwired message
// queue is an error rather than a no-op.
func (s *OLAPSignaler) SendInternalEvents(ctx context.Context, tenantId uuid.UUID, events []InternalTaskEvent) error {
	ctx, span := telemetry.NewSpan(ctx, "OLAPSignaler.SendInternalEvents")
	defer span.End()

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "tenant.id", Value: tenantId})

	if len(events) == 0 {
		return nil
	}

	if s.mq == nil {
		return fmt.Errorf("cannot send internal events: message queue is not wired")
	}

	msg, err := NewInternalEventMessage(tenantId, events...)

	if err != nil {
		err = fmt.Errorf("could not create internal event message: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "could not create internal event message")
		return err
	}

	return s.mq.SendMessage(
		ctx,
		msgqueue.TASK_PROCESSING_QUEUE,
		msg,
	)
}

type bucketedTasks struct {
	queued    []*V1TaskWithPayload
	failed    []*V1TaskWithPayload
	cancelled []*V1TaskWithPayload
	skipped   []*V1TaskWithPayload
}

func bucketTasksByInitialState(tasks []*V1TaskWithPayload) bucketedTasks {
	b := bucketedTasks{}

	for _, task := range tasks {
		switch task.InitialState {
		case sqlcv1.V1TaskInitialStateQUEUED:
			b.queued = append(b.queued, task)
		case sqlcv1.V1TaskInitialStateFAILED:
			b.failed = append(b.failed, task)
		case sqlcv1.V1TaskInitialStateCANCELLED:
			b.cancelled = append(b.cancelled, task)
		case sqlcv1.V1TaskInitialStateSKIPPED:
			b.skipped = append(b.skipped, task)
		}
	}

	return b
}

// monitoringEvents builds the initial-state monitoring event payloads for the buckets.
func (b bucketedTasks) monitoringEvents() []CreateMonitoringEventPayload {
	events := make([]CreateMonitoringEventPayload, 0, len(b.queued)+len(b.failed)+len(b.cancelled)+len(b.skipped))

	for _, task := range b.queued {
		msg := ""

		if len(task.ConcurrencyKeys) > 0 {
			msg = "concurrency keys evaluated as:"

			for _, key := range task.ConcurrencyKeys {
				msg += fmt.Sprintf(" %s", key)
			}
		}

		events = append(events, CreateMonitoringEventPayload{
			TaskId:         task.ID,
			RetryCount:     task.RetryCount,
			EventType:      sqlcv1.V1EventTypeOlapQUEUED,
			EventTimestamp: time.Now(),
			EventMessage:   msg,
		})
	}

	for _, task := range b.failed {
		events = append(events, CreateMonitoringEventPayload{
			TaskId:         task.ID,
			RetryCount:     task.RetryCount,
			EventType:      sqlcv1.V1EventTypeOlapFAILED,
			EventPayload:   task.InitialStateReason.String,
			EventTimestamp: time.Now(),
		})
	}

	for _, task := range b.cancelled {
		events = append(events, CreateMonitoringEventPayload{
			TaskId:         task.ID,
			RetryCount:     task.RetryCount,
			EventType:      sqlcv1.V1EventTypeOlapCANCELLED,
			EventTimestamp: time.Now(),
		})
	}

	for _, task := range b.skipped {
		events = append(events, CreateMonitoringEventPayload{
			TaskId:         task.ID,
			RetryCount:     task.RetryCount,
			EventType:      sqlcv1.V1EventTypeOlapSKIPPED,
			EventTimestamp: time.Now(),
		})
	}

	return events
}

// internalEvents builds the terminal-state internal events for the buckets: tasks
// created in a terminal state still need to drive downstream event matches.
func (b bucketedTasks) internalEvents(tenantId uuid.UUID) []InternalTaskEvent {
	events := make([]InternalTaskEvent, 0, len(b.failed)+len(b.cancelled)+len(b.skipped))

	for _, task := range b.cancelled {
		events = append(events, InternalTaskEvent{
			TenantID:       tenantId,
			TaskID:         task.ID,
			TaskExternalID: task.ExternalID,
			RetryCount:     task.RetryCount,
			EventType:      sqlcv1.V1TaskEventTypeCANCELLED,
			Data:           NewCancelledTaskOutputEventFromTask(task).Bytes(),
		})
	}

	for _, task := range b.failed {
		events = append(events, InternalTaskEvent{
			TenantID:       tenantId,
			TaskID:         task.ID,
			TaskExternalID: task.ExternalID,
			RetryCount:     task.RetryCount,
			EventType:      sqlcv1.V1TaskEventTypeFAILED,
			Data:           NewFailedTaskOutputEventFromTask(task).Bytes(),
		})
	}

	for _, task := range b.skipped {
		events = append(events, InternalTaskEvent{
			TenantID:       tenantId,
			TaskID:         task.ID,
			TaskExternalID: task.ExternalID,
			RetryCount:     task.RetryCount,
			EventType:      sqlcv1.V1TaskEventTypeCOMPLETED,
			Data:           NewSkippedTaskOutputEventFromTask(task).Bytes(),
		})
	}

	return events
}

// tasksCreated stages created-dag/created-task messages and the initial-state
// monitoring events on tx, and returns the post-commit closure for the
// non-transactional side effects.
func (s *OLAPSignaler) tasksCreated(ctx context.Context, tx pgx.Tx, tenantId uuid.UUID, tasks []*V1TaskWithPayload, dags []*DAGWithData) (func(), error) {
	msgs := make([]*msgqueue.Message, 0, 3)

	if len(dags) > 0 {
		dagPayloads := make([]CreatedDAGPayload, len(dags))

		for i, dag := range dags {
			dagPayloads[i] = CreatedDAGPayload{DAGWithData: dag}
		}

		msg, err := msgqueue.NewTenantMessage(tenantId, msgqueue.MsgIDCreatedDAG, false, true, dagPayloads...)

		if err != nil {
			return nil, fmt.Errorf("could not create created-dag message: %w", err)
		}

		msgs = append(msgs, msg)
	}

	if len(tasks) > 0 {
		taskPayloads := make([]CreatedTaskPayload, len(tasks))

		for i, task := range tasks {
			taskPayloads[i] = CreatedTaskPayload{V1TaskWithPayload: task}
		}

		msg, err := msgqueue.NewTenantMessage(tenantId, msgqueue.MsgIDCreatedTask, false, true, taskPayloads...)

		if err != nil {
			return nil, fmt.Errorf("could not create created-task message: %w", err)
		}

		msgs = append(msgs, msg)
	}

	buckets := bucketTasksByInitialState(tasks)

	if events := buckets.monitoringEvents(); len(events) > 0 {
		msg, err := msgqueue.NewTenantMessage(tenantId, msgqueue.MsgIDCreateMonitoringEvent, false, true, events...)

		if err != nil {
			return nil, fmt.Errorf("could not create monitoring event message: %w", err)
		}

		msgs = append(msgs, msg)
	}

	if err := s.shared.olapOutbox.stage(ctx, tx, msgs...); err != nil {
		return nil, err
	}

	return s.postCommitSideEffects(tenantId, buckets), nil
}

// tasksUpdated stages the initial-state monitoring events for updated tasks (e.g.
// replays which reset existing tasks) — no created-task messages — and returns the
// post-commit closure.
func (s *OLAPSignaler) tasksUpdated(ctx context.Context, tx pgx.Tx, tenantId uuid.UUID, tasks []*V1TaskWithPayload) (func(), error) {
	buckets := bucketTasksByInitialState(tasks)

	if events := buckets.monitoringEvents(); len(events) > 0 {
		msg, err := msgqueue.NewTenantMessage(tenantId, msgqueue.MsgIDCreateMonitoringEvent, false, true, events...)

		if err != nil {
			return nil, fmt.Errorf("could not create monitoring event message: %w", err)
		}

		if err := s.shared.olapOutbox.stage(ctx, tx, msg); err != nil {
			return nil, err
		}
	}

	return s.postCommitSideEffects(tenantId, buckets), nil
}

// tasksReplayed stages RETRIED_BY_USER monitoring events for replayed tasks.
func (s *OLAPSignaler) tasksReplayed(ctx context.Context, tx pgx.Tx, tenantId uuid.UUID, tasks []TaskIdInsertedAtRetryCount) error {
	if len(tasks) == 0 {
		return nil
	}

	events := make([]CreateMonitoringEventPayload, len(tasks))

	for i, task := range tasks {
		events[i] = CreateMonitoringEventPayload{
			TaskId:         task.Id,
			RetryCount:     task.RetryCount,
			EventType:      sqlcv1.V1EventTypeOlapRETRIEDBYUSER,
			EventTimestamp: time.Now(),
			EventMessage:   "Task was replayed, resetting task result.",
		}
	}

	msg, err := msgqueue.NewTenantMessage(tenantId, msgqueue.MsgIDCreateMonitoringEvent, false, true, events...)

	if err != nil {
		return fmt.Errorf("could not create monitoring event message: %w", err)
	}

	return s.shared.olapOutbox.stage(ctx, tx, msg)
}

// eventsCreated stages the created-event-trigger message linking user events to the
// runs they triggered.
func (s *OLAPSignaler) eventsCreated(ctx context.Context, tx pgx.Tx, tenantId uuid.UUID, eventIdToOpts map[uuid.UUID]EventTriggerOpts, eventIdsToRuns map[uuid.UUID][]*Run) error {
	if len(eventIdsToRuns) == 0 {
		return nil
	}

	eventTriggerOpts := make([]CreatedEventTriggerPayloadSingleton, 0)

	for eventExternalId, runs := range eventIdsToRuns {
		opts := eventIdToOpts[eventExternalId]

		// need this for backwards compat when we deploy this version
		// can remove later
		seenAt := opts.SeenAt
		if seenAt.IsZero() {
			seenAt = time.Now().UTC()
		}

		if len(runs) == 0 {
			eventTriggerOpts = append(eventTriggerOpts, CreatedEventTriggerPayloadSingleton{
				EventSeenAt:             seenAt,
				EventKey:                opts.Key,
				EventExternalId:         opts.ExternalId,
				EventPayload:            opts.Data,
				EventAdditionalMetadata: opts.AdditionalMetadata,
				TriggeringWebhookName:   opts.TriggeringWebhookName,
				EventScope:              opts.Scope,
			})
		} else {
			for _, run := range runs {
				runExtID := run.WorkflowRunExternalID
				eventTriggerOpts = append(eventTriggerOpts, CreatedEventTriggerPayloadSingleton{
					MaybeRunId:              &run.Id,
					MaybeRunInsertedAt:      &run.InsertedAt,
					MaybeRunExternalId:      &runExtID,
					EventSeenAt:             seenAt,
					EventKey:                opts.Key,
					EventExternalId:         opts.ExternalId,
					EventPayload:            opts.Data,
					EventAdditionalMetadata: opts.AdditionalMetadata,
					EventScope:              opts.Scope,
					FilterId:                run.FilterId,
					TriggeringWebhookName:   opts.TriggeringWebhookName,
				})
			}
		}
	}

	msg, err := CreatedEventTriggerMessage(tenantId, CreatedEventTriggerPayload{Payloads: eventTriggerOpts})

	if err != nil {
		return fmt.Errorf("could not create event trigger message: %w", err)
	}

	return s.shared.olapOutbox.stage(ctx, tx, msg)
}

// celEvaluationFailures stages a cel-evaluation-failure message.
func (s *OLAPSignaler) celEvaluationFailures(ctx context.Context, tx pgx.Tx, tenantId uuid.UUID, failures []CELEvaluationFailure) error {
	if len(failures) == 0 {
		return nil
	}

	msg, err := CELEvaluationFailureMessage(tenantId, failures)

	if err != nil {
		return fmt.Errorf("could not create CEL evaluation failure message: %w", err)
	}

	return s.shared.olapOutbox.stage(ctx, tx, msg)
}

// postCommitSideEffects returns the closure covering the side effects which must not
// run inside the staging transaction: the scheduler-partition notify for queued
// tasks, internal events for tasks created in a terminal state, and prometheus
// counters. Errors are logged, not returned — the durable writes have already
// committed. Depending on the transaction style, repository methods run the closure
// inline after commit or async via OptimisticTx.AddPostCommit.
func (s *OLAPSignaler) postCommitSideEffects(tenantId uuid.UUID, buckets bucketedTasks) func() {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), postCommitTimeout)
		defer cancel()

		if len(buckets.queued) > 0 {
			s.notifyScheduler(ctx, tenantId, buckets.queued)
		}

		if events := buckets.internalEvents(tenantId); len(events) > 0 {
			if err := s.SendInternalEvents(ctx, tenantId, events); err != nil {
				s.shared.l.Err(err).Ctx(ctx).Msg("could not send internal events for created tasks")
			}
		}

		s.recordMetrics(ctx, tenantId, buckets)
	}
}

// notifyScheduler wakes the tenant's scheduler partition so queued tasks are
// scheduled promptly.
func (s *OLAPSignaler) notifyScheduler(ctx context.Context, tenantId uuid.UUID, tasks []*V1TaskWithPayload) {
	if s.pubsub == nil || s.tenant == nil {
		return
	}

	tenant, err := s.tenant.GetTenantByID(ctx, tenantId)

	if err != nil {
		s.shared.l.Err(err).Ctx(ctx).Msg("could not get tenant for scheduler notification")
		return
	}

	if !tenant.SchedulerPartitionId.Valid {
		return
	}

	msg, err := NotifyTaskCreated(tenantId, tasks)

	if err != nil {
		s.shared.l.Err(err).Ctx(ctx).Str("scheduler_partition_id", tenant.SchedulerPartitionId.String).Msg("could not create message for scheduler partition topic")
		return
	}

	err = s.pubsub.Pub(
		ctx,
		msgqueue.SchedulerPartitionTopic(tenant.SchedulerPartitionId.String),
		msg,
	)

	if err != nil {
		s.shared.l.Err(err).Ctx(ctx).Str("scheduler_partition_id", tenant.SchedulerPartitionId.String).Msg("could not publish message to scheduler partition topic")
	}
}

func (s *OLAPSignaler) recordMetrics(ctx context.Context, tenantId uuid.UUID, buckets bucketedTasks) {
	if s.promGate == nil {
		return
	}

	tenantMetricsEnabled := s.promGate.Enabled(ctx, tenantId)

	record := func(count int, counters ...func(tenantEnabled bool)) {
		for i := 0; i < count; i++ {
			for _, counter := range counters {
				counter(tenantMetricsEnabled)
			}
		}
	}

	record(len(buckets.queued)+len(buckets.failed)+len(buckets.cancelled)+len(buckets.skipped), func(tenantEnabled bool) {
		prometheus.CreatedTasks.Inc()
		if tenantEnabled {
			prometheus.TenantCreatedTasks.WithLabelValues(tenantId.String()).Inc()
		}
	})

	record(len(buckets.failed), func(tenantEnabled bool) {
		prometheus.FailedTasks.Inc()
		if tenantEnabled {
			prometheus.TenantFailedTasks.WithLabelValues(tenantId.String()).Inc()
		}
	})

	record(len(buckets.cancelled), func(tenantEnabled bool) {
		prometheus.CancelledTasks.Inc()
		if tenantEnabled {
			prometheus.TenantCancelledTasks.WithLabelValues(tenantId.String()).Inc()
		}
	})

	record(len(buckets.skipped), func(tenantEnabled bool) {
		prometheus.SkippedTasks.Inc()
		if tenantEnabled {
			prometheus.TenantSkippedTasks.WithLabelValues(tenantId.String()).Inc()
		}
	})
}

// composePostCommit merges post-commit closures, skipping nils.
func composePostCommit(fns ...func()) func() {
	return func() {
		for _, fn := range fns {
			if fn != nil {
				fn()
			}
		}
	}
}

// stageTriggerSignals stages every OLAP message produced by a workflow trigger on the
// trigger transaction: created-task/created-dag, the initial-state monitoring events,
// the created-event-trigger message (for event-triggered workflows), and any CEL
// evaluation failures. It returns the post-commit closure for the non-transactional
// side effects.
func (r *sharedRepository) stageTriggerSignals(
	ctx context.Context,
	tx sqlcv1.DBTX,
	tenantId uuid.UUID,
	tasks []*V1TaskWithPayload,
	dags []*DAGWithData,
	coreEvents *createCoreUserEventOpts,
	celFailures []CELEvaluationFailure,
) (func(), error) {
	pgxTx, ok := tx.(pgx.Tx)

	if !ok {
		return nil, fmt.Errorf("cannot stage olap messages: tx does not implement pgx.Tx")
	}

	postSignal, err := r.signaler.tasksCreated(ctx, pgxTx, tenantId, tasks, dags)

	if err != nil {
		return nil, err
	}

	allFailures := celFailures

	if coreEvents != nil {
		eventIdToOpts := make(map[uuid.UUID]EventTriggerOpts, len(coreEvents.opts))

		for _, opt := range coreEvents.opts {
			eventIdToOpts[opt.ExternalId] = opt
		}

		eventIdsToRuns := getEventExternalIdToRuns(coreEvents.opts, coreEvents.externalIdToEventIdAndFilterId, tasks, dags)

		if err := r.signaler.eventsCreated(ctx, pgxTx, tenantId, eventIdToOpts, eventIdsToRuns); err != nil {
			return nil, err
		}

		if len(coreEvents.celEvaluationFailures) > 0 {
			allFailures = append(append([]CELEvaluationFailure{}, celFailures...), coreEvents.celEvaluationFailures...)
		}
	}

	if err := r.signaler.celEvaluationFailures(ctx, pgxTx, tenantId, allFailures); err != nil {
		return nil, err
	}

	return postSignal, nil
}
