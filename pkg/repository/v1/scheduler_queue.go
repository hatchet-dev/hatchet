package v1

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"

	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

type RateLimitResult struct {
	ExceededKey    string
	ExceededUnits  int32
	ExceededVal    int32
	TaskId         int64
	TaskInsertedAt pgtype.Timestamptz
	RetryCount     int32
}

type AssignedItem struct {
	WorkerId pgtype.UUID

	QueueItem *sqlcv1.V1QueueItem
}

type AssignResults struct {
	Assigned           []*AssignedItem
	Unassigned         []*sqlcv1.V1QueueItem
	SchedulingTimedOut []*sqlcv1.V1QueueItem
	RateLimited        []*RateLimitResult
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

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, d.pool, d.l, 5000)

	if err != nil {
		return nil, nil, err
	}

	defer rollback()

	durPrepare := time.Since(checkpoint)
	checkpoint = time.Now()

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

	taskIds := make([]int64, 0, len(r.Assigned))
	workerIds := make([]pgtype.UUID, 0, len(r.Assigned))

	// if there are any idsToUnqueue that are not in the queuedItems, this means they were
	// deleted from the v1_queue_items table, so we should not assign them
	for id, assignedItem := range queueItemIdsToAssignedItem {
		if _, ok := queuedItemsMap[id]; ok {
			taskIds = append(taskIds, assignedItem.QueueItem.TaskID)
			workerIds = append(workerIds, assignedItem.WorkerId)
		}
	}

	timeAfterBulkQueueItems := time.Since(checkpoint)
	checkpoint = time.Now()

	updatedTasks, err := d.queries.UpdateTasksToAssigned(ctx, tx, sqlcv1.UpdateTasksToAssignedParams{
		Taskids:   taskIds,
		Workerids: workerIds,
		Tenantid:  d.tenantId,
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

		upsertRateLimitBulkParams.Keys = append(upsertRateLimitBulkParams.Keys, key)
		upsertRateLimitBulkParams.Windows = append(upsertRateLimitBulkParams.Windows, getWindowParamFromDurString(duration))
		upsertRateLimitBulkParams.Limitvalues = append(upsertRateLimitBulkParams.Limitvalues, int32(limitValue)) // nolint: gosec
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
