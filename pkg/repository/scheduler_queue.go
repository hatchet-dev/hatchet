package repository

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
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

const rateLimitedRequeueAfterThreshold = 2 * time.Second

type AssignedItem struct {
	WorkerId pgtype.UUID

	QueueItem *sqlcv1.V1QueueItem

	Batch *BatchAssignmentMetadata
}

type BatchAssignmentMetadata struct {
	State string

	Reason string

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

func (q *queueFactoryRepository) NewQueue(tenantId pgtype.UUID, queueName string) QueueRepository {
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

func (b *batchQueueFactoryRepository) NewBatchQueue(tenantId pgtype.UUID) BatchQueueRepository {
	return &batchQueueRepository{
		sharedRepository: b.sharedRepository,
		tenantId:         tenantId,
	}
}

type queueRepository struct {
	*sharedRepository

	tenantId  pgtype.UUID
	queueName string

	gtId   pgtype.Int8
	gtIdMu sync.RWMutex

	updateMinIdMu sync.Mutex

	cachedStepIdHasRateLimit *cache.Cache
}

func newQueueRepository(shared *sharedRepository, tenantId pgtype.UUID, queueName string) *queueRepository {
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
	ctx, span := telemetry.NewSpan(ctx, "mark-queue-items-processed")
	defer span.End()

	start := time.Now()
	checkpoint := start

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, d.pool, d.l)

	if err != nil {
		return nil, nil, err
	}

	defer rollback()

	durPrepare := time.Since(checkpoint)
	checkpoint = time.Now()

	idsToUnqueue := make([]int64, 0, len(r.Assigned)+len(r.Buffered))
	queueItemIdsToAssignedItem := make(map[int64]*AssignedItem, len(r.Assigned))
	taskIdToAssignedItem := make(map[int64]*AssignedItem, len(r.Assigned))

	for _, assignedItem := range r.Assigned {
		idsToUnqueue = append(idsToUnqueue, assignedItem.QueueItem.ID)
		queueItemIdsToAssignedItem[assignedItem.QueueItem.ID] = assignedItem
		taskIdToAssignedItem[assignedItem.QueueItem.TaskID] = assignedItem
	}

	tasksToRelease := make([]TaskIdInsertedAtRetryCount, 0, len(r.SchedulingTimedOut))

	for _, id := range r.SchedulingTimedOut {
		idsToUnqueue = append(idsToUnqueue, id.ID)
		tasksToRelease = append(tasksToRelease, TaskIdInsertedAtRetryCount{
			Id:         id.TaskID,
			InsertedAt: id.TaskInsertedAt,
			RetryCount: id.RetryCount,
		})
	}

	bufferedQueueItemIDs := make([]int64, 0, len(r.Buffered))
	bufferedTaskIds := make([]int64, 0, len(r.Buffered))
	bufferedTaskInsertedAts := make([]pgtype.Timestamptz, 0, len(r.Buffered))
	bufferedRetryCounts := make([]int32, 0, len(r.Buffered))

	for _, buffered := range r.Buffered {
		if buffered == nil || buffered.QueueItem == nil {
			continue
		}

		idsToUnqueue = append(idsToUnqueue, buffered.QueueItem.ID)
		bufferedQueueItemIDs = append(bufferedQueueItemIDs, buffered.QueueItem.ID)
		bufferedTaskIds = append(bufferedTaskIds, buffered.QueueItem.TaskID)
		bufferedTaskInsertedAts = append(bufferedTaskInsertedAts, buffered.QueueItem.TaskInsertedAt)
		bufferedRetryCounts = append(bufferedRetryCounts, buffered.QueueItem.RetryCount)
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

	queuedItemIds, err := d.queries.BulkQueueItems(ctx, tx, idsToUnqueue)

	if err != nil {
		return nil, nil, err
	}

	_, err = d.releaseTasks(ctx, tx, sqlchelpers.UUIDToStr(d.tenantId), tasksToRelease)

	if err != nil {
		return nil, nil, err
	}

	queuedItemsMap := make(map[int64]struct{}, len(queuedItemIds))

	for _, id := range queuedItemIds {
		queuedItemsMap[id] = struct{}{}
	}

	validBufferedTaskIds := make([]int64, 0, len(bufferedTaskIds))
	validBufferedInsertedAts := make([]pgtype.Timestamptz, 0, len(bufferedTaskInsertedAts))
	validBufferedRetryCounts := make([]int32, 0, len(bufferedRetryCounts))

	for idx, queueItemID := range bufferedQueueItemIDs {
		if _, ok := queuedItemsMap[queueItemID]; !ok {
			continue
		}

		if idx >= len(bufferedTaskIds) || idx >= len(bufferedTaskInsertedAts) || idx >= len(bufferedRetryCounts) {
			continue
		}

		validBufferedTaskIds = append(validBufferedTaskIds, bufferedTaskIds[idx])
		validBufferedInsertedAts = append(validBufferedInsertedAts, bufferedTaskInsertedAts[idx])
		validBufferedRetryCounts = append(validBufferedRetryCounts, bufferedRetryCounts[idx])
	}

	taskIds := make([]int64, 0, len(r.Assigned))
	taskInsertedAts := make([]pgtype.Timestamptz, 0, len(r.Assigned))
	workerIds := make([]pgtype.UUID, 0, len(r.Assigned))

	var minTaskInsertedAt pgtype.Timestamptz

	// if there are any idsToUnqueue that are not in the queuedItems, this means they were
	// deleted from the v1_queue_items table, so we should not assign them
	for id, assignedItem := range queueItemIdsToAssignedItem {
		if _, ok := queuedItemsMap[id]; ok {
			taskIds = append(taskIds, assignedItem.QueueItem.TaskID)
			taskInsertedAts = append(taskInsertedAts, assignedItem.QueueItem.TaskInsertedAt)
			workerIds = append(workerIds, assignedItem.WorkerId)

			if assignedItem.QueueItem.TaskInsertedAt.Valid && (!minTaskInsertedAt.Valid || assignedItem.QueueItem.TaskInsertedAt.Time.Before(minTaskInsertedAt.Time)) {
				minTaskInsertedAt = assignedItem.QueueItem.TaskInsertedAt
			}
		}
	}

	timeAfterBulkQueueItems := time.Since(checkpoint)
	checkpoint = time.Now()

	if len(validBufferedTaskIds) > 0 {
		err = d.queries.InsertBufferedTaskRuntimes(ctx, tx, sqlcv1.InsertBufferedTaskRuntimesParams{
			Tenantid:        d.tenantId,
			Taskids:         validBufferedTaskIds,
			Taskinsertedats: validBufferedInsertedAts,
			Taskretrycounts: validBufferedRetryCounts,
		})

		if err != nil {
			return nil, nil, err
		}

		validBufferedTaskIds = nil
		validBufferedInsertedAts = nil
		validBufferedRetryCounts = nil
	}

	updatedTasks, err := d.queries.UpdateTasksToAssigned(ctx, tx, sqlcv1.UpdateTasksToAssignedParams{
		Taskids:           taskIds,
		Taskinsertedats:   taskInsertedAts,
		Mintaskinsertedat: minTaskInsertedAt,
		Workerids:         workerIds,
		Tenantid:          d.tenantId,
	})

	if err != nil {
		return nil, nil, err
	}

	timeAfterUpdateStepRuns := time.Since(checkpoint)

	if err := commit(ctx); err != nil {
		return nil, nil, err
	}

	go func() {
		// if we committed, we can update the min id
		d.updateMinId()
	}()

	succeeded = make([]*AssignedItem, 0, len(r.Assigned))
	failed = make([]*AssignedItem, 0, len(r.Assigned))

	for _, row := range updatedTasks {
		if assignedItem, ok := taskIdToAssignedItem[row.TaskID]; ok {
			succeeded = append(succeeded, assignedItem)
			delete(taskIdToAssignedItem, row.TaskID)
		}
	}

	for _, assignedItem := range taskIdToAssignedItem {
		failed = append(failed, assignedItem)
	}

	if sinceStart := time.Since(start); sinceStart > 100*time.Millisecond {
		d.l.Warn().Dur(
			"duration", sinceStart,
		).Dur(
			"prepare", durPrepare,
		).Dur(
			"update", timeAfterUpdateStepRuns,
		).Dur(
			"bulkqueue", timeAfterBulkQueueItems,
		).Int(
			"assigned", len(succeeded),
		).Int(
			"failed", len(failed),
		).Msgf(
			"marking queue items processed took longer than 100ms",
		)
	}

	return succeeded, failed, nil
}

func (d *queueRepository) GetTaskRateLimits(ctx context.Context, queueItems []*sqlcv1.V1QueueItem) (map[int64]map[string]int32, error) {
	ctx, span := telemetry.NewSpan(ctx, "get-step-run-rate-limits")
	defer span.End()

	taskIds := make([]int64, 0, len(queueItems))
	taskInsertedAts := make([]pgtype.Timestamptz, 0, len(queueItems))
	stepsWithRateLimits := make(map[string]bool)
	stepIdToTasks := make(map[string][]int64)
	taskIdToStepId := make(map[int64]string)

	for _, item := range queueItems {
		taskIds = append(taskIds, item.TaskID)
		taskInsertedAts = append(taskInsertedAts, item.TaskInsertedAt)

		stepId := sqlchelpers.UUIDToStr(item.StepID)

		stepIdToTasks[stepId] = append(stepIdToTasks[stepId], item.TaskID)
		taskIdToStepId[item.TaskID] = stepId
	}

	// check if we have any rate limits for these step ids
	skipRateLimiting := true

	for stepIdStr := range stepIdToTasks {
		if hasRateLimit, ok := d.cachedStepIdHasRateLimit.Get(stepIdStr); !ok || hasRateLimit.(bool) {
			skipRateLimiting = false
			break
		}
	}

	if skipRateLimiting {
		return nil, nil
	}

	// get all step run expression evals which correspond to rate limits, grouped by step run id
	expressionEvals, err := d.queries.ListTaskExpressionEvals(ctx, d.pool, sqlcv1.ListTaskExpressionEvalsParams{
		Taskids:         taskIds,
		Taskinsertedats: taskInsertedAts,
	})

	if err != nil {
		return nil, err
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

	upsertRateLimitBulkParams := sqlcv1.UpsertRateLimitsBulkParams{
		Tenantid: d.tenantId,
	}

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

					// buffErr := d.bulkEventBuffer.FireForget(sqlchelpers.UUIDToStr(d.tenantId), &repository.CreateStepRunEventOpts{
					// 	StepRunId:     sqlchelpers.UUIDToStr(eval.StepRunId),
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

					// buffErr := d.bulkEventBuffer.FireForget(sqlchelpers.UUIDToStr(d.tenantId), &repository.CreateStepRunEventOpts{
					// 	StepRunId:     sqlchelpers.UUIDToStr(eval.StepRunId),
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
			upsertRateLimitBulkParams.Keys = append(upsertRateLimitBulkParams.Keys, key)
			upsertRateLimitBulkParams.Windows = append(upsertRateLimitBulkParams.Windows, getWindowParamFromDurString(duration))
			upsertRateLimitBulkParams.Limitvalues = append(upsertRateLimitBulkParams.Limitvalues, int32(limitValue)) // nolint: gosec
		}
	}

	var stepRateLimits []*sqlcv1.StepRateLimit

	if len(upsertRateLimitBulkParams.Keys) > 0 {
		// upsert all rate limits based on the keys, limit values, and durations
		err = d.queries.UpsertRateLimitsBulk(ctx, d.pool, upsertRateLimitBulkParams)

		if err != nil {
			return nil, fmt.Errorf("could not bulk upsert dynamic rate limits: %w", err)
		}
	}

	// get all existing static rate limits for steps to the mapping, mapping back from step ids to step run ids
	uniqueStepIds := make([]pgtype.UUID, 0, len(stepIdToTasks))

	for stepId := range stepIdToTasks {
		uniqueStepIds = append(uniqueStepIds, sqlchelpers.UUIDFromStr(stepId))
	}

	stepRateLimits, err = d.queries.ListRateLimitsForSteps(ctx, d.pool, sqlcv1.ListRateLimitsForStepsParams{
		Tenantid: d.tenantId,
		Stepids:  uniqueStepIds,
	})

	if err != nil {
		return nil, fmt.Errorf("could not list rate limits for steps: %w", err)
	}

	for _, row := range stepRateLimits {
		stepsWithRateLimits[sqlchelpers.UUIDToStr(row.StepId)] = true
		stepId := sqlchelpers.UUIDToStr(row.StepId)
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
		d.cachedStepIdHasRateLimit.Set(stepId, hasRateLimit)
	}

	return taskIdToKeyToUnits, nil
}

func (d *queueRepository) GetDesiredLabels(ctx context.Context, stepIds []pgtype.UUID) (map[string][]*sqlcv1.GetDesiredLabelsRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "get-desired-labels")
	defer span.End()

	uniqueStepIds := sqlchelpers.UniqueSet(stepIds)

	labels, err := d.queries.GetDesiredLabels(ctx, d.pool, uniqueStepIds)

	if err != nil {
		return nil, err
	}

	stepIdToLabels := make(map[string][]*sqlcv1.GetDesiredLabelsRow)

	for _, label := range labels {
		stepId := sqlchelpers.UUIDToStr(label.StepId)

		if _, ok := stepIdToLabels[stepId]; !ok {
			stepIdToLabels[stepId] = make([]*sqlcv1.GetDesiredLabelsRow, 0)
		}

		stepIdToLabels[stepId] = append(stepIdToLabels[stepId], label)
	}

	return stepIdToLabels, nil
}

