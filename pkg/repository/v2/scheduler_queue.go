package v2

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"

	"github.com/hatchet-dev/hatchet/pkg/repository/v2/sqlcv2"
)

type RateLimitResult struct {
	ExceededKey   string
	ExceededUnits int32
	ExceededVal   int32
	TaskId        int64
}

type AssignedItem struct {
	WorkerId pgtype.UUID

	QueueItem *sqlcv2.V2QueueItem
}

type AssignResults struct {
	Assigned           []*AssignedItem
	Unassigned         []*sqlcv2.V2QueueItem
	SchedulingTimedOut []*sqlcv2.V2QueueItem
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

func (d *queueRepository) ListQueueItems(ctx context.Context, limit int) ([]*sqlcv2.V2QueueItem, error) {
	ctx, span := telemetry.NewSpan(ctx, "list-queue-items")
	defer span.End()

	start := time.Now()
	checkpoint := start

	qis, err := d.queries.ListQueueItemsForQueue(ctx, d.pool, sqlcv2.ListQueueItemsForQueueParams{
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

	minId, err := d.queries.GetMinUnprocessedQueueItemId(dbCtx, d.pool, sqlcv2.GetMinUnprocessedQueueItemIdParams{
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

	idsToUnqueue := make([]int64, len(r.Assigned))
	taskIds := make([]int64, len(r.Assigned))
	workerIds := make([]pgtype.UUID, len(r.Assigned))

	for i, assignedItem := range r.Assigned {
		idsToUnqueue[i] = assignedItem.QueueItem.ID
		taskIds[i] = assignedItem.QueueItem.TaskID
		workerIds[i] = assignedItem.WorkerId
	}

	unassignedTaskIds := make([]int64, 0, len(r.Unassigned))

	for _, id := range r.Unassigned {
		unassignedTaskIds = append(unassignedTaskIds, id.TaskID)
	}

	for _, id := range r.SchedulingTimedOut {
		idsToUnqueue = append(idsToUnqueue, id.ID)
	}

	updatedTasks, err := d.queries.UpdateTasksToAssigned(ctx, tx, sqlcv2.UpdateTasksToAssignedParams{
		Taskids:   taskIds,
		Workerids: workerIds,
		Tenantid:  d.tenantId,
	})

	if err != nil {
		return nil, nil, err
	}

	timeAfterUpdateStepRuns := time.Since(checkpoint)
	checkpoint = time.Now()

	err = d.queries.BulkQueueItems(ctx, tx, idsToUnqueue)

	if err != nil {
		return nil, nil, err
	}

	timeAfterBulkQueueItems := time.Since(checkpoint)

	if err := commit(ctx); err != nil {
		return nil, nil, err
	}

	go func() {
		// if we committed, we can update the min id
		d.updateMinId()
	}()

	taskIdToAssignedItem := make(map[int64]*AssignedItem, len(updatedTasks))

	for _, assignedItem := range r.Assigned {
		taskIdToAssignedItem[assignedItem.QueueItem.TaskID] = assignedItem
	}

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

// TODO: ADD RATE LIMITS
// func (d *queueRepository) GetStepRunRateLimits(ctx context.Context, queueItems []*sqlcv2.QueueItem) (map[string]map[string]int32, error) {
// 	ctx, span := telemetry.NewSpan(ctx, "get-step-run-rate-limits")
// 	defer span.End()

// 	stepRunIds := make([]pgtype.UUID, 0, len(queueItems))
// 	stepIds := make([]pgtype.UUID, 0, len(queueItems))
// 	stepsWithRateLimits := make(map[string]bool)

// 	for _, item := range queueItems {
// 		stepRunIds = append(stepRunIds, item.StepRunId)
// 		stepIds = append(stepIds, item.StepId)
// 	}

// 	stepIdToStepRuns := make(map[string][]string)
// 	stepRunIdToStepId := make(map[string]string)

// 	for i, stepRunId := range stepRunIds {
// 		stepId := sqlchelpers.UUIDToStr(stepIds[i])
// 		stepRunIdStr := sqlchelpers.UUIDToStr(stepRunId)

// 		if _, ok := stepIdToStepRuns[stepId]; !ok {
// 			stepIdToStepRuns[stepId] = make([]string, 0)
// 		}

// 		stepIdToStepRuns[stepId] = append(stepIdToStepRuns[stepId], stepRunIdStr)
// 		stepRunIdToStepId[stepRunIdStr] = stepId
// 	}

// 	// check if we have any rate limits for these step ids
// 	skipRateLimiting := true

// 	for stepIdStr := range stepIdToStepRuns {
// 		if hasRateLimit, ok := d.cachedStepIdHasRateLimit.Get(stepIdStr); !ok || hasRateLimit.(bool) {
// 			skipRateLimiting = false
// 			break
// 		}
// 	}

// 	if skipRateLimiting {
// 		return nil, nil
// 	}

// 	// get all step run expression evals which correspond to rate limits, grouped by step run id
// 	expressionEvals, err := d.queries.ListStepRunExpressionEvals(ctx, d.pool, stepRunIds)

// 	if err != nil {
// 		return nil, err
// 	}

// 	stepRunAndGlobalKeyToKey := make(map[string]string)
// 	stepRunToKeys := make(map[string][]string)

// 	for _, eval := range expressionEvals {
// 		stepRunId := sqlchelpers.UUIDToStr(eval.StepRunId)
// 		globalKey := eval.Key

// 		// Only append if this is a key expression. Note that we have a uniqueness constraint on
// 		// the stepRunId, kind, and key, so we will not insert duplicate values into the array.
// 		if eval.Kind == sqlcv2.StepExpressionKindDYNAMICRATELIMITKEY {
// 			stepsWithRateLimits[stepRunIdToStepId[stepRunId]] = true

// 			k := eval.ValueStr.String

// 			if _, ok := stepRunToKeys[stepRunId]; !ok {
// 				stepRunToKeys[stepRunId] = make([]string, 0)
// 			}

// 			stepRunToKeys[stepRunId] = append(stepRunToKeys[stepRunId], k)

// 			stepRunAndGlobalKey := fmt.Sprintf("%s-%s", stepRunId, globalKey)

// 			stepRunAndGlobalKeyToKey[stepRunAndGlobalKey] = k
// 		}
// 	}

// 	rateLimitKeyToEvals := make(map[string][]*sqlcv2.StepRunExpressionEval)

// 	for _, eval := range expressionEvals {
// 		k := stepRunAndGlobalKeyToKey[fmt.Sprintf("%s-%s", sqlchelpers.UUIDToStr(eval.StepRunId), eval.Key)]

// 		if _, ok := rateLimitKeyToEvals[k]; !ok {
// 			rateLimitKeyToEvals[k] = make([]*sqlcv2.StepRunExpressionEval, 0)
// 		}

// 		rateLimitKeyToEvals[k] = append(rateLimitKeyToEvals[k], eval)
// 	}

// 	upsertRateLimitBulkParams := sqlcv2.UpsertRateLimitsBulkParams{
// 		Tenantid: d.tenantId,
// 	}

// 	stepRunToKeyToUnits := make(map[string]map[string]int32)

// 	for key, evals := range rateLimitKeyToEvals {
// 		var duration string
// 		var limitValue int
// 		var skip bool

// 		for _, eval := range evals {
// 			// add to stepRunToKeyToUnits
// 			stepRunId := sqlchelpers.UUIDToStr(eval.StepRunId)

// 			// throw an error if there are multiple rate limits with the same keys, but different limit values or durations
// 			if eval.Kind == sqlcv2.StepExpressionKindDYNAMICRATELIMITWINDOW {
// 				if duration == "" {
// 					duration = eval.ValueStr.String
// 				} else if duration != eval.ValueStr.String {
// 					largerDuration, err := getLargerDuration(duration, eval.ValueStr.String)

// 					if err != nil {
// 						skip = true
// 						break
// 					}

// 					message := fmt.Sprintf("Multiple rate limits with key %s have different durations: %s vs %s. Using longer window %s.", key, duration, eval.ValueStr.String, largerDuration)
// 					timeSeen := time.Now().UTC()
// 					reason := sqlcv2.StepRunEventReasonRATELIMITERROR
// 					severity := sqlcv2.StepRunEventSeverityWARNING
// 					data := map[string]interface{}{}

// 					buffErr := d.bulkEventBuffer.FireForget(sqlchelpers.UUIDToStr(d.tenantId), &repository.CreateStepRunEventOpts{
// 						StepRunId:     sqlchelpers.UUIDToStr(eval.StepRunId),
// 						EventMessage:  &message,
// 						EventReason:   &reason,
// 						EventSeverity: &severity,
// 						Timestamp:     &timeSeen,
// 						EventData:     data,
// 					})

// 					if buffErr != nil {
// 						d.l.Err(buffErr).Msg("could not buffer step run event")
// 					}

// 					duration = largerDuration
// 				}
// 			}

// 			if eval.Kind == sqlcv2.StepExpressionKindDYNAMICRATELIMITVALUE {
// 				if limitValue == 0 {
// 					limitValue = int(eval.ValueInt.Int32)
// 				} else if limitValue != int(eval.ValueInt.Int32) {
// 					message := fmt.Sprintf("Multiple rate limits with key %s have different limit values: %d vs %d. Using lower value %d.", key, limitValue, eval.ValueInt.Int32, min(limitValue, int(eval.ValueInt.Int32)))
// 					timeSeen := time.Now().UTC()
// 					reason := sqlcv2.StepRunEventReasonRATELIMITERROR
// 					severity := sqlcv2.StepRunEventSeverityWARNING
// 					data := map[string]interface{}{}

// 					buffErr := d.bulkEventBuffer.FireForget(sqlchelpers.UUIDToStr(d.tenantId), &repository.CreateStepRunEventOpts{
// 						StepRunId:     sqlchelpers.UUIDToStr(eval.StepRunId),
// 						EventMessage:  &message,
// 						EventReason:   &reason,
// 						EventSeverity: &severity,
// 						Timestamp:     &timeSeen,
// 						EventData:     data,
// 					})

// 					if buffErr != nil {
// 						d.l.Err(buffErr).Msg("could not buffer step run event")
// 					}

// 					limitValue = min(limitValue, int(eval.ValueInt.Int32))
// 				}
// 			}

// 			if eval.Kind == sqlcv2.StepExpressionKindDYNAMICRATELIMITUNITS {
// 				if _, ok := stepRunToKeyToUnits[stepRunId]; !ok {
// 					stepRunToKeyToUnits[stepRunId] = make(map[string]int32)
// 				}

// 				stepRunToKeyToUnits[stepRunId][key] = eval.ValueInt.Int32
// 			}
// 		}

// 		if skip {
// 			continue
// 		}

// 		upsertRateLimitBulkParams.Keys = append(upsertRateLimitBulkParams.Keys, key)
// 		upsertRateLimitBulkParams.Windows = append(upsertRateLimitBulkParams.Windows, getWindowParamFromDurString(duration))
// 		upsertRateLimitBulkParams.Limitvalues = append(upsertRateLimitBulkParams.Limitvalues, int32(limitValue)) // nolint: gosec
// 	}

// 	var stepRateLimits []*sqlcv2.StepRateLimit

// 	if len(upsertRateLimitBulkParams.Keys) > 0 {
// 		// upsert all rate limits based on the keys, limit values, and durations
// 		err = d.queries.UpsertRateLimitsBulk(ctx, d.pool, upsertRateLimitBulkParams)

// 		if err != nil {
// 			return nil, fmt.Errorf("could not bulk upsert dynamic rate limits: %w", err)
// 		}
// 	}

// 	// get all existing static rate limits for steps to the mapping, mapping back from step ids to step run ids
// 	uniqueStepIds := make([]pgtype.UUID, 0, len(stepIdToStepRuns))

// 	for stepId := range stepIdToStepRuns {
// 		uniqueStepIds = append(uniqueStepIds, sqlchelpers.UUIDFromStr(stepId))
// 	}

// 	stepRateLimits, err = d.queries.ListRateLimitsForSteps(ctx, d.pool, sqlcv2.ListRateLimitsForStepsParams{
// 		Tenantid: d.tenantId,
// 		Stepids:  uniqueStepIds,
// 	})

// 	if err != nil {
// 		return nil, fmt.Errorf("could not list rate limits for steps: %w", err)
// 	}

// 	for _, row := range stepRateLimits {
// 		stepsWithRateLimits[sqlchelpers.UUIDToStr(row.StepId)] = true
// 		stepId := sqlchelpers.UUIDToStr(row.StepId)
// 		stepRuns := stepIdToStepRuns[stepId]

// 		for _, stepRunId := range stepRuns {
// 			if _, ok := stepRunToKeyToUnits[stepRunId]; !ok {
// 				stepRunToKeyToUnits[stepRunId] = make(map[string]int32)
// 			}

// 			stepRunToKeyToUnits[stepRunId][row.RateLimitKey] = row.Units
// 		}
// 	}

// 	// store all step ids in the cache, so we can skip rate limiting for steps without rate limits
// 	for stepId := range stepIdToStepRuns {
// 		hasRateLimit := stepsWithRateLimits[stepId]
// 		d.cachedStepIdHasRateLimit.Set(stepId, hasRateLimit)
// 	}

// 	return stepRunToKeyToUnits, nil
// }

func (d *queueRepository) GetDesiredLabels(ctx context.Context, stepIds []pgtype.UUID) (map[string][]*sqlcv2.GetDesiredLabelsRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "get-desired-labels")
	defer span.End()

	uniqueStepIds := sqlchelpers.UniqueSet(stepIds)

	labels, err := d.queries.GetDesiredLabels(ctx, d.pool, uniqueStepIds)

	if err != nil {
		return nil, err
	}

	stepIdToLabels := make(map[string][]*sqlcv2.GetDesiredLabelsRow)

	for _, label := range labels {
		stepId := sqlchelpers.UUIDToStr(label.StepId)

		if _, ok := stepIdToLabels[stepId]; !ok {
			stepIdToLabels[stepId] = make([]*sqlcv2.GetDesiredLabelsRow, 0)
		}

		stepIdToLabels[stepId] = append(stepIdToLabels[stepId], label)
	}

	return stepIdToLabels, nil
}
