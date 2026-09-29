package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/internal/listutils"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type BatchFlushReason string

const (
	FlushReasonBatchSizeReached        BatchFlushReason = "batch_size_reached"
	FlushReasonWorkerChanged           BatchFlushReason = "worker_changed"
	FlushReasonDispatcherChanged       BatchFlushReason = "dispatcher_changed"
	FlushReasonIntervalElapsed         BatchFlushReason = "interval_elapsed"
	FlushReasonBufferDrained           BatchFlushReason = "buffer_drained"
	FlushReasonBufferMemorySizeReached BatchFlushReason = "memory_size_reached"
)

type RateLimitResult struct {
	*sqlcv1.V1QueueItem

	ExceededKey    string
	ExceededUnits  int32
	ExceededVal    int32
	NextRefillAt   *time.Time
	TaskId         int64
	TaskInsertedAt pgtype.Timestamptz
	RetryCount     int32
}

type taskIdRetryCount struct {
	taskId     int64
	retryCount int32
}

type AssignedItem struct {
	WorkerId uuid.UUID

	QueueItem *sqlcv1.V1QueueItem

	// IsAssignedLocally refers to whether the item has been assigned to a worker registered in the same
	// process as the scheduler process.
	IsAssignedLocally bool

	IsDurable bool

	Batch *BatchAssignmentMetadata
}

type BatchAssignmentMetadata struct {
	State string

	Reason BatchFlushReason

	TriggeredAt time.Time

	ConfiguredBatchMaxSize int32

	// ConfiguredBatchMaxIntervalMs is stored in milliseconds.
	ConfiguredBatchMaxIntervalMs int32

	ConfiguredBatchGroupMaxRuns int32

	Pending int32

	NextFlushAt *time.Time

	BatchID string

	StepID        string
	ActionID      string
	BatchGroupKey string
}

type AssignResults struct {
	Assigned           []*AssignedItem
	Buffered           []*AssignedItem
	Batched            []*sqlcv1.V1QueueItem
	Unassigned         []*sqlcv1.V1QueueItem
	SchedulingTimedOut []*sqlcv1.V1QueueItem
	RateLimited        []*RateLimitResult
	RateLimitedToMove  []*RateLimitResult
}

type queueFactoryRepository struct {
	*sharedRepository
}

func newQueueFactoryRepository(shared *sharedRepository) *queueFactoryRepository {
	return &queueFactoryRepository{
		sharedRepository: shared,
	}
}

func (q *queueFactoryRepository) NewQueue(tenantId uuid.UUID, queueName string) QueueRepository {
	return newQueueRepository(q.sharedRepository, tenantId, queueName)
}

type batchQueueFactoryRepository struct {
	*sharedRepository
}

func newBatchQueueFactoryRepository(shared *sharedRepository) *batchQueueFactoryRepository {
	return &batchQueueFactoryRepository{
		sharedRepository: shared,
	}
}

func (b *batchQueueFactoryRepository) NewBatchQueue(tenantId uuid.UUID) BatchQueueRepository {
	return &batchQueueRepository{
		sharedRepository: b.sharedRepository,
		tenantId:         tenantId,
	}
}

type queueRepository struct {
	*sharedRepository

	tenantId  uuid.UUID
	queueName string

	gtId   pgtype.Int8
	gtIdMu sync.RWMutex

	updateMinIdMu sync.Mutex

	cachedStepIdHasRateLimit *cache.Cache
}

func newQueueRepository(shared *sharedRepository, tenantId uuid.UUID, queueName string) *queueRepository {
	c := cache.New(5 * time.Minute)

	return &queueRepository{
		sharedRepository:         shared,
		tenantId:                 tenantId,
		queueName:                queueName,
		cachedStepIdHasRateLimit: c,
	}
}

func (d *queueRepository) Cleanup() {
	d.cachedStepIdHasRateLimit.Stop()
}

func (d *queueRepository) setMinId(id int64) {
	d.gtIdMu.Lock()
	defer d.gtIdMu.Unlock()

	d.gtId = pgtype.Int8{
		Int64: id,
		Valid: true,
	}
}

func (d *queueRepository) getMinId() pgtype.Int8 {
	d.gtIdMu.RLock()
	defer d.gtIdMu.RUnlock()

	val := d.gtId

	return val
}

func (d *queueRepository) ListQueueItems(ctx context.Context, limit int) ([]*sqlcv1.V1QueueItem, error) {
	ctx, span := telemetry.NewSpan(ctx, "list-queue-items")
	defer span.End()

	start := time.Now()
	checkpoint := start

	qis, err := d.queries.ListQueueItemsForQueue(ctx, d.pool, sqlcv1.ListQueueItemsForQueueParams{
		Tenantid: d.tenantId,
		Queue:    d.queueName,
		GtId:     d.getMinId(),
		Limit: pgtype.Int4{
			Int32: int32(limit), // nolint: gosec
			Valid: true,
		},
	})

	if err != nil {
		return nil, err
	}

	if len(qis) == 0 {
		return nil, nil
	}

	listTime := time.Since(checkpoint)
	checkpoint = time.Now()

	// TODO: REMOVE INVALID TASKS?
	// resQis, err := d.removeInvalidStepRuns(ctx, qis)

	// if err != nil {
	// 	return nil, err
	// }

	removeInvalidTime := time.Since(checkpoint)

	if sinceStart := time.Since(start); sinceStart > 100*time.Millisecond {
		d.l.Warn().Dur(
			"list", listTime,
		).Dur(
			"remove_invalid", removeInvalidTime,
		).Msgf(
			"listing %d queue items for queue %s took longer than 100ms (%s)", len(qis), d.queueName, sinceStart.String(),
		)
	}

	return qis, nil
}