func (d *queueRepository) GetStepBatchConfigs(ctx context.Context, stepIds []pgtype.UUID) (map[string]bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "get-step-batch-configs")
	defer span.End()

	uniqueStepIds := sqlchelpers.UniqueSet(stepIds)
	res := make(map[string]bool, len(uniqueStepIds))

	for _, stepID := range uniqueStepIds {
		res[sqlchelpers.UUIDToStr(stepID)] = false
	}

	steps, err := d.queries.ListStepsWithBatchConfig(ctx, d.pool, uniqueStepIds)
	if err != nil {
		return nil, err
	}

	for _, step := range steps {
		res[sqlchelpers.UUIDToStr(step)] = true
	}

	return res, nil
}

func (d *queueRepository) RequeueRateLimitedItems(ctx context.Context, tenantId pgtype.UUID, queueName string) ([]*sqlcv1.RequeueRateLimitedQueueItemsRow, error) {
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
	saveQueues, err := d.upsertQueues(ctx, tx, sqlchelpers.UUIDToStr(tenantId), []string{queueName})

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
	tenantId pgtype.UUID
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

func (b *batchQueueRepository) ListBatchedQueueItems(ctx context.Context, stepId pgtype.UUID, batchKey string, afterId pgtype.Int8, limit int32) ([]*sqlcv1.V1BatchedQueueItem, error) {
	ctx, span := telemetry.NewSpan(ctx, "list-batched-queue-items")
	defer span.End()

	params := sqlcv1.ListBatchedQueueItemsForBatchParams{
		Tenantid: b.tenantId,
		Stepid:   stepId,
		Batchkey: batchKey,
		AfterId:  afterId,
	}

	if limit > 0 {
		params.Limit = pgtype.Int4{
			Int32: limit,
			Valid: true,
		}
	}

	return b.queries.ListBatchedQueueItemsForBatch(ctx, b.pool, params)
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

	// Deduplicate assignments by (task_id, task_inserted_at) to avoid
	// "cannot affect row a second time" errors in UpdateTasksToAssigned.
	// UpdateTasksToAssigned joins on (id, inserted_at) and uses the DB-stored
	// retry_count, so multiple assignments for the same (task_id, inserted_at)
	// in a single call will target the same v1_task_runtime row.

	// FIXME: It is not clear why we're ending up in this state, but we should investigate why and fix it.
	type taskKey struct {
		taskId     int64
		insertedAt time.Time
	}

	seenTasks := make(map[taskKey]bool)
	deduplicatedAssignments := make([]*BatchAssignment, 0, len(assignments))

	for _, assignment := range assignments {
		if assignment == nil {
			continue
		}

		key := taskKey{
			taskId:     assignment.TaskID,
			insertedAt: assignment.TaskInsertedAt.Time,
		}

		if seenTasks[key] {
			b.l.Warn().
				Int64("task_id", assignment.TaskID).
				Int32("retry_count", assignment.RetryCount).
				Str("step_id", sqlchelpers.UUIDToStr(assignment.StepID)).
				Str("action_id", assignment.ActionID).
				Str("batch_key", assignment.BatchKey).
				Str("batch_id", assignment.BatchID).
				Msg("skipping duplicate task in batch assignments")
			continue
		}

		seenTasks[key] = true
		deduplicatedAssignments = append(deduplicatedAssignments, assignment)
	}

	b.l.Debug().
		Int("incoming_assignment_count", len(assignments)).
		Int("deduped_assignment_count", len(deduplicatedAssignments)).
		Msg("prepared batch assignments for commit")

	ids := make([]int64, 0, len(deduplicatedAssignments))
	taskIds := make([]int64, 0, len(deduplicatedAssignments))
	taskInsertedAts := make([]pgtype.Timestamptz, 0, len(deduplicatedAssignments))
	workerIds := make([]pgtype.UUID, 0, len(deduplicatedAssignments))

	var minTaskInsertedAt pgtype.Timestamptz

	for _, assignment := range deduplicatedAssignments {
		if assignment == nil {
			continue
		}

		ids = append(ids, assignment.BatchQueueItemID)
		taskIds = append(taskIds, assignment.TaskID)
		taskInsertedAts = append(taskInsertedAts, assignment.TaskInsertedAt)
		workerIds = append(workerIds, assignment.WorkerID)

		if assignment.TaskInsertedAt.Valid && (!minTaskInsertedAt.Valid || assignment.TaskInsertedAt.Time.Before(minTaskInsertedAt.Time)) {
			minTaskInsertedAt = assignment.TaskInsertedAt
		}
	}

	if len(ids) == 0 {
		return nil, nil
	}

	tx, err := b.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			b.l.Error().Err(err).Msg("rollback failed after commit assignments")
		}
	}()

	if err := b.queries.DeleteBatchedQueueItems(ctx, tx, ids); err != nil {
		return nil, fmt.Errorf("could not delete batched queue items: %w", err)
	}

	updated, err := b.queries.UpdateTasksToAssigned(ctx, tx, sqlcv1.UpdateTasksToAssignedParams{
		Taskids:           taskIds,
		Taskinsertedats:   taskInsertedAts,
		Workerids:         workerIds,
		Mintaskinsertedat: minTaskInsertedAt,
		Tenantid:          b.tenantId,
	})
	if err != nil {
		return nil, fmt.Errorf("could not update tasks to assigned: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("could not commit batch assignment transaction: %w", err)
	}

	updatedTaskIDs := make(map[int64]struct{}, len(updated))
	for _, row := range updated {
		if row != nil {
			updatedTaskIDs[row.TaskID] = struct{}{}
		}
	}

	succeeded := make([]*BatchAssignment, 0, len(deduplicatedAssignments))
	for _, a := range deduplicatedAssignments {
		if a == nil {
			continue
		}
		if _, ok := updatedTaskIDs[a.TaskID]; ok {
			succeeded = append(succeeded, a)
		}
	}

	return succeeded, nil
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
