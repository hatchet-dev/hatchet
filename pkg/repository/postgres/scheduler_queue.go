package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

type queueFactoryRepository struct {
	*sharedRepository
}

func newQueueFactoryRepository(shared *sharedRepository) *queueFactoryRepository {
	return &queueFactoryRepository{
		sharedRepository: shared,
	}
}

func (q *queueFactoryRepository) NewQueue(tenantId pgtype.UUID, queueName string) repository.QueueRepository {
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

func (d *queueRepository) ListQueueItems(ctx context.Context, limit int) ([]*dbsqlc.QueueItem, error) {
	ctx, span := telemetry.NewSpan(ctx, "list-queue-items")
	defer span.End()

	start := time.Now()
	checkpoint := start

	qis, err := d.queries.ListQueueItemsForQueue(ctx, d.pool, dbsqlc.ListQueueItemsForQueueParams{
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

	resQis, err := d.removeInvalidStepRuns(ctx, qis)

	if err != nil {
		return nil, err
	}

	removeInvalidTime := time.Since(checkpoint)

	if sinceStart := time.Since(start); sinceStart > 100*time.Millisecond {
		d.l.Warn().Dur(
			"list", listTime,
		).Dur(
			"remove_invalid", removeInvalidTime,
		).Msgf(
			"listing %d queue items for queue %s took longer than 100ms (%s)", len(resQis), d.queueName, sinceStart.String(),
		)
	}

	return resQis, nil
}

// removeInvalidStepRuns removes all duplicate step runs and step runs which are in a finalized state from
// the queue. It returns the remaining queue items and an error if one occurred.
func (s *queueRepository) removeInvalidStepRuns(ctx context.Context, qis []*dbsqlc.ListQueueItemsForQueueRow) ([]*dbsqlc.QueueItem, error) {
	if len(qis) == 0 {
		return nil, nil
	}

	// remove duplicates
	encountered := map[string]bool{}
	remaining1 := make([]*dbsqlc.QueueItem, 0, len(qis))
	cancelled := make([]int64, 0, len(qis))

	for _, v := range qis {
		stepRunId := sqlchelpers.UUIDToStr(v.QueueItem.StepRunId)

		if encountered[stepRunId] {
			cancelled = append(cancelled, v.QueueItem.ID)
			continue
		}

		encountered[stepRunId] = true
		remaining1 = append(remaining1, &v.QueueItem)
	}

	finalizedStepRunsMap := make(map[string]bool)

	for _, v := range qis {
		if v.Status == dbsqlc.StepRunStatusCANCELLED || v.Status == dbsqlc.StepRunStatusSUCCEEDED || v.Status == dbsqlc.StepRunStatusFAILED || v.Status == dbsqlc.StepRunStatusCANCELLING {
			stepRunId := sqlchelpers.UUIDToStr(v.QueueItem.StepRunId)
			s.l.Warn().Msgf("step run %s is in state %s, skipping queueing", stepRunId, string(v.Status))
			finalizedStepRunsMap[stepRunId] = true
		}
	}

	// remove cancelled step runs from the queue items
	remaining2 := make([]*dbsqlc.QueueItem, 0, len(remaining1))

	for _, qi := range remaining1 {
		if _, ok := finalizedStepRunsMap[sqlchelpers.UUIDToStr(qi.StepRunId)]; ok {
			cancelled = append(cancelled, qi.ID)
			continue
		}

		remaining2 = append(remaining2, qi)
	}

	if len(cancelled) == 0 {
		return remaining2, nil
	}

	// If we've reached this point, we have queue items to cancel. We prepare a transaction in order
	// to set a statement timeout.
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, s.pool, s.l, 5000)

	if err != nil {
		return nil, err
	}

	defer rollback()

	err = s.queries.BulkQueueItems(ctx, tx, cancelled)

	if err != nil {
		return nil, err
	}

	if err := commit(ctx); err != nil {
		return nil, err
	}

	return remaining2, nil
}

func (s *queueRepository) bulkStepRunsAssigned(
	tenantId string,
	assignedAt time.Time,
	stepRunIds []pgtype.UUID,
	workerIds []pgtype.UUID,
) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	workerIdToStepRunIds := make(map[string][]string)

	for i := range stepRunIds {
		workerId := sqlchelpers.UUIDToStr(workerIds[i])

		if _, ok := workerIdToStepRunIds[workerId]; !ok {
			workerIdToStepRunIds[workerId] = make([]string, 0)
		}

		workerIdToStepRunIds[workerId] = append(workerIdToStepRunIds[workerId], sqlchelpers.UUIDToStr(stepRunIds[i]))
		message := fmt.Sprintf("Assigned to worker %s", workerId)
		timeSeen := assignedAt
		reasons := dbsqlc.StepRunEventReasonASSIGNED
		severity := dbsqlc.StepRunEventSeverityINFO
		data := map[string]interface{}{"worker_id": workerId}

		err := s.bulkEventBuffer.FireForget(tenantId, &repository.CreateStepRunEventOpts{
			StepRunId:     sqlchelpers.UUIDToStr(stepRunIds[i]),
			EventMessage:  &message,
			EventReason:   &reasons,
			EventSeverity: &severity,
			Timestamp:     &timeSeen,
			EventData:     data,
		})

		if err != nil {
			s.l.Err(err).Msg("could not buffer step run event")
		}
	}

	orderedWorkerIds := make([]pgtype.UUID, 0)
	assignedStepRuns := make([][]byte, 0)

	for workerId, stepRunIds := range workerIdToStepRunIds {
		orderedWorkerIds = append(orderedWorkerIds, sqlchelpers.UUIDFromStr(workerId))
		assignedStepRunsBytes, _ := json.Marshal(stepRunIds) // nolint: errcheck
		assignedStepRuns = append(assignedStepRuns, assignedStepRunsBytes)
	}

	err := s.queries.CreateWorkerAssignEvents(ctx, s.pool, dbsqlc.CreateWorkerAssignEventsParams{
		Workerids:        orderedWorkerIds,
		Assignedstepruns: assignedStepRuns,
	})

	if err != nil {
		s.l.Err(err).Msg("could not create worker assign events")
	}
}

func (s *queueRepository) bulkStepRunsUnassigned(
	tenantId string,
	stepRunIds []pgtype.UUID,
) {
	for _, stepRunId := range stepRunIds {
		message := "No worker available"
		timeSeen := time.Now().UTC()
		severity := dbsqlc.StepRunEventSeverityWARNING
		reason := dbsqlc.StepRunEventReasonREQUEUEDNOWORKER
		data := map[string]interface{}{}

		err := s.bulkEventBuffer.FireForget(tenantId, &repository.CreateStepRunEventOpts{
			StepRunId:     sqlchelpers.UUIDToStr(stepRunId),
			EventMessage:  &message,
			EventReason:   &reason,
			EventSeverity: &severity,
			Timestamp:     &timeSeen,
			EventData:     data,
		})

		if err != nil {
			s.l.Err(err).Msg("could not buffer step run event")
		}
	}
}

func (s *queueRepository) bulkStepRunsRateLimited(
	tenantId string,
	rateLimits []*repository.RateLimitResult,
) {
	for _, rlResult := range rateLimits {
		message := fmt.Sprintf(
			"Rate limit exceeded for key %s, attempting to consume %d units, but only had %d remaining",
			rlResult.ExceededKey,
			rlResult.ExceededUnits,
			rlResult.ExceededVal,
		)

		reason := dbsqlc.StepRunEventReasonREQUEUEDRATELIMIT
		severity := dbsqlc.StepRunEventSeverityWARNING
		timeSeen := time.Now().UTC()
		data := map[string]interface{}{
			"rate_limit_key": rlResult.ExceededKey,
		}

		err := s.bulkEventBuffer.FireForget(tenantId, &repository.CreateStepRunEventOpts{
			StepRunId:     sqlchelpers.UUIDToStr(rlResult.StepRunId),
			EventMessage:  &message,
			EventReason:   &reason,
			EventSeverity: &severity,
			Timestamp:     &timeSeen,
			EventData:     data,
		})

		if err != nil {
			s.l.Err(err).Msg("could not buffer step run event")
		}
	}
}

func (d *queueRepository) updateMinId() {
	if !d.updateMinIdMu.TryLock() {
		return
	}
	defer d.updateMinIdMu.Unlock()

	dbCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	minId, err := d.queries.GetMinUnprocessedQueueItemId(dbCtx, d.pool, dbsqlc.GetMinUnprocessedQueueItemIdParams{
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

func (d *queueRepository) MarkQueueItemsProcessed(ctx context.Context, r *repository.AssignResults) (succeeded []*repository.AssignedItem, failed []*repository.AssignedItem, err error) {
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
	stepRunIds := make([]pgtype.UUID, len(r.Assigned))
	workerIds := make([]pgtype.UUID, len(r.Assigned))
	stepTimeouts := make([]string, len(r.Assigned))

	for i, assignedItem := range r.Assigned {
		idsToUnqueue[i] = assignedItem.QueueItem.ID
		stepRunIds[i] = assignedItem.QueueItem.StepRunId
		workerIds[i] = assignedItem.WorkerId
		stepTimeouts[i] = assignedItem.QueueItem.StepTimeout.String
	}

	unassignedStepRunIds := make([]pgtype.UUID, 0, len(r.Unassigned))

	for _, id := range r.Unassigned {
		unassignedStepRunIds = append(unassignedStepRunIds, id.StepRunId)
	}

	timedOutStepRuns := make([]pgtype.UUID, 0, len(r.SchedulingTimedOut))

	for _, id := range r.SchedulingTimedOut {
		idsToUnqueue = append(idsToUnqueue, id.ID)
		timedOutStepRuns = append(timedOutStepRuns, id.StepRunId)
	}

	_, err = d.queries.BulkMarkStepRunsAsCancelling(ctx, tx, timedOutStepRuns)

	if err != nil {
		return nil, nil, fmt.Errorf("could not bulk mark step runs as cancelling: %w", err)
	}

	updatedStepRuns, err := d.queries.UpdateStepRunsToAssigned(ctx, tx, dbsqlc.UpdateStepRunsToAssignedParams{
		Steprunids:      stepRunIds,
		Workerids:       workerIds,
		Stepruntimeouts: stepTimeouts,
		Tenantid:        d.tenantId,
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

		assignedStepRuns := make([]pgtype.UUID, len(updatedStepRuns))
		assignedWorkerIds := make([]pgtype.UUID, len(updatedStepRuns))

		for i, row := range updatedStepRuns {
			assignedStepRuns[i] = row.StepRunId
			assignedWorkerIds[i] = row.WorkerId
		}

		d.bulkStepRunsAssigned(sqlchelpers.UUIDToStr(d.tenantId), time.Now().UTC(), assignedStepRuns, assignedWorkerIds)
		d.bulkStepRunsUnassigned(sqlchelpers.UUIDToStr(d.tenantId), unassignedStepRunIds)
		d.bulkStepRunsRateLimited(sqlchelpers.UUIDToStr(d.tenantId), r.RateLimited)
	}()

	stepRunIdToAssignedItem := make(map[string]*repository.AssignedItem, len(updatedStepRuns))

	for _, assignedItem := range r.Assigned {
		stepRunIdToAssignedItem[sqlchelpers.UUIDToStr(assignedItem.QueueItem.StepRunId)] = assignedItem
	}

	succeeded = make([]*repository.AssignedItem, 0, len(r.Assigned))
	failed = make([]*repository.AssignedItem, 0, len(r.Assigned))

	for _, row := range updatedStepRuns {
		if assignedItem, ok := stepRunIdToAssignedItem[sqlchelpers.UUIDToStr(row.StepRunId)]; ok {
			succeeded = append(succeeded, assignedItem)
			delete(stepRunIdToAssignedItem, sqlchelpers.UUIDToStr(row.StepRunId))
		}
	}

	for _, assignedItem := range stepRunIdToAssignedItem {
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
		).Int(
			"unassigned", len(unassignedStepRunIds),
		).Int(
			"timed_out", len(timedOutStepRuns),
		).Msgf(
			"marking queue items processed took longer than 100ms",
		)
	}

	return succeeded, failed, nil
}

func (d *queueRepository) GetStepRunRateLimits(ctx context.Context, queueItems []*dbsqlc.QueueItem) (map[string]map[string]int32, error) {
	ctx, span := telemetry.NewSpan(ctx, "get-step-run-rate-limits")
	defer span.End()

	stepRunIds := make([]pgtype.UUID, 0, len(queueItems))
	stepIds := make([]pgtype.UUID, 0, len(queueItems))
	stepsWithRateLimits := make(map[string]bool)

	for _, item := range queueItems {
		stepRunIds = append(stepRunIds, item.StepRunId)
		stepIds = append(stepIds, item.StepId)
	}

	stepIdToStepRuns := make(map[string][]string)
	stepRunIdToStepId := make(map[string]string)

	for i, stepRunId := range stepRunIds {
		stepId := sqlchelpers.UUIDToStr(stepIds[i])
		stepRunIdStr := sqlchelpers.UUIDToStr(stepRunId)

		if _, ok := stepIdToStepRuns[stepId]; !ok {
			stepIdToStepRuns[stepId] = make([]string, 0)
		}

		stepIdToStepRuns[stepId] = append(stepIdToStepRuns[stepId], stepRunIdStr)
		stepRunIdToStepId[stepRunIdStr] = stepId
	}

	// check if we have any rate limits for these step ids
	skipRateLimiting := true

	for stepIdStr := range stepIdToStepRuns {
		if hasRateLimit, ok := d.cachedStepIdHasRateLimit.Get(stepIdStr); !ok || hasRateLimit.(bool) {
			skipRateLimiting = false
			break
		}
	}

	if skipRateLimiting {
		return nil, nil
	}

	// get all step run expression evals which correspond to rate limits, grouped by step run id
	expressionEvals, err := d.queries.ListStepRunExpressionEvals(ctx, d.pool, stepRunIds)

	if err != nil {
		return nil, err
	}

	stepRunAndGlobalKeyToKey := make(map[string]string)
	stepRunToKeys := make(map[string][]string)

	for _, eval := range expressionEvals {
		stepRunId := sqlchelpers.UUIDToStr(eval.StepRunId)
		globalKey := eval.Key

		// Only append if this is a key expression. Note that we have a uniqueness constraint on
		// the stepRunId, kind, and key, so we will not insert duplicate values into the array.
		if eval.Kind == dbsqlc.StepExpressionKindDYNAMICRATELIMITKEY {
			stepsWithRateLimits[stepRunIdToStepId[stepRunId]] = true

			k := eval.ValueStr.String

			if _, ok := stepRunToKeys[stepRunId]; !ok {
				stepRunToKeys[stepRunId] = make([]string, 0)
			}

			stepRunToKeys[stepRunId] = append(stepRunToKeys[stepRunId], k)

			stepRunAndGlobalKey := fmt.Sprintf("%s-%s", stepRunId, globalKey)

			stepRunAndGlobalKeyToKey[stepRunAndGlobalKey] = k
		}
	}

	rateLimitKeyToEvals := make(map[string][]*dbsqlc.StepRunExpressionEval)

	for _, eval := range expressionEvals {
		k := stepRunAndGlobalKeyToKey[fmt.Sprintf("%s-%s", sqlchelpers.UUIDToStr(eval.StepRunId), eval.Key)]

		if _, ok := rateLimitKeyToEvals[k]; !ok {
			rateLimitKeyToEvals[k] = make([]*dbsqlc.StepRunExpressionEval, 0)
		}

		rateLimitKeyToEvals[k] = append(rateLimitKeyToEvals[k], eval)
	}

	upsertRateLimitBulkParams := dbsqlc.UpsertRateLimitsBulkParams{
		Tenantid: d.tenantId,
	}

	stepRunToKeyToUnits := make(map[string]map[string]int32)

	for key, evals := range rateLimitKeyToEvals {
		var duration string
		var limitValue int
		var skip bool

		for _, eval := range evals {
			// add to stepRunToKeyToUnits
			stepRunId := sqlchelpers.UUIDToStr(eval.StepRunId)

			// throw an error if there are multiple rate limits with the same keys, but different limit values or durations
			if eval.Kind == dbsqlc.StepExpressionKindDYNAMICRATELIMITWINDOW {
				if duration == "" {
					duration = eval.ValueStr.String
				} else if duration != eval.ValueStr.String {
					largerDuration, err := getLargerDuration(duration, eval.ValueStr.String)

					if err != nil {
						skip = true
						break
					}

					message := fmt.Sprintf("Multiple rate limits with key %s have different durations: %s vs %s. Using longer window %s.", key, duration, eval.ValueStr.String, largerDuration)
					timeSeen := time.Now().UTC()
					reason := dbsqlc.StepRunEventReasonRATELIMITERROR
					severity := dbsqlc.StepRunEventSeverityWARNING
					data := map[string]interface{}{}

					buffErr := d.bulkEventBuffer.FireForget(sqlchelpers.UUIDToStr(d.tenantId), &repository.CreateStepRunEventOpts{
						StepRunId:     sqlchelpers.UUIDToStr(eval.StepRunId),
						EventMessage:  &message,
						EventReason:   &reason,
						EventSeverity: &severity,
						Timestamp:     &timeSeen,
						EventData:     data,
					})

					if buffErr != nil {
						d.l.Err(buffErr).Msg("could not buffer step run event")
					}

					duration = largerDuration
				}
			}

			if eval.Kind == dbsqlc.StepExpressionKindDYNAMICRATELIMITVALUE {
				if limitValue == 0 {
					limitValue = int(eval.ValueInt.Int32)
				} else if limitValue != int(eval.ValueInt.Int32) {
					message := fmt.Sprintf("Multiple rate limits with key %s have different limit values: %d vs %d. Using lower value %d.", key, limitValue, eval.ValueInt.Int32, min(limitValue, int(eval.ValueInt.Int32)))
					timeSeen := time.Now().UTC()
					reason := dbsqlc.StepRunEventReasonRATELIMITERROR
					severity := dbsqlc.StepRunEventSeverityWARNING
					data := map[string]interface{}{}

					buffErr := d.bulkEventBuffer.FireForget(sqlchelpers.UUIDToStr(d.tenantId), &repository.CreateStepRunEventOpts{
						StepRunId:     sqlchelpers.UUIDToStr(eval.StepRunId),
						EventMessage:  &message,
						EventReason:   &reason,
						EventSeverity: &severity,
						Timestamp:     &timeSeen,
						EventData:     data,
					})

					if buffErr != nil {
						d.l.Err(buffErr).Msg("could not buffer step run event")
					}

					limitValue = min(limitValue, int(eval.ValueInt.Int32))
				}
			}

			if eval.Kind == dbsqlc.StepExpressionKindDYNAMICRATELIMITUNITS {
				if _, ok := stepRunToKeyToUnits[stepRunId]; !ok {
					stepRunToKeyToUnits[stepRunId] = make(map[string]int32)
				}

				stepRunToKeyToUnits[stepRunId][key] = eval.ValueInt.Int32
			}
		}

		if skip {
			continue
		}

		upsertRateLimitBulkParams.Keys = append(upsertRateLimitBulkParams.Keys, key)
		upsertRateLimitBulkParams.Windows = append(upsertRateLimitBulkParams.Windows, getWindowParamFromDurString(duration))
		upsertRateLimitBulkParams.Limitvalues = append(upsertRateLimitBulkParams.Limitvalues, int32(limitValue)) // nolint: gosec
	}

	var stepRateLimits []*dbsqlc.StepRateLimit

	if len(upsertRateLimitBulkParams.Keys) > 0 {
		// upsert all rate limits based on the keys, limit values, and durations
		err = d.queries.UpsertRateLimitsBulk(ctx, d.pool, upsertRateLimitBulkParams)

		if err != nil {
			return nil, fmt.Errorf("could not bulk upsert dynamic rate limits: %w", err)
		}
	}

	// get all existing static rate limits for steps to the mapping, mapping back from step ids to step run ids
	uniqueStepIds := make([]pgtype.UUID, 0, len(stepIdToStepRuns))

	for stepId := range stepIdToStepRuns {
		uniqueStepIds = append(uniqueStepIds, sqlchelpers.UUIDFromStr(stepId))
	}

	stepRateLimits, err = d.queries.ListRateLimitsForSteps(ctx, d.pool, dbsqlc.ListRateLimitsForStepsParams{
		Tenantid: d.tenantId,
		Stepids:  uniqueStepIds,
	})

	if err != nil {
		return nil, fmt.Errorf("could not list rate limits for steps: %w", err)
	}

	for _, row := range stepRateLimits {
		stepsWithRateLimits[sqlchelpers.UUIDToStr(row.StepId)] = true
		stepId := sqlchelpers.UUIDToStr(row.StepId)
		stepRuns := stepIdToStepRuns[stepId]

		for _, stepRunId := range stepRuns {
			if _, ok := stepRunToKeyToUnits[stepRunId]; !ok {
				stepRunToKeyToUnits[stepRunId] = make(map[string]int32)
			}

			stepRunToKeyToUnits[stepRunId][row.RateLimitKey] = row.Units
		}
	}

	// store all step ids in the cache, so we can skip rate limiting for steps without rate limits
	for stepId := range stepIdToStepRuns {
		hasRateLimit := stepsWithRateLimits[stepId]
		d.cachedStepIdHasRateLimit.Set(stepId, hasRateLimit)
	}

	return stepRunToKeyToUnits, nil
}

func (d *queueRepository) GetDesiredLabels(ctx context.Context, stepIds []pgtype.UUID) (map[string][]*dbsqlc.GetDesiredLabelsRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "get-desired-labels")
	defer span.End()

	uniqueStepIds := sqlchelpers.UniqueSet(stepIds)

	labels, err := d.queries.GetDesiredLabels(ctx, d.pool, uniqueStepIds)

	if err != nil {
		return nil, err
	}

	stepIdToLabels := make(map[string][]*dbsqlc.GetDesiredLabelsRow)

	for _, label := range labels {
		stepId := sqlchelpers.UUIDToStr(label.StepId)

		if _, ok := stepIdToLabels[stepId]; !ok {
			stepIdToLabels[stepId] = make([]*dbsqlc.GetDesiredLabelsRow, 0)
		}

		stepIdToLabels[stepId] = append(stepIdToLabels[stepId], label)
	}

	return stepIdToLabels, nil
}