func (d *queueRepository) updateMinId() {
	if !d.updateMinIdMu.TryLock() {
		return
	}
	defer d.updateMinIdMu.Unlock()

	dbCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	minId, err := d.queries.GetMinUnprocessedQueueItemId(dbCtx, d.pool, sqlcv1.GetMinUnprocessedQueueItemIdParams{
		Tenantid: d.tenantId,
		Queue:    d.queueName,
	})

	if err != nil {
		d.l.Error().Err(err).Msg("error getting min id")
		return
	}

	if minId != 0 {
		d.setMinId(minId)
	}
}

func (d *queueRepository) MarkQueueItemsProcessed(ctx context.Context, r *AssignResults) (succeeded []*AssignedItem, failed []*AssignedItem, err error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, d.pool, d.l)

	if err != nil {
		return nil, nil, err
	}

	defer rollback()

	succeeded, failed, err = d.markQueueItemsProcessed(ctx, d.tenantId, r, tx)

	if err != nil {
		return nil, nil, err
	}

	if err := commit(ctx); err != nil {
		return nil, nil, err
	}

	go func() {
		// if we committed, we can update the min id
		d.updateMinId()
	}()

	return succeeded, failed, nil
}

func (d *sharedRepository) markQueueItemsProcessed(ctx context.Context, tenantId uuid.UUID, r *AssignResults, tx sqlcv1.DBTX) (succeeded []*AssignedItem, failed []*AssignedItem, err error) {
	ctx, span := telemetry.NewSpan(ctx, "mark-queue-items-processed")
	defer span.End()

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "tenant.id", Value: tenantId.String()},
		telemetry.AttributeKV{Key: "batch.assigned", Value: len(r.Assigned)},
		telemetry.AttributeKV{Key: "batch.unassigned", Value: len(r.Unassigned)},
		telemetry.AttributeKV{Key: "batch.scheduling_timed_out", Value: len(r.SchedulingTimedOut)},
		telemetry.AttributeKV{Key: "batch.rate_limited", Value: len(r.RateLimited)},
		telemetry.AttributeKV{Key: "batch.rate_limited_to_move", Value: len(r.RateLimitedToMove)},
	)

	start := time.Now()

	succeeded = make([]*AssignedItem, 0, len(r.Assigned))
	failed = make([]*AssignedItem, 0, len(r.Assigned))

	// assign keys: removed from the queue and given a runtime row in one statement
	taskRetryToAssignedItem := make(map[taskIdRetryCount]*AssignedItem, len(r.Assigned))
	taskIds := make([]int64, 0, len(r.Assigned))
	taskInsertedAts := make([]pgtype.Timestamptz, 0, len(r.Assigned))
	taskRetryCounts := make([]int32, 0, len(r.Assigned))
	workerIds := make([]uuid.UUID, 0, len(r.Assigned))
	stepTimeouts := make([]pgtype.Interval, 0, len(r.Assigned))

	var minTaskInsertedAt pgtype.Timestamptz

	for _, assignedItem := range r.Assigned {
		qi := assignedItem.QueueItem
		key := taskIdRetryCount{taskId: qi.TaskID, retryCount: qi.RetryCount}

		// a key appears twice when a queue item read earlier was replaced (evicted and
		// restored) and the scheduler assigned both copies; the statement deletes by key
		// and the runtime upsert rejects a repeated key, so only the first copy is flushed
		// and the others are reported failed (nacked, releasing their slots)
		if _, seen := taskRetryToAssignedItem[key]; seen {
			failed = append(failed, assignedItem)
			continue
		}

		// the queue item carries the task's step_timeout, so it is parsed here instead
		// of per row in the statement; the statement adds it to CURRENT_TIMESTAMP. A
		// timeout the grammar raises on fails only its own item (nacked, so its slot is
		// released) and the rest of the batch still flushes.
		stepTimeout, err := durationToInterval(qi.StepTimeout.String)

		if err != nil {
			d.l.Warn().Err(err).Int64("task_id", qi.TaskID).Int32("retry_count", qi.RetryCount).Msg("could not parse the step timeout of an assigned queue item, reporting it failed")
			failed = append(failed, assignedItem)
			continue
		}

		taskRetryToAssignedItem[key] = assignedItem

		taskIds = append(taskIds, qi.TaskID)
		taskInsertedAts = append(taskInsertedAts, qi.TaskInsertedAt)
		taskRetryCounts = append(taskRetryCounts, qi.RetryCount)
		workerIds = append(workerIds, assignedItem.WorkerId)
		stepTimeouts = append(stepTimeouts, stepTimeout)

		if qi.TaskInsertedAt.Valid && (!minTaskInsertedAt.Valid || qi.TaskInsertedAt.Time.Before(minTaskInsertedAt.Time)) {
			minTaskInsertedAt = qi.TaskInsertedAt
		}
	}

	// remove keys: only taken out of the queue (released or buffered below)
	removeCount := len(r.SchedulingTimedOut) + len(r.Buffered)
	removeTaskIds := make([]int64, 0, removeCount)
	removeTaskInsertedAts := make([]pgtype.Timestamptz, 0, removeCount)
	removeRetryCounts := make([]int32, 0, removeCount)

	tasksToRelease := make([]TaskIdInsertedAtRetryCount, 0, len(r.SchedulingTimedOut))

	for _, id := range r.SchedulingTimedOut {
		removeTaskIds = append(removeTaskIds, id.TaskID)
		removeTaskInsertedAts = append(removeTaskInsertedAts, id.TaskInsertedAt)
		removeRetryCounts = append(removeRetryCounts, id.RetryCount)
		tasksToRelease = append(tasksToRelease, TaskIdInsertedAtRetryCount{
			Id:         id.TaskID,
			InsertedAt: id.TaskInsertedAt,
			RetryCount: id.RetryCount,
		})
	}

	bufferedItems := make([]*sqlcv1.V1QueueItem, 0, len(r.Buffered))

	for _, buffered := range r.Buffered {
		if buffered == nil || buffered.QueueItem == nil {
			continue
		}

		removeTaskIds = append(removeTaskIds, buffered.QueueItem.TaskID)
		removeTaskInsertedAts = append(removeTaskInsertedAts, buffered.QueueItem.TaskInsertedAt)
		removeRetryCounts = append(removeRetryCounts, buffered.QueueItem.RetryCount)
		bufferedItems = append(bufferedItems, buffered.QueueItem)
	}

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "batch.ids_to_unqueue", Value: len(taskIds) + len(removeTaskIds)},
		telemetry.AttributeKV{Key: "batch.tasks_to_release", Value: len(tasksToRelease)},
	)

	// FlushAssignedQueueItems locks existing runtime rows first, so this transaction's lock
	// order stays consistent with RestoreEvictedTasks; it must run before any other queue
	// item is deleted in this transaction.
	var (
		flushed       []*sqlcv1.FlushAssignedQueueItemsRow
		flushDuration time.Duration
	)

	if len(taskIds)+len(removeTaskIds) > 0 {
		flushStart := time.Now()

		flushed, err = d.queries.FlushAssignedQueueItems(ctx, tx, sqlcv1.FlushAssignedQueueItemsParams{
			Taskids:               taskIds,
			Taskinsertedats:       taskInsertedAts,
			Taskretrycounts:       taskRetryCounts,
			Workerids:             workerIds,
			Steptimeouts:          stepTimeouts,
			Removetaskids:         removeTaskIds,
			Removetaskinsertedats: removeTaskInsertedAts,
			Removeretrycounts:     removeRetryCounts,
			Tenantid:              tenantId,
			Mintaskinsertedat:     minTaskInsertedAt,
		})

		flushDuration = time.Since(flushStart)

		if err != nil {
			return nil, nil, err
		}
	}

	// a key that did not come back was deleted from v1_queue_item underneath the
	// scheduler (cancellation, another scheduler), so it is neither assigned nor buffered
	deletedKeys := make(map[taskIdRetryCount]struct{}, len(flushed))
	incrementInvocationCountOpts := make([]IncrementDurableTaskInvocationCountsOpts, 0)

	for _, row := range flushed {
		key := taskIdRetryCount{taskId: row.TaskID, retryCount: row.RetryCount}
		deletedKeys[key] = struct{}{}

		if row.WorkerID == nil {
			continue
		}

		assignedItem, ok := taskRetryToAssignedItem[key]

		if !ok {
			continue
		}

		if row.IsDurable.Valid {
			assignedItem.IsDurable = row.IsDurable.Bool

			if row.IsDurable.Bool {
				incrementInvocationCountOpts = append(incrementInvocationCountOpts, IncrementDurableTaskInvocationCountsOpts{
					TaskId:         row.TaskID,
					TaskInsertedAt: row.TaskInsertedAt,
					TenantId:       tenantId,
				})
			}
		}

		succeeded = append(succeeded, assignedItem)
		delete(taskRetryToAssignedItem, key)
	}

	for _, assignedItem := range taskRetryToAssignedItem {
		failed = append(failed, assignedItem)
	}

	// move batch queue items from v1_queue_item -> v1_batched_queue_item (replaces trigger-based redirect)
	batchedQueueItemIDs := make([]int64, 0, len(r.Batched))

	for _, batched := range r.Batched {
		if batched == nil {
			continue
		}

		batchedQueueItemIDs = append(batchedQueueItemIDs, batched.ID)
	}

	if len(batchedQueueItemIDs) > 0 {
		_, err = d.queries.MoveQueueItemsToBatchedQueue(ctx, tx, batchedQueueItemIDs)
		if err != nil {
			return nil, nil, err
		}
	}

	// remove rate limited queue items from the queue and place them in the v1_rate_limited_queue_items table
	qisToMoveToRateLimited := make([]int64, 0, len(r.RateLimited))
	qisToMoveToRateLimitedRQAfter := make([]pgtype.Timestamptz, 0, len(r.RateLimited))

	for _, row := range r.RateLimitedToMove {
		qisToMoveToRateLimited = append(qisToMoveToRateLimited, row.ID)
		qisToMoveToRateLimitedRQAfter = append(qisToMoveToRateLimitedRQAfter, sqlchelpers.TimestamptzFromTime(*row.NextRefillAt))
	}

	if len(qisToMoveToRateLimited) > 0 {
		_, err = d.queries.MoveRateLimitedQueueItems(ctx, tx, sqlcv1.MoveRateLimitedQueueItemsParams{
			Ids:          qisToMoveToRateLimited,
			Requeueafter: qisToMoveToRateLimitedRQAfter,
		})

		if err != nil {
			return nil, nil, err
		}
	}

	if len(tasksToRelease) > 0 {
		_, err = d.releaseTasks(ctx, tx, tenantId, tasksToRelease)

		if err != nil {
			return nil, nil, err
		}
	}

	validBufferedTaskIds := make([]int64, 0, len(bufferedItems))
	validBufferedInsertedAts := make([]pgtype.Timestamptz, 0, len(bufferedItems))
	validBufferedRetryCounts := make([]int32, 0, len(bufferedItems))

	for _, qi := range bufferedItems {
		if _, ok := deletedKeys[taskIdRetryCount{taskId: qi.TaskID, retryCount: qi.RetryCount}]; !ok {
			continue
		}

		validBufferedTaskIds = append(validBufferedTaskIds, qi.TaskID)
		validBufferedInsertedAts = append(validBufferedInsertedAts, qi.TaskInsertedAt)
		validBufferedRetryCounts = append(validBufferedRetryCounts, qi.RetryCount)
	}

	if len(validBufferedTaskIds) > 0 {
		err = d.queries.InsertBufferedTaskRuntimes(ctx, tx, sqlcv1.InsertBufferedTaskRuntimesParams{
			Tenantid:        tenantId,
			Taskids:         validBufferedTaskIds,
			Taskinsertedats: validBufferedInsertedAts,
			Taskretrycounts: validBufferedRetryCounts,
		})

		if err != nil {
			return nil, nil, err
		}
	}

	if len(incrementInvocationCountOpts) > 0 {
		_, err := d.incrementDurableTaskInvocationCounts(ctx, tx, incrementInvocationCountOpts)

		if err != nil {
			return nil, nil, err
		}
	}

	sinceStart := time.Since(start)

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "result.succeeded", Value: len(succeeded)},
		telemetry.AttributeKV{Key: "result.failed", Value: len(failed)},
		telemetry.AttributeKV{Key: "duration.total_ms", Value: sinceStart.Milliseconds()},
		telemetry.AttributeKV{Key: "duration.flush_ms", Value: flushDuration.Milliseconds()},
	)

	if sinceStart > 100*time.Millisecond {
		d.l.Warn().Dur(
			"duration", sinceStart,
		).Dur(
			"flush", flushDuration,
		).Int(
			"assigned", len(succeeded),
		).Int(
			"failed", len(failed),
		).Int(
			"unassigned", len(r.Unassigned),
		).Int(
			"scheduling_timed_out", len(r.SchedulingTimedOut),
		).Int(
			"rate_limited", len(r.RateLimited),
		).Int(
			"rate_limited_to_move", len(r.RateLimitedToMove),
		).Int(
			"ids_to_unqueue", len(taskIds)+len(removeTaskIds),
		).Int(
			"tasks_to_release", len(tasksToRelease),
		).Msgf(
			"marking queue items processed took longer than 100ms",
		)
	}

	return succeeded, failed, nil
}

