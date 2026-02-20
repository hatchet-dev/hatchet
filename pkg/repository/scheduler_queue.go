package repository

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
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
	WorkerId uuid.UUID

	QueueItem *sqlcv1.V1QueueItem

	// IsAssignedLocally refers to whether the item has been assigned to a worker registered in the same
	// process as the scheduler process.
	IsAssignedLocally bool
}

type AssignResults struct {
	Assigned           []*AssignedItem
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

	succeeded, failed, err = d.markQueueItemsProcessed(ctx, d.tenantId, r, tx, false)

	if err := commit(ctx); err != nil {
		return nil, nil, err
	}

	go func() {
		// if we committed, we can update the min id
		d.updateMinId()
	}()

	return succeeded, failed, nil
}

func (d *sharedRepository) markQueueItemsProcessed(ctx context.Context, tenantId uuid.UUID, r *AssignResults, tx sqlcv1.DBTX, isOptimistic bool) (succeeded []*AssignedItem, failed []*AssignedItem, err error) {
	ctx, span := telemetry.NewSpan(ctx, "mark-queue-items-processed")
	defer span.End()

	start := time.Now()
	checkpoint := start

	idsToUnqueue := make([]int64, 0, len(r.Assigned))
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

	if !isOptimistic {
		// we don't want to waste a query if we're scheduling optimistically; this only happens on insert so there's
		// nothing to release
		_, err = d.releaseTasks(ctx, tx, tenantId, tasksToRelease)

		if err != nil {
			return nil, nil, err
		}
	}

	queuedItemsMap := make(map[int64]struct{}, len(queuedItemIds))

	for _, id := range queuedItemIds {
		queuedItemsMap[id] = struct{}{}
	}

	taskIds := make([]int64, 0, len(r.Assigned))
	tenantIds := make([]uuid.UUID, 0, len(r.Assigned))
	taskInsertedAts := make([]pgtype.Timestamptz, 0, len(r.Assigned))
	workerIds := make([]uuid.UUID, 0, len(r.Assigned))

	var minTaskInsertedAt pgtype.Timestamptz

	// if there are any idsToUnqueue that are not in the queuedItems, this means they were
	// deleted from the v1_queue_items table, so we should not assign them
	for id, assignedItem := range queueItemIdsToAssignedItem {
		if _, ok := queuedItemsMap[id]; ok {
			taskIds = append(taskIds, assignedItem.QueueItem.TaskID)
			taskInsertedAts = append(taskInsertedAts, assignedItem.QueueItem.TaskInsertedAt)
			tenantIds = append(tenantIds, tenantId)
			workerIds = append(workerIds, assignedItem.WorkerId)

			if assignedItem.QueueItem.TaskInsertedAt.Valid && (!minTaskInsertedAt.Valid || assignedItem.QueueItem.TaskInsertedAt.Time.Before(minTaskInsertedAt.Time)) {
				minTaskInsertedAt = assignedItem.QueueItem.TaskInsertedAt
			}
		}
	}

	timeAfterBulkQueueItems := time.Since(checkpoint)
	checkpoint = time.Now()

	updatedTasks, err := d.queries.UpdateTasksToAssigned(ctx, tx, sqlcv1.UpdateTasksToAssignedParams{
		Taskids:           taskIds,
		Taskinsertedats:   taskInsertedAts,
		Mintaskinsertedat: minTaskInsertedAt,
		Workerids:         workerIds,
		Tenantid:          tenantId,
	})

	if err != nil {
		return nil, nil, err
	}

	incrementInvocationCountOpts := make([]IncrementDurableTaskInvocationCountsOpts, 0)

	for _, t := range updatedTasks {
		if t.IsDurable.Valid && t.IsDurable.Bool {
			incrementInvocationCountOpts = append(incrementInvocationCountOpts, IncrementDurableTaskInvocationCountsOpts{
				TaskId:         t.TaskID,
				TaskInsertedAt: t.TaskInsertedAt,
				TenantId:       tenantId,
			})
		}
	}

	if len(incrementInvocationCountOpts) > 0 {
		_, err := d.incrementDurableTaskInvocationCounts(ctx, tx, []IncrementDurableTaskInvocationCountsOpts{})

		if err != nil {
			return nil, nil, err
		}
	}

	timeAfterUpdateStepRuns := time.Since(checkpoint)

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

func (d *queueRepository) GetTaskRateLimits(ctx context.Context, tx *OptimisticTx, queueItems []*sqlcv1.V1QueueItem) (map[int64]map[string]int32, error) {
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
		return nil, nil
	}

	// get all step run expression evals which correspond to rate limits, grouped by step run id
	expressionEvals, err := d.queries.ListTaskExpressionEvals(ctx, queryTx, sqlcv1.ListTaskExpressionEvalsParams{
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
			upsertRateLimitBulkParams.Keys = append(upsertRateLimitBulkParams.Keys, key)
			upsertRateLimitBulkParams.Windows = append(upsertRateLimitBulkParams.Windows, getWindowParamFromDurString(duration))
			upsertRateLimitBulkParams.Limitvalues = append(upsertRateLimitBulkParams.Limitvalues, int32(limitValue)) // nolint: gosec
		}
	}

	var stepRateLimits []*sqlcv1.StepRateLimit

	if len(upsertRateLimitBulkParams.Keys) > 0 {
		// upsert all rate limits based on the keys, limit values, and durations
		err = d.queries.UpsertRateLimitsBulk(ctx, queryTx, upsertRateLimitBulkParams)

		if err != nil {
			return nil, fmt.Errorf("could not bulk upsert dynamic rate limits: %w", err)
		}
	}

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
		return nil, fmt.Errorf("could not list rate limits for steps: %w", err)
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

	return taskIdToKeyToUnits, nil
}

func (d *queueRepository) GetDesiredLabels(ctx context.Context, tx *OptimisticTx, stepIds []uuid.UUID) (map[uuid.UUID][]*sqlcv1.GetDesiredLabelsRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "get-desired-labels")
	defer span.End()

	stepIdsToLookup := make([]uuid.UUID, 0, len(stepIds))
	stepIdToLabels := make(map[uuid.UUID][]*sqlcv1.GetDesiredLabelsRow)

	uniqueStepIds := sqlchelpers.UniqueSet(stepIds)

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

	uniqueStepIds := sqlchelpers.UniqueSet(stepIds)

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
