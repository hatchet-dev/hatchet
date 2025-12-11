package v1

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1repo "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

const defaultBatchPollInterval = 200 * time.Millisecond
const batchFetchLimit int32 = 256

type BatchFlushReason string

const (
	flushReasonBatchSizeReached  BatchFlushReason = "batch_size_reached"
	flushReasonWorkerChanged     BatchFlushReason = "worker_changed"
	flushReasonDispatcherChanged BatchFlushReason = "dispatcher_changed"
	flushReasonIntervalElapsed   BatchFlushReason = "interval_elapsed"
	flushReasonBufferDrained     BatchFlushReason = "buffer_drained"
)

// BatchScheduler coordinates batching for a (step_id, batch_key) pair. It buffers queue
// items in-memory, flushing them once batch requirements are satisfied.
type BatchScheduler struct {
	cf             *sharedConfig
	tenantId       pgtype.UUID
	stepId         pgtype.UUID
	batchKey       string
	repo           v1repo.BatchQueueRepository
	queueFactory   v1repo.QueueFactoryRepository
	scheduler      *Scheduler
	emitResults    func(*QueueResults)
	assignOverride assignmentFn
	reserveBatch   batchReservationFunc
	maxRuns        int

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}

	buffer        []*sqlcv1.V1BatchedQueueItem
	afterID       int64
	flushDeadline *time.Time

	batchSize     int
	flushInterval time.Duration

	l zerolog.Logger
}

type assignmentFn func(ctx context.Context, queueItems []*sqlcv1.V1QueueItem, labels map[string][]*sqlcv1.GetDesiredLabelsRow, rateLimits map[int64]map[string]int32) ([]*assignedQueueItem, []*sqlcv1.V1QueueItem, error)

type batchReservationFunc func(context.Context, *BatchReservationRequest) (bool, error)

type BatchReservationRequest struct {
	TenantID string
	StepID   pgtype.UUID
	ActionID string
	BatchKey string
	BatchID  string
	MaxRuns  int
}

func newBatchScheduler(
	cf *sharedConfig,
	tenantId pgtype.UUID,
	resource *sqlcv1.ListDistinctBatchResourcesRow,
	queueFactory v1repo.QueueFactoryRepository,
	scheduler *Scheduler,
	emitResults func(*QueueResults),
	reserveBatch batchReservationFunc,
) *BatchScheduler {
	if resource == nil {
		return nil
	}

	batchKey := ""
	if resource.BatchKey != "" {
		batchKey = resource.BatchKey
	}

	logger := cf.l.With().
		Str("tenant_id", sqlchelpers.UUIDToStr(tenantId)).
		Str("step_id", sqlchelpers.UUIDToStr(resource.StepID)).
		Str("batch_key", batchKey).
		Logger()

	batchSize := resource.BatchSize
	if batchSize <= 0 {
		batchSize = 1
	}

	var flushInterval time.Duration
	if resource.BatchFlushIntervalMs > 0 {
		flushInterval = time.Duration(resource.BatchFlushIntervalMs) * time.Millisecond
	}

	return &BatchScheduler{
		cf:            cf,
		tenantId:      tenantId,
		stepId:        resource.StepID,
		batchKey:      batchKey,
		repo:          cf.repo.BatchQueue().NewBatchQueue(tenantId),
		queueFactory:  queueFactory,
		scheduler:     scheduler,
		emitResults:   emitResults,
		reserveBatch:  reserveBatch,
		batchSize:     int(batchSize),
		flushInterval: flushInterval,
		maxRuns:       int(resource.BatchMaxRuns),
		l:             logger,
	}
}

func (b *BatchScheduler) Start(ctx context.Context) {
	if b.cancel != nil {
		// already started
		return
	}

	b.ctx, b.cancel = context.WithCancel(ctx)
	b.done = make(chan struct{})

	go b.run()
}