func (d *queueRepository) GetTaskRateLimits(ctx context.Context, tx *OptimisticTx, queueItems []*sqlcv1.V1QueueItem) (map[int64]map[string]int32, map[string]RateLimitDefinition, error) {
	ctx, span := telemetry.NewSpan(ctx, "get-step-run-rate-limits")
	defer span.End()

	var queryTx sqlcv1.DBTX

	if tx != nil {
		queryTx = tx.tx
	} else {
		queryTx = d.pool
	}

	taskIds := make([]int64, 0, len(queueItems))
	taskInsertedAts := make([]pgtype.Timestamptz, 0, len(queueItems))
	stepsWithRateLimits := make(map[uuid.UUID]bool)
	stepIdToTasks := make(map[uuid.UUID][]int64)
	taskIdToStepId := make(map[int64]uuid.UUID)

	for _, item := range queueItems {
		taskIds = append(taskIds, item.TaskID)
		taskInsertedAts = append(taskInsertedAts, item.TaskInsertedAt)

		stepId := item.StepID

		stepIdToTasks[stepId] = append(stepIdToTasks[stepId], item.TaskID)
		taskIdToStepId[item.TaskID] = stepId
	}

	// check if we have any rate limits for these step ids
	skipRateLimiting := true

	for stepId := range stepIdToTasks {
		if hasRateLimit, ok := d.cachedStepIdHasRateLimit.Get(stepId.String()); !ok || hasRateLimit.(bool) {
			skipRateLimiting = false
			break
		}
	}

	if skipRateLimiting {
		return nil, nil, nil
	}

	// get all step run expression evals which correspond to rate limits, grouped by step run id
	expressionEvals, err := d.queries.ListTaskExpressionEvals(ctx, queryTx, sqlcv1.ListTaskExpressionEvalsParams{
		Taskids:         taskIds,
		Taskinsertedats: taskInsertedAts,
	})

	if err != nil {
		return nil, nil, err
	}

	taskIdAndGlobalKeyToKey := make(map[string]string)
	taskIdToKeys := make(map[int64][]string)

	for _, eval := range expressionEvals {
		taskId := eval.TaskID
		globalKey := eval.Key

		// Only append if this is a key expression. Note that we have a uniqueness constraint on
		// the stepRunId, kind, and key, so we will not insert duplicate values into the array.
		if eval.Kind == sqlcv1.StepExpressionKindDYNAMICRATELIMITKEY {
			stepsWithRateLimits[taskIdToStepId[taskId]] = true

			k := eval.ValueStr.String

			if _, ok := taskIdToKeys[taskId]; !ok {
				taskIdToKeys[taskId] = make([]string, 0)
			}

			taskIdToKeys[taskId] = append(taskIdToKeys[taskId], k)

			taskIdAndGlobalKey := fmt.Sprintf("%d-%s", taskId, globalKey)

			taskIdAndGlobalKeyToKey[taskIdAndGlobalKey] = k
		}
	}

	rateLimitKeyToEvals := make(map[string][]*sqlcv1.V1TaskExpressionEval)

	for _, eval := range expressionEvals {
		k := taskIdAndGlobalKeyToKey[fmt.Sprintf("%d-%s", eval.TaskID, eval.Key)]

		if _, ok := rateLimitKeyToEvals[k]; !ok {
			rateLimitKeyToEvals[k] = make([]*sqlcv1.V1TaskExpressionEval, 0)
		}

		rateLimitKeyToEvals[k] = append(rateLimitKeyToEvals[k], eval)
	}

	definitions := make(map[string]RateLimitDefinition)

	taskIdToKeyToUnits := make(map[int64]map[string]int32)

	for key, evals := range rateLimitKeyToEvals {
		var duration string
		var limitValue int
		var skip bool

		for _, eval := range evals {
			// add to taskIdToKeyToUnits
			taskId := eval.TaskID

			// throw an error if there are multiple rate limits with the same keys, but different limit values or durations
			if eval.Kind == sqlcv1.StepExpressionKindDYNAMICRATELIMITWINDOW {
				if duration == "" {
					duration = eval.ValueStr.String
				} else if duration != eval.ValueStr.String {
					largerDuration, err := getLargerDuration(duration, eval.ValueStr.String)

					if err != nil {
						skip = true
						break
					}

					// FIXME: this is a helpful debug log, but we aren't propagating this all the way back to OLAP yet
					// message := fmt.Sprintf("Multiple rate limits with key %s have different durations: %s vs %s. Using longer window %s.", key, duration, eval.ValueStr.String, largerDuration)
					// timeSeen := time.Now().UTC()
					// reason := sqlcv1.StepRunEventReasonRATELIMITERROR
					// severity := sqlcv1.StepRunEventSeverityWARNING
					// data := map[string]interface{}{}

					// buffErr := d.bulkEventBuffer.FireForget(d.tenantId.String(), &repository.CreateStepRunEventOpts{
					// 	StepRunId:     eval.StepRunId.String(),
					// 	EventMessage:  &message,
					// 	EventReason:   &reason,
					// 	EventSeverity: &severity,
					// 	Timestamp:     &timeSeen,
					// 	EventData:     data,
					// })

					// if buffErr != nil {
					// 	d.l.Err(buffErr).Msg("could not buffer step run event")
					// }

					duration = largerDuration
				}
			}

			if eval.Kind == sqlcv1.StepExpressionKindDYNAMICRATELIMITVALUE {
				if limitValue == 0 {
					limitValue = int(eval.ValueInt.Int32)
				} else if limitValue != int(eval.ValueInt.Int32) {
					// FIXME: this is a helpful debug log, but we aren't propagating this all the way back to OLAP yet
					// message := fmt.Sprintf("Multiple rate limits with key %s have different limit values: %d vs %d. Using lower value %d.", key, limitValue, eval.ValueInt.Int32, min(limitValue, int(eval.ValueInt.Int32)))
					// timeSeen := time.Now().UTC()
					// reason := sqlcv1.StepRunEventReasonRATELIMITERROR
					// severity := sqlcv1.StepRunEventSeverityWARNING
					// data := map[string]interface{}{}

					// buffErr := d.bulkEventBuffer.FireForget(d.tenantId.String(), &repository.CreateStepRunEventOpts{
					// 	StepRunId:     eval.StepRunId.String(),
					// 	EventMessage:  &message,
					// 	EventReason:   &reason,
					// 	EventSeverity: &severity,
					// 	Timestamp:     &timeSeen,
					// 	EventData:     data,
					// })

					// if buffErr != nil {
					// 	d.l.Err(buffErr).Msg("could not buffer step run event")
					// }

					limitValue = min(limitValue, int(eval.ValueInt.Int32))
				}
			}

			if eval.Kind == sqlcv1.StepExpressionKindDYNAMICRATELIMITUNITS {
				if _, ok := taskIdToKeyToUnits[taskId]; !ok {
					taskIdToKeyToUnits[taskId] = make(map[string]int32)
				}

				taskIdToKeyToUnits[taskId][key] = eval.ValueInt.Int32
			}
		}

		if skip {
			continue
		}

		// important: we use -1 as a sentinel value for a placeholder to indicate we don't need to upsert
		if limitValue >= 0 {
			definitions[key] = RateLimitDefinition{
				LimitValue: int32(limitValue), // nolint: gosec
				Window:     getWindowParamFromDurString(duration),
			}
		}
	}

	var stepRateLimits []*sqlcv1.StepRateLimit

	// get all existing static rate limits for steps to the mapping, mapping back from step ids to step run ids
	uniqueStepIds := make([]uuid.UUID, 0, len(stepIdToTasks))

	for stepId := range stepIdToTasks {
		uniqueStepIds = append(uniqueStepIds, stepId)
	}

	stepRateLimits, err = d.queries.ListRateLimitsForSteps(ctx, queryTx, sqlcv1.ListRateLimitsForStepsParams{
		Tenantid: d.tenantId,
		Stepids:  uniqueStepIds,
	})

	if err != nil {
		return nil, nil, fmt.Errorf("could not list rate limits for steps: %w", err)
	}

	for _, row := range stepRateLimits {
		stepsWithRateLimits[row.StepId] = true
		stepId := row.StepId
		tasks := stepIdToTasks[stepId]

		for _, taskId := range tasks {
			if _, ok := taskIdToKeyToUnits[taskId]; !ok {
				taskIdToKeyToUnits[taskId] = make(map[string]int32)
			}

			taskIdToKeyToUnits[taskId][row.RateLimitKey] = row.Units
		}
	}

	// store all step ids in the cache, so we can skip rate limiting for steps without rate limits
	for stepId := range stepIdToTasks {
		hasRateLimit := stepsWithRateLimits[stepId]
		d.cachedStepIdHasRateLimit.Set(stepId.String(), hasRateLimit)
	}

	return taskIdToKeyToUnits, definitions, nil
}

func (d *queueRepository) GetDesiredLabels(ctx context.Context, tx *OptimisticTx, stepIds []uuid.UUID) (map[uuid.UUID][]*sqlcv1.GetDesiredLabelsRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "get-desired-labels")
	defer span.End()

	stepIdsToLookup := make([]uuid.UUID, 0, len(stepIds))
	stepIdToLabels := make(map[uuid.UUID][]*sqlcv1.GetDesiredLabelsRow)

	uniqueStepIds := listutils.Uniq(stepIds)

	for _, stepId := range uniqueStepIds {
		if value, found := d.stepIdLabelsCache.Get(stepId); found {
			stepIdToLabels[stepId] = value
		} else {
			stepIdsToLookup = append(stepIdsToLookup, stepId)
		}
	}

	if len(stepIdsToLookup) == 0 {
		return stepIdToLabels, nil
	}

	var queryTx sqlcv1.DBTX

	if tx != nil {
		queryTx = tx.tx
	} else {
		queryTx = d.pool
	}

	labels, err := d.queries.GetDesiredLabels(ctx, queryTx, stepIdsToLookup)

	if err != nil {
		return nil, err
	}

	for _, label := range labels {
		stepId := label.StepId

		if _, ok := stepIdToLabels[stepId]; !ok {
			stepIdToLabels[stepId] = make([]*sqlcv1.GetDesiredLabelsRow, 0)
		}

		stepIdToLabels[stepId] = append(stepIdToLabels[stepId], label)
	}

	for stepId, labels := range stepIdToLabels {
		d.stepIdLabelsCache.Add(stepId, labels)
	}

	return stepIdToLabels, nil
}