func (b *BatchScheduler) Cleanup(ctx context.Context) error {
	if b.cancel != nil {
		b.cancel()
	}

	if b.done != nil {
		select {
		case <-b.done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

func (b *BatchScheduler) run() {
	defer close(b.done)

	if b.batchSize <= 0 {
		// fall back to singleton batching if unspecified
		b.batchSize = 1
	}

	ticker := time.NewTicker(defaultBatchPollInterval)
	defer ticker.Stop()

	for {
		if err := b.tick(); err != nil {
			if b.ctx.Err() != nil {
				return
			}

			b.l.Error().Err(err).Msg("batch scheduler tick failed")
		}

		select {
		case <-b.ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (b *BatchScheduler) tick() error {
	if err := b.fetchNewItems(); err != nil {
		return err
	}

	// Check for timed out items in the buffer
	if err := b.checkBufferForTimeouts(); err != nil {
		return err
	}

	if b.batchSize > 0 && len(b.buffer) >= b.batchSize {
		if err := b.flush(b.batchSize, flushReasonBatchSizeReached); err != nil {
			return err
		}
	}

	if b.flushDeadline != nil && time.Now().After(*b.flushDeadline) && len(b.buffer) > 0 {
		if err := b.flush(len(b.buffer), flushReasonIntervalElapsed); err != nil {
			return err
		}
	}

	return nil
}

func (b *BatchScheduler) fetchNewItems() error {
	if b.ctx.Err() != nil {
		return b.ctx.Err()
	}

	after := pgtype.Int8{}
	if b.afterID > 0 {
		after.Int64 = b.afterID
		after.Valid = true
	}

	limit := pgtype.Int4{
		Int32: batchFetchLimit,
		Valid: true,
	}

	items, err := b.repo.ListBatchedQueueItems(b.ctx, b.stepId, b.batchKey, after, limit.Int32)
	if err != nil {
		return err
	}

	if len(items) == 0 {
		return nil
	}

	if b.flushInterval > 0 && len(b.buffer) == 0 && b.flushDeadline == nil {
		deadline := time.Now().Add(b.flushInterval)
		b.flushDeadline = &deadline
	}

	newItems := make([]*sqlcv1.V1BatchedQueueItem, 0, len(items))
	timedOutItems := make([]*sqlcv1.V1BatchedQueueItem, 0)

	now := time.Now().UTC()

	for _, item := range items {
		if item == nil {
			continue
		}

		// Check if item has exceeded schedule timeout
		if item.ScheduleTimeoutAt.Valid && item.ScheduleTimeoutAt.Time.Before(now) {
			timedOutItems = append(timedOutItems, item)
			if item.ID > b.afterID {
				b.afterID = item.ID
			}
			continue
		}

		b.buffer = append(b.buffer, item)
		if item.ID > b.afterID {
			b.afterID = item.ID
		}

		newItems = append(newItems, item)
	}

	// Handle timed out items
	if len(timedOutItems) > 0 {
		if err := b.handleScheduleTimeouts(timedOutItems); err != nil {
			b.l.Error().Err(err).Msg("failed to handle schedule timeouts")
		}
	}

	b.emitWaitingEvents(newItems)

	return nil
}

func (b *BatchScheduler) flush(count int, reason BatchFlushReason) error {
	if len(b.buffer) == 0 || count <= 0 {
		return nil
	}

	if count > len(b.buffer) {
		count = len(b.buffer)
	}

	toFlush := make([]*sqlcv1.V1BatchedQueueItem, 0, count)
	toFlush = append(toFlush, b.buffer[:count]...)

	remaining, err := b.assignAndDispatch(b.ctx, toFlush, reason)
	if err != nil {
		return err
	}

	b.buffer = b.buffer[count:]

	if len(remaining) > 0 {
		// Requeue remaining items at the front to preserve ordering.
		b.buffer = append(remaining, b.buffer...)
	}

	if b.flushInterval > 0 && len(b.buffer) > 0 {
		deadline := time.Now().Add(b.flushInterval)
		b.flushDeadline = &deadline
	} else {
		b.flushDeadline = nil
	}

	return nil
}

func (b *BatchScheduler) emitWaitingEvents(newItems []*sqlcv1.V1BatchedQueueItem) {
	if b.emitResults == nil || len(newItems) == 0 {
		return
	}

	flushIntervalMs := int32(0)
	if b.flushInterval > 0 {
		flushIntervalMs = int32(b.flushInterval / time.Millisecond)
	}

	var nextFlush *time.Time
	if b.flushDeadline != nil {
		copyDeadline := *b.flushDeadline
		nextFlush = &copyDeadline
	}

	pending := int32(len(b.buffer))

	buffered := make([]*v1repo.AssignedItem, 0, len(newItems))

	for _, item := range newItems {
		if item == nil {
			continue
		}

		queueItem := &sqlcv1.V1QueueItem{
			ID:                item.ID,
			TenantID:          item.TenantID,
			Queue:             item.Queue,
			TaskID:            item.TaskID,
			TaskInsertedAt:    item.TaskInsertedAt,
			ExternalID:        item.ExternalID,
			ActionID:          item.ActionID,
			StepID:            item.StepID,
			WorkflowID:        item.WorkflowID,
			WorkflowRunID:     item.WorkflowRunID,
			ScheduleTimeoutAt: item.ScheduleTimeoutAt,
			StepTimeout:       item.StepTimeout,
			Priority:          item.Priority,
			Sticky:            item.Sticky,
			DesiredWorkerID:   item.DesiredWorkerID,
			RetryCount:        item.RetryCount,
			BatchKey: pgtype.Text{
				String: item.BatchKey,
				Valid:  strings.TrimSpace(item.BatchKey) != "",
			},
		}

		triggeredAt := time.Now().UTC()
		if item.InsertedAt.Valid {
			triggeredAt = item.InsertedAt.Time.UTC()
		}

		metaBatchKey := ""
		if queueItem.BatchKey.Valid {
			metaBatchKey = strings.TrimSpace(queueItem.BatchKey.String)
		}

		buffered = append(buffered, &v1repo.AssignedItem{
			QueueItem: queueItem,
			Batch: &v1repo.BatchAssignmentMetadata{
				State:                     "waiting",
				TriggeredAt:               triggeredAt,
				ConfiguredBatchSize:       int32(b.batchSize),
				ConfiguredFlushIntervalMs: flushIntervalMs,
				MaxRuns:                   int32(b.maxRuns),
				Pending:                   pending,
				NextFlushAt:               nextFlush,
				BatchID:                   "",
				StepID:                    sqlchelpers.UUIDToStr(queueItem.StepID),
				ActionID:                  queueItem.ActionID,
				BatchKey:                  metaBatchKey,
			},
		})
	}

	if len(buffered) == 0 {
		return
	}

	b.emitResults(&QueueResults{
		TenantId: b.tenantId,
		Buffered: buffered,
	})
}

func (b *BatchScheduler) assignQueueItems(
	ctx context.Context,
	queueItems []*sqlcv1.V1QueueItem,
	labels map[string][]*sqlcv1.GetDesiredLabelsRow,
	rateLimits map[int64]map[string]int32,
) ([]*assignedQueueItem, []*sqlcv1.V1QueueItem, error) {
	if b.assignOverride != nil {
		return b.assignOverride(ctx, queueItems, labels, rateLimits)
	}

	resultsCh := b.scheduler.tryAssign(ctx, queueItems, labels, rateLimits)

	assigned := make([]*assignedQueueItem, 0)
	failed := make([]*sqlcv1.V1QueueItem, 0)

	for res := range resultsCh {
		if res == nil {
			continue
		}

		assigned = append(assigned, res.assigned...)

		for _, unassigned := range res.unassigned {
			if unassigned != nil {
				failed = append(failed, unassigned)
			}
		}

		for _, rl := range res.rateLimited {
			if rl != nil && rl.qi != nil {
				failed = append(failed, rl.qi)
			}
		}

		for _, rl := range res.rateLimitedToMove {
			if rl != nil && rl.qi != nil {
				failed = append(failed, rl.qi)
			}
		}
	}

	return assigned, failed, nil
}

func (b *BatchScheduler) assignAndDispatch(ctx context.Context, items []*sqlcv1.V1BatchedQueueItem, reason BatchFlushReason) ([]*sqlcv1.V1BatchedQueueItem, error) {
	if b.scheduler == nil {
		return items, fmt.Errorf("batch scheduler missing core scheduler")
	}

	if b.queueFactory == nil {
		return items, fmt.Errorf("batch scheduler missing queue factory")
	}

	if b.emitResults == nil {
		return items, fmt.Errorf("batch scheduler missing results emitter")
	}

	// Check for timed out items before attempting to assign (matching regular scheduler behavior)
	nonTimedOutItems := make([]*sqlcv1.V1BatchedQueueItem, 0, len(items))
	timedOutItems := make([]*sqlcv1.V1BatchedQueueItem, 0)
	now := time.Now().UTC()

	for _, item := range items {
		if item == nil {
			continue
		}

		// Check if item has exceeded schedule timeout
		if item.ScheduleTimeoutAt.Valid && item.ScheduleTimeoutAt.Time.Before(now) {
			timedOutItems = append(timedOutItems, item)
		} else {
			nonTimedOutItems = append(nonTimedOutItems, item)
		}
	}

	// Handle timed out items
	if len(timedOutItems) > 0 {
		if err := b.handleScheduleTimeouts(timedOutItems); err != nil {
			b.l.Error().Err(err).Msg("failed to handle schedule timeouts during assignment")
		}
	}

	// Use non-timed-out items for assignment
	items = nonTimedOutItems

	if len(items) == 0 {
		return nil, nil
	}

	queueToItems := make(map[string][]*sqlcv1.V1BatchedQueueItem)

	for _, item := range items {
		if item == nil {
			continue
		}

		queueToItems[item.Queue] = append(queueToItems[item.Queue], item)
	}

	remaining := make([]*sqlcv1.V1BatchedQueueItem, 0)
	allAssigned := make([]*v1repo.AssignedItem, 0)
	allAckIds := make([]int, 0)

	for queueName, group := range queueToItems {
		b.l.Debug().
			Str("queue", queueName).
			Int("group_size", len(group)).
			Msg("processing batched queue group")
		queueRepo := b.queueFactory.NewQueue(b.tenantId, queueName)
		if queueRepo == nil {
			remaining = append(remaining, group...)
			continue
		}

		queueItems := make([]*sqlcv1.V1QueueItem, 0, len(group))
		queueItemsByID := make(map[int64]*sqlcv1.V1QueueItem, len(group))
		idToBatched := make(map[int64]*sqlcv1.V1BatchedQueueItem, len(group))

		stepID := b.stepId
		actionID := ""
		batchKey := strings.TrimSpace(b.batchKey)

		if len(group) > 0 && group[0] != nil {
			if group[0].StepID.Valid {
				stepID = group[0].StepID
			}

			actionID = group[0].ActionID

			if trimmed := strings.TrimSpace(group[0].BatchKey); trimmed != "" {
				batchKey = trimmed
			}
		}

		for _, batched := range group {
			if batched == nil {
				continue
			}

			queueItem := &sqlcv1.V1QueueItem{
				ID:                batched.ID,
				TenantID:          batched.TenantID,
				Queue:             batched.Queue,
				TaskID:            batched.TaskID,
				TaskInsertedAt:    batched.TaskInsertedAt,
				ExternalID:        batched.ExternalID,
				ActionID:          batched.ActionID,
				StepID:            batched.StepID,
				WorkflowID:        batched.WorkflowID,
				WorkflowRunID:     batched.WorkflowRunID,
				ScheduleTimeoutAt: batched.ScheduleTimeoutAt,
				StepTimeout:       batched.StepTimeout,
				Priority:          batched.Priority,
				Sticky:            batched.Sticky,
				DesiredWorkerID:   batched.DesiredWorkerID,
				RetryCount:        batched.RetryCount,
				BatchKey: pgtype.Text{
					String: batched.BatchKey,
					Valid:  strings.TrimSpace(batched.BatchKey) != "",
				},
			}

			queueItems = append(queueItems, queueItem)
			queueItemsByID[queueItem.ID] = queueItem
			idToBatched[queueItem.ID] = batched
		}

		if len(queueItems) == 0 {
			queueRepo.Cleanup()
			continue
		}

		b.l.Debug().
			Str("queue", queueName).
			Int("queue_items", len(queueItems)).
			Msg("built queue items for batched group")

		schedulingItem := queueItems[0]

		stepLabels, err := queueRepo.GetDesiredLabels(ctx, []pgtype.UUID{b.stepId})
		if err != nil {
			queueRepo.Cleanup()
			return items, fmt.Errorf("get desired labels: %w", err)
		}

		rateLimits, err := queueRepo.GetTaskRateLimits(ctx, []*sqlcv1.V1QueueItem{schedulingItem})
		if err != nil {
			queueRepo.Cleanup()
			return items, fmt.Errorf("get task rate limits: %w", err)
		}

		stepKey := sqlchelpers.UUIDToStr(b.stepId)
		assigned, failedQueueItems, err := b.assignQueueItems(ctx, []*sqlcv1.V1QueueItem{schedulingItem}, map[string][]*sqlcv1.GetDesiredLabelsRow{
			stepKey: stepLabels[stepKey],
		}, rateLimits)
		if err != nil {
			queueRepo.Cleanup()
			return items, err
		}

		queueRepo.Cleanup()

		ackIds := make([]int, 0, len(assigned))

		requeueGroup := func() {
			if len(ackIds) > 0 {
				b.scheduler.nack(ackIds)
			}

			remaining = append(remaining, group...)
			b.emitWaitingEvents(group)

			if b.flushInterval > 0 && b.flushDeadline == nil {
				deadline := time.Now().Add(b.flushInterval)
				b.flushDeadline = &deadline
			}
		}

		if len(assigned) == 0 || len(failedQueueItems) > 0 {
			requeueGroup()
			continue
		}

		assignedItem := assigned[0]
		if assignedItem == nil || assignedItem.QueueItem == nil || !assignedItem.WorkerId.Valid {
			requeueGroup()
			continue
		}

		workerID := assignedItem.WorkerId
		if assignedItem.AckId > 0 {
			ackIds = append(ackIds, assignedItem.AckId)
		}

		batchKeyNormalized := strings.TrimSpace(batchKey)
		if b.maxRuns > 0 && batchKeyNormalized != "" && b.cf.taskRepo != nil {
			stepIDStr := sqlchelpers.UUIDToStr(stepID)
			tenantIDStr := sqlchelpers.UUIDToStr(b.tenantId)

			activeCount, err := b.cf.taskRepo.CountActiveTaskBatchRuns(ctx, tenantIDStr, stepIDStr, batchKeyNormalized)
			if err != nil {
				b.l.Error().Err(err).Msg("failed counting active batch runs; deferring batch")
				requeueGroup()
				continue
			}

			if activeCount >= b.maxRuns {
				requeueGroup()
				continue
			}
		}

		batchID := uuid.NewString()
		allowed := true

		if b.reserveBatch != nil && b.maxRuns > 0 && batchKeyNormalized != "" {
			req := &BatchReservationRequest{
				TenantID: sqlchelpers.UUIDToStr(b.tenantId),
				StepID:   stepID,
				ActionID: actionID,
				BatchKey: batchKeyNormalized,
				BatchID:  batchID,
				MaxRuns:  b.maxRuns,
			}

			var err error
			allowed, err = b.reserveBatch(ctx, req)
			if err != nil {
				b.l.Error().Err(err).Msg("failed to reserve batch run")
				allowed = false
			}
		}

		if !allowed {
			requeueGroup()
			continue
		}

		batchAssignments := make([]*v1repo.BatchAssignment, 0, len(group))
		queueResults := make([]*v1repo.AssignedItem, 0, len(group))
		triggeredAt := time.Now().UTC()

		flushIntervalMs := int32(0)
		if b.flushInterval > 0 {
			flushIntervalMs = int32(b.flushInterval / time.Millisecond)
		}

		for _, batched := range group {
			if batched == nil {
				continue
			}

			queueItem := queueItemsByID[batched.ID]
			if queueItem == nil {
				continue
			}

			batchAssignments = append(batchAssignments, &v1repo.BatchAssignment{
				BatchQueueItemID: batched.ID,
				TaskID:           batched.TaskID,
				TaskInsertedAt:   batched.TaskInsertedAt,
				RetryCount:       batched.RetryCount,
				WorkerID:         workerID,
				BatchID:          batchID,
				StepID:           stepID,
				ActionID:         actionID,
				BatchKey:         batchKeyNormalized,
			})

			queueResults = append(queueResults, &v1repo.AssignedItem{
				WorkerId:  workerID,
				QueueItem: queueItem,
				Batch: &v1repo.BatchAssignmentMetadata{
					State:                     "flushed",
					Reason:                    string(reason),
					TriggeredAt:               triggeredAt,
					ConfiguredBatchSize:       int32(b.batchSize),
					ConfiguredFlushIntervalMs: flushIntervalMs,
					MaxRuns:                   int32(b.maxRuns),
					Pending:                   0,
					NextFlushAt:               nil,
					BatchID:                   batchID,
					StepID:                    sqlchelpers.UUIDToStr(stepID),
					ActionID:                  actionID,
					BatchKey:                  batchKeyNormalized,
				},
			})
		}

		if len(batchAssignments) == 0 {
			if len(ackIds) > 0 {
				b.scheduler.nack(ackIds)
			}
			continue
		}

		if err := b.repo.CommitAssignments(ctx, batchAssignments); err != nil {
			if len(ackIds) > 0 {
				b.scheduler.nack(ackIds)
			}
			return items, fmt.Errorf("commit batch assignments: %w", err)
		}

		allAssigned = append(allAssigned, queueResults...)
		allAckIds = append(allAckIds, ackIds...)
	}

	if len(allAckIds) > 0 {
		b.scheduler.ack(allAckIds)
	}

	if len(allAssigned) > 0 {
		result := &QueueResults{
			TenantId: b.tenantId,
			Assigned: allAssigned,
		}
		b.emitResults(result)
	}

	return remaining, nil
}

func (b *BatchScheduler) checkBufferForTimeouts() error {
	if len(b.buffer) == 0 {
		return nil
	}

	timedOutItems := make([]*sqlcv1.V1BatchedQueueItem, 0)
	remainingItems := make([]*sqlcv1.V1BatchedQueueItem, 0, len(b.buffer))

	now := time.Now().UTC()

	for _, item := range b.buffer {
		if item == nil {
			continue
		}

		// Check if item has exceeded schedule timeout
		if item.ScheduleTimeoutAt.Valid && item.ScheduleTimeoutAt.Time.Before(now) {
			timedOutItems = append(timedOutItems, item)
		} else {
			remainingItems = append(remainingItems, item)
		}
	}

	// Update the buffer to only contain non-timed-out items
	b.buffer = remainingItems

	// Handle timed out items
	if len(timedOutItems) > 0 {
		if err := b.handleScheduleTimeouts(timedOutItems); err != nil {
			return err
		}
	}

	return nil
}

func (b *BatchScheduler) handleScheduleTimeouts(timedOutItems []*sqlcv1.V1BatchedQueueItem) error {
	if len(timedOutItems) == 0 {
		return nil
	}

	idsToDelete := make([]int64, 0, len(timedOutItems))
	schedulingTimedOut := make([]*sqlcv1.V1QueueItem, 0, len(timedOutItems))

	for _, item := range timedOutItems {
		if item == nil {
			continue
		}

		idsToDelete = append(idsToDelete, item.ID)

		queueItem := &sqlcv1.V1QueueItem{
			ID:                item.ID,
			TenantID:          item.TenantID,
			Queue:             item.Queue,
			TaskID:            item.TaskID,
			TaskInsertedAt:    item.TaskInsertedAt,
			ExternalID:        item.ExternalID,
			ActionID:          item.ActionID,
			StepID:            item.StepID,
			WorkflowID:        item.WorkflowID,
			WorkflowRunID:     item.WorkflowRunID,
			ScheduleTimeoutAt: item.ScheduleTimeoutAt,
			StepTimeout:       item.StepTimeout,
			Priority:          item.Priority,
			Sticky:            item.Sticky,
			DesiredWorkerID:   item.DesiredWorkerID,
			RetryCount:        item.RetryCount,
			BatchKey: pgtype.Text{
				String: item.BatchKey,
				Valid:  strings.TrimSpace(item.BatchKey) != "",
			},
		}

		schedulingTimedOut = append(schedulingTimedOut, queueItem)
	}

	// Delete the timed out items from the batched queue
	if len(idsToDelete) > 0 {
		if err := b.repo.DeleteBatchedQueueItems(b.ctx, idsToDelete); err != nil {
			return fmt.Errorf("failed to delete timed out batched queue items: %w", err)
		}
	}

	// Emit the timed out items so they can be processed downstream
	if b.emitResults != nil && len(schedulingTimedOut) > 0 {
		b.emitResults(&QueueResults{
			TenantId:           b.tenantId,
			SchedulingTimedOut: schedulingTimedOut,
		})
	}

	b.l.Info().Int("count", len(timedOutItems)).Msg("batched tasks timed out during scheduling")

	return nil
}