func (d *queueRepository) GetStepSlotRequests(ctx context.Context, tx *OptimisticTx, stepIds []uuid.UUID) (map[uuid.UUID]map[string]int32, error) {
	ctx, span := telemetry.NewSpan(ctx, "get-step-slot-requests")
	defer span.End()

	uniqueStepIds := listutils.Uniq(stepIds)

	stepIdsToLookup := make([]uuid.UUID, 0, len(uniqueStepIds))
	stepIdToRequests := make(map[uuid.UUID]map[string]int32, len(uniqueStepIds))

	for _, stepId := range uniqueStepIds {
		if value, found := d.stepIdSlotRequestsCache.Get(stepId); found {
			stepIdToRequests[stepId] = value
		} else {
			stepIdsToLookup = append(stepIdsToLookup, stepId)
		}
	}

	if len(stepIdsToLookup) == 0 {
		return stepIdToRequests, nil
	}

	var queryTx sqlcv1.DBTX

	if tx != nil {
		queryTx = tx.tx
	} else {
		queryTx = d.pool
	}

	rows, err := d.queries.GetStepSlotRequests(ctx, queryTx, sqlcv1.GetStepSlotRequestsParams{
		Stepids:  stepIdsToLookup,
		Tenantid: d.tenantId,
	})
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		if _, ok := stepIdToRequests[row.StepID]; !ok {
			stepIdToRequests[row.StepID] = make(map[string]int32)
		}

		stepIdToRequests[row.StepID][row.SlotType] = row.Units
	}

	// cache empty results so we skip DB lookups for steps without explicit slot requests
	for _, stepId := range stepIdsToLookup {
		if _, ok := stepIdToRequests[stepId]; !ok {
			stepIdToRequests[stepId] = map[string]int32{}
		}

		d.stepIdSlotRequestsCache.Add(stepId, stepIdToRequests[stepId])
	}

	return stepIdToRequests, nil
}

func (d *queueRepository) GetStepBatchConfigs(ctx context.Context, tx *OptimisticTx, stepIds []uuid.UUID) (map[string]bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "get-step-batch-configs")
	defer span.End()

	uniqueStepIds := listutils.Uniq(stepIds)
	res := make(map[string]bool, len(uniqueStepIds))

	stepIdsToLookup := make([]uuid.UUID, 0, len(uniqueStepIds))

	for _, stepId := range uniqueStepIds {
		if value, found := d.stepIdHasBatchConfigCache.Get(stepId); found {
			res[stepId.String()] = value
		} else {
			res[stepId.String()] = false
			stepIdsToLookup = append(stepIdsToLookup, stepId)
		}
	}

	if len(stepIdsToLookup) == 0 {
		return res, nil
	}

	var queryTx sqlcv1.DBTX

	if tx != nil {
		queryTx = tx.tx
	} else {
		queryTx = d.pool
	}

	steps, err := d.queries.ListStepsWithBatchConfig(ctx, queryTx, stepIdsToLookup)
	if err != nil {
		return nil, err
	}

	for _, step := range steps {
		res[step.String()] = true
	}

	// cache the outcome for every looked-up step — steps without a batch config
	// cache false, so the common case also skips the DB lookup
	for _, stepId := range stepIdsToLookup {
		d.stepIdHasBatchConfigCache.Add(stepId, res[stepId.String()])
	}

	return res, nil
}

// ListWorkflowNamesByIds resolves workflow ids to names through the shared
// workflowIdNameCache, fetching only uncached ids from the database. Ids which cannot be
// resolved are absent from the result.
func (d *queueRepository) ListWorkflowNamesByIds(ctx context.Context, workflowIds []uuid.UUID) (map[uuid.UUID]string, error) {
	workflowIdToName := make(map[uuid.UUID]string, len(workflowIds))
	misses := make([]uuid.UUID, 0)

	for _, id := range workflowIds {
		if name, ok := d.workflowIdNameCache.Get(id); ok {
			workflowIdToName[id] = name
		} else {
			misses = append(misses, id)
		}
	}

	if len(misses) == 0 {
		return workflowIdToName, nil
	}

	rows, err := d.queries.ListWorkflowNamesByIds(ctx, d.pool, misses)

	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		d.workflowIdNameCache.Add(row.ID, row.Name)
		workflowIdToName[row.ID] = row.Name
	}

	return workflowIdToName, nil
}

func (d *queueRepository) RequeueRateLimitedItems(ctx context.Context, tenantId uuid.UUID, queueName string) ([]*sqlcv1.RequeueRateLimitedQueueItemsRow, error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, d.pool, d.l)

	if err != nil {
		return nil, err
	}

	defer rollback()

	rows, err := d.queries.RequeueRateLimitedQueueItems(ctx, tx, sqlcv1.RequeueRateLimitedQueueItemsParams{
		Tenantid: tenantId,
		Queue:    queueName,
	})

	if err != nil {
		return nil, err
	}

	// if we moved items in v1_queue_item, we need to update the active status of the queue, in case we've
	// been rate limited for longer than a day and the queue has gone inactive
	saveQueues, err := d.upsertQueues(ctx, tx, tenantId, []string{queueName})

	if err != nil {
		return nil, err
	}

	if err := commit(ctx); err != nil {
		return nil, err
	}

	saveQueues()

	return rows, nil
}

type batchQueueRepository struct {
	*sharedRepository
	tenantId uuid.UUID
}

func (b *batchQueueRepository) ListBatchResources(ctx context.Context) ([]*sqlcv1.ListDistinctBatchResourcesRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "list-batch-resources")
	defer span.End()

	rows, err := b.queries.ListDistinctBatchResources(ctx, b.pool, b.tenantId)
	if err != nil {
		return nil, err
	}

	return rows, nil
}

func (b *batchQueueRepository) ListBatchedQueueItems(ctx context.Context, stepId uuid.UUID, excludeIds []int64, limit int32) ([]*sqlcv1.V1BatchedQueueItem, error) {
	ctx, span := telemetry.NewSpan(ctx, "list-batched-queue-items")
	defer span.End()

	if excludeIds == nil {
		excludeIds = []int64{}
	}

	params := sqlcv1.ListBatchedQueueItemsForStepParams{
		Tenantid:   b.tenantId,
		Stepid:     stepId,
		Excludeids: excludeIds,
	}

	if limit > 0 {
		params.Limit = pgtype.Int4{
			Int32: limit,
			Valid: true,
		}
	}

	rows, err := b.queries.ListBatchedQueueItemsForStep(ctx, b.pool, params)
	if err != nil {
		return nil, err
	}

	// ListBatchedQueueItemsForStep selects from a CTE rather than the table directly (to keep the
	// exclude-id filter from being planned against the priority index -- see the query comment),
	// so sqlc can't trace it back to the V1BatchedQueueItem model on its own; the query uses
	// sqlc.embed() (aliasing the CTE to the table's own name) to embed it explicitly instead.
	items := make([]*sqlcv1.V1BatchedQueueItem, len(rows))
	for i, row := range rows {
		if row == nil {
			continue
		}
		item := row.V1BatchedQueueItem
		items[i] = &item
	}

	return items, nil
}

func (b *batchQueueRepository) DeleteBatchedQueueItems(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}

	return b.queries.DeleteBatchedQueueItems(ctx, b.pool, ids)
}

func (b *batchQueueRepository) ListExistingBatchedQueueItemIds(ctx context.Context, ids []int64) (map[int64]struct{}, error) {
	if len(ids) == 0 {
		return map[int64]struct{}{}, nil
	}

	rows, err := b.queries.ListExistingBatchedQueueItemIds(ctx, b.pool, sqlcv1.ListExistingBatchedQueueItemIdsParams{
		Tenantid: b.tenantId,
		Ids:      ids,
	})
	if err != nil {
		return nil, err
	}

	res := make(map[int64]struct{}, len(rows))
	for _, id := range rows {
		res[id] = struct{}{}
	}

	return res, nil
}

func (b *batchQueueRepository) GetBatchedQueueItemsByIds(ctx context.Context, ids []int64) ([]*sqlcv1.V1BatchedQueueItem, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	ctx, span := telemetry.NewSpan(ctx, "get-batched-queue-items-by-ids")
	defer span.End()

	return b.queries.GetBatchedQueueItemsByIds(ctx, b.pool, sqlcv1.GetBatchedQueueItemsByIdsParams{
		Tenantid: b.tenantId,
		Ids:      ids,
	})
}

func (b *batchQueueRepository) MoveBatchedQueueItems(ctx context.Context, ids []int64) ([]*sqlcv1.MoveBatchedQueueItemsRow, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	return b.queries.MoveBatchedQueueItems(ctx, b.pool, ids)
}

func (b *batchQueueRepository) CommitAssignments(ctx context.Context, assignments []*BatchAssignment) ([]*BatchAssignment, error) {
	if len(assignments) == 0 {
		return nil, nil
	}

	ctx, span := telemetry.NewSpan(ctx, "commit-batch-assignments")
	defer span.End()

	tx, err := b.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not begin transaction: %w", err)
	}
	defer func() {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			b.l.Error().Err(rollbackErr).Msg("rollback failed after commit assignments")
		}
	}()

	succeeded, err := b.commitAssignmentsTx(ctx, tx, assignments)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("could not commit batch assignment transaction: %w", err)
	}

	return succeeded, nil
}

// commitAssignmentsTx holds CommitAssignments' dedup-and-write logic, factored out so
// ReserveAndCommitBatchRun can run it inside the same transaction as the reservation itself,
// instead of opening a second, separate transaction the way CommitAssignments does on its own.
func (b *batchQueueRepository) commitAssignmentsTx(ctx context.Context, tx pgx.Tx, assignments []*BatchAssignment) ([]*BatchAssignment, error) {

	b.l.Debug().
		Int("incoming_assignment_count", len(assignments)).
		Msg("prepared batch assignments for commit")

	ids := make([]int64, 0, len(assignments))
	taskIds := make([]int64, 0, len(assignments))
	taskInsertedAts := make([]pgtype.Timestamptz, 0, len(assignments))
	taskRetryCounts := make([]int32, 0, len(assignments))
	workerIds := make([]uuid.UUID, 0, len(assignments))

	var minTaskInsertedAt pgtype.Timestamptz

	for _, assignment := range assignments {
		if assignment == nil {
			continue
		}

		ids = append(ids, assignment.BatchQueueItemID)
		taskIds = append(taskIds, assignment.TaskID)
		taskInsertedAts = append(taskInsertedAts, assignment.TaskInsertedAt)
		taskRetryCounts = append(taskRetryCounts, assignment.RetryCount)
		workerIds = append(workerIds, assignment.WorkerID)

		if assignment.TaskInsertedAt.Valid && (!minTaskInsertedAt.Valid || assignment.TaskInsertedAt.Time.Before(minTaskInsertedAt.Time)) {
			minTaskInsertedAt = assignment.TaskInsertedAt
		}
	}

	if len(ids) == 0 {
		return nil, nil
	}

	if err := b.queries.DeleteBatchedQueueItems(ctx, tx, ids); err != nil {
		return nil, fmt.Errorf("could not delete batched queue items: %w", err)
	}

	updated, err := b.queries.UpdateTasksToAssigned(ctx, tx, sqlcv1.UpdateTasksToAssignedParams{
		Taskids:           taskIds,
		Taskinsertedats:   taskInsertedAts,
		Taskretrycounts:   taskRetryCounts,
		Workerids:         workerIds,
		Mintaskinsertedat: minTaskInsertedAt,
		Tenantid:          b.tenantId,
	})
	if err != nil {
		return nil, fmt.Errorf("could not update tasks to assigned: %w", err)
	}

	updatedTaskIDs := make(map[taskIdRetryCount]struct{}, len(updated))
	for _, row := range updated {
		if row != nil {
			updatedTaskIDs[taskIdRetryCount{taskId: row.TaskID, retryCount: row.RetryCount}] = struct{}{}
		}
	}

	succeeded := make([]*BatchAssignment, 0, len(assignments))
	for _, a := range assignments {
		if a == nil {
			continue
		}
		if _, ok := updatedTaskIDs[taskIdRetryCount{taskId: a.TaskID, retryCount: a.RetryCount}]; ok {
			succeeded = append(succeeded, a)
		}
	}

	if err := b.applyBatchMetadataTx(ctx, tx, succeeded); err != nil {
		return nil, err
	}

	return succeeded, nil
}

// applyBatchMetadataTx sets batch_id/batch_size/batch_index/batch_key/worker_id using the same transaction used for
// reserving and committing the batch
func (b *batchQueueRepository) applyBatchMetadataTx(ctx context.Context, tx pgx.Tx, assignments []*BatchAssignment) error {
	groups := make(map[string][]*BatchAssignment)
	order := make([]string, 0, 1)

	for _, a := range assignments {
		if a == nil || strings.TrimSpace(a.BatchID) == "" {
			continue
		}

		if _, ok := groups[a.BatchID]; !ok {
			order = append(order, a.BatchID)
		}

		groups[a.BatchID] = append(groups[a.BatchID], a)
	}

	for _, batchID := range order {
		group := groups[batchID]

		taskIds := make([]int64, len(group))
		taskInsertedAts := make([]pgtype.Timestamptz, len(group))
		batchIndexes := make([]int32, len(group))

		for i, a := range group {
			taskIds[i] = a.TaskID
			taskInsertedAts[i] = a.TaskInsertedAt
			batchIndexes[i] = int32(i) // nolint: gosec
		}

		if err := b.queries.UpdateTaskBatchMetadata(ctx, tx, sqlcv1.UpdateTaskBatchMetadataParams{
			Batchid:         uuid.MustParse(batchID),
			Batchsize:       int32(len(group)), // nolint: gosec
			Workerid:        group[0].WorkerID,
			Batchkey:        group[0].BatchKey,
			Tenantid:        b.tenantId,
			Taskids:         taskIds,
			Taskinsertedats: taskInsertedAts,
			Batchindexes:    batchIndexes,
		}); err != nil {
			return fmt.Errorf("could not update task batch metadata: %w", err)
		}
	}

	return nil
}

func (b *batchQueueRepository) ReserveAndCommitBatchRun(
	ctx context.Context,
	tenantId, stepId uuid.UUID,
	actionId, batchKey, batchId string,
	maxRuns int,
	assignments []*BatchAssignment,
) (bool, []*BatchAssignment, error) {
	if maxRuns <= 0 || strings.TrimSpace(batchKey) == "" {
		succeeded, err := b.CommitAssignments(ctx, assignments)
		return true, succeeded, err
	}

	ctx, span := telemetry.NewSpan(ctx, "reserve-and-commit-batch-run")
	defer span.End()

	tx, err := b.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return false, nil, fmt.Errorf("could not begin transaction: %w", err)
	}
	defer func() {
		if txErr := tx.Rollback(ctx); txErr != nil && !errors.Is(txErr, pgx.ErrTxClosed) {
			b.l.Error().Err(txErr).Msg("rollback failed after reserve and commit batch run")
		}
	}()

	// Serializes concurrent reservation attempts for the same (tenant, step, batch_key) group
	// across the whole reserve-then-activate sequence.
	if advisoryLockErr := b.queries.AdvisoryLock(ctx, tx, sqlchelpers.AdvisoryLockKey(tenantId.String()+":"+stepId.String())); advisoryLockErr != nil {
		return false, nil, fmt.Errorf("could not acquire batch reservation lock: %w", advisoryLockErr)
	}

	reserved, err := b.queries.ReserveTaskBatchRun(ctx, tx, sqlcv1.ReserveTaskBatchRunParams{
		Tenantid: tenantId,
		Stepid:   stepId,
		Batchkey: batchKey,
		Actionid: actionId,
		Batchid:  uuid.MustParse(batchId),
		Maxruns:  int32(maxRuns), // nolint: gosec
	})
	if err != nil {
		return false, nil, err
	}

	if !reserved {
		if commitErr := tx.Commit(ctx); commitErr != nil {
			return false, nil, fmt.Errorf("could not commit batch reservation transaction: %w", commitErr)
		}

		return false, nil, nil
	}

	succeeded, err := b.commitAssignmentsTx(ctx, tx, assignments)
	if err != nil {
		return true, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return true, nil, fmt.Errorf("could not commit batch reservation transaction: %w", err)
	}

	return true, succeeded, nil
}

func getLargerDuration(s1, s2 string) (string, error) {
	i1, err := getDurationIndex(s1)
	if err != nil {
		return "", err
	}

	i2, err := getDurationIndex(s2)
	if err != nil {
		return "", err
	}

	if i1 > i2 {
		return s1, nil
	}

	return s2, nil
}

var durationStrings = []string{
	"SECOND",
	"MINUTE",
	"HOUR",
	"DAY",
	"WEEK",
	"MONTH",
	"YEAR",
}

func getWindowParamFromDurString(dur string) string {
	// validate duration string
	found := false

	for _, d := range durationStrings {
		if d == dur {
			found = true
			break
		}
	}

	if !found {
		return "MINUTE"
	}

	return fmt.Sprintf("1 %s", dur)
}

func getDurationIndex(s string) (int, error) {
	for i, d := range durationStrings {
		if d == s {
			return i, nil
		}
	}

	return -1, fmt.Errorf("invalid duration string: %s", s)
}
