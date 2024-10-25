package v2

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/buffer"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

type queuerRepo interface {
	ListQueueItems(ctx context.Context, limit int) ([]*dbsqlc.QueueItem, error)
	MarkQueueItemsProcessed(ctx context.Context, r *assignResults) (succeeded []*AssignedQueueItem, failed []*AssignedQueueItem, err error)
	GetStepRunRateLimits(ctx context.Context, queueItems []*dbsqlc.QueueItem) (map[string]map[string]int32, error)
	GetDesiredLabels(ctx context.Context, stepIds []pgtype.UUID) (map[string][]*dbsqlc.GetDesiredLabelsRow, error)
}

type queuerDbQueries struct {
	tenantId  pgtype.UUID
	queueName string

	queries *dbsqlc.Queries
	pool    *pgxpool.Pool
	l       *zerolog.Logger

	gtId   pgtype.Int8
	gtIdMu sync.RWMutex

	eventBuffer              *buffer.BulkEventWriter
	cachedStepIdHasRateLimit *cache.Cache
}

func newQueueItemDbQueries(cf *sharedConfig, tenantId pgtype.UUID, eventBuffer *buffer.BulkEventWriter, queueName string,
) (*queuerDbQueries, func()) {
	c := cache.New(5 * time.Minute)
	return &queuerDbQueries{
		tenantId:                 tenantId,
		queueName:                queueName,
		queries:                  cf.queries,
		pool:                     cf.pool,
		l:                        cf.l,
		eventBuffer:              eventBuffer,
		cachedStepIdHasRateLimit: c,
	}, c.Stop
}

func (d *queuerDbQueries) setMinId(id int64) {
	d.gtIdMu.Lock()
	defer d.gtIdMu.Unlock()

	d.gtId = pgtype.Int8{
		Int64: id,
		Valid: true,
	}
}

func (d *queuerDbQueries) getMinId() pgtype.Int8 {
	d.gtIdMu.RLock()
	defer d.gtIdMu.RUnlock()

	val := d.gtId

	return val
}

func (d *queuerDbQueries) ListQueueItems(ctx context.Context, limit int) ([]*dbsqlc.QueueItem, error) {
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
func (s *queuerDbQueries) removeInvalidStepRuns(ctx context.Context, qis []*dbsqlc.ListQueueItemsForQueueRow) ([]*dbsqlc.QueueItem, error) {
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

func (s *queuerDbQueries) bulkStepRunsAssigned(
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

		_, err := s.eventBuffer.BuffItem(tenantId, &repository.CreateStepRunEventOpts{
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

func (s *queuerDbQueries) bulkStepRunsUnassigned(
	tenantId string,
	stepRunIds []pgtype.UUID,
) {
	for _, stepRunId := range stepRunIds {
		message := "No worker available"
		timeSeen := time.Now().UTC()
		severity := dbsqlc.StepRunEventSeverityWARNING
		reason := dbsqlc.StepRunEventReasonREQUEUEDNOWORKER
		data := map[string]interface{}{}

		_, err := s.eventBuffer.BuffItem(tenantId, &repository.CreateStepRunEventOpts{
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

func (s *queuerDbQueries) bulkStepRunsRateLimited(
	tenantId string,
	rateLimits []*scheduleRateLimitResult,
) {
	for _, rlResult := range rateLimits {
		message := fmt.Sprintf(
			"Rate limit exceeded for key %s, attempting to consume %d units, but only had %d remaining",
			rlResult.exceededKey,
			rlResult.exceededUnits,
			rlResult.exceededVal,
		)

		reason := dbsqlc.StepRunEventReasonREQUEUEDRATELIMIT
		severity := dbsqlc.StepRunEventSeverityWARNING
		timeSeen := time.Now().UTC()
		data := map[string]interface{}{
			"rate_limit_key": rlResult.exceededKey,
		}

		_, err := s.eventBuffer.BuffItem(tenantId, &repository.CreateStepRunEventOpts{
			StepRunId:     rlResult.stepRunId,
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

func (d *queuerDbQueries) updateMinId() {
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

func (d *queuerDbQueries) MarkQueueItemsProcessed(ctx context.Context, r *assignResults) (
	succeeded []*AssignedQueueItem, failed []*AssignedQueueItem, err error,
) {
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

	idsToUnqueue := make([]int64, len(r.assigned))
	stepRunIds := make([]pgtype.UUID, len(r.assigned))
	workerIds := make([]pgtype.UUID, len(r.assigned))
	stepTimeouts := make([]string, len(r.assigned))

	for i, assignedItem := range r.assigned {
		idsToUnqueue[i] = assignedItem.QueueItem.ID
		stepRunIds[i] = assignedItem.QueueItem.StepRunId
		workerIds[i] = assignedItem.WorkerId
		stepTimeouts[i] = assignedItem.QueueItem.StepTimeout.String
	}

	unassignedStepRunIds := make([]pgtype.UUID, 0, len(r.unassigned))

	for _, id := range r.unassigned {
		unassignedStepRunIds = append(unassignedStepRunIds, id.StepRunId)
	}

	timedOutStepRuns := make([]pgtype.UUID, 0, len(r.schedulingTimedOut))

	for _, id := range r.schedulingTimedOut {
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
		d.bulkStepRunsRateLimited(sqlchelpers.UUIDToStr(d.tenantId), r.rateLimited)
	}()

	stepRunIdToAssignedItem := make(map[string]*AssignedQueueItem, len(updatedStepRuns))

	for _, assignedItem := range r.assigned {
		stepRunIdToAssignedItem[sqlchelpers.UUIDToStr(assignedItem.QueueItem.StepRunId)] = assignedItem
	}

	succeeded = make([]*AssignedQueueItem, 0, len(r.assigned))
	failed = make([]*AssignedQueueItem, 0, len(r.assigned))

	for _, row := range updatedStepRuns {
		succeeded = append(succeeded, stepRunIdToAssignedItem[sqlchelpers.UUIDToStr(row.StepRunId)])
		delete(stepRunIdToAssignedItem, sqlchelpers.UUIDToStr(row.StepRunId))
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

func (d *queuerDbQueries) GetStepRunRateLimits(ctx context.Context, queueItems []*dbsqlc.QueueItem) (map[string]map[string]int32, error) {
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

					_, buffErr := d.eventBuffer.BuffItem(sqlchelpers.UUIDToStr(d.tenantId), &repository.CreateStepRunEventOpts{
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

					_, buffErr := d.eventBuffer.BuffItem(sqlchelpers.UUIDToStr(d.tenantId), &repository.CreateStepRunEventOpts{
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

	rateLimitsForTenant, err := d.queries.ListRateLimitsForTenantWithMutate(ctx, d.pool, d.tenantId)

	if err != nil {
		return nil, fmt.Errorf("could not list rate limits for tenant: %w", err)
	}

	mapRateLimitsForTenant := make(map[string]*dbsqlc.ListRateLimitsForTenantWithMutateRow)

	for _, row := range rateLimitsForTenant {
		mapRateLimitsForTenant[row.Key] = row
	}

	// store all step ids in the cache, so we can skip rate limiting for steps without rate limits
	for stepId := range stepIdToStepRuns {
		hasRateLimit := stepsWithRateLimits[stepId]
		d.cachedStepIdHasRateLimit.Set(stepId, hasRateLimit)
	}

	return stepRunToKeyToUnits, nil
}

func (d *queuerDbQueries) GetDesiredLabels(ctx context.Context, stepIds []pgtype.UUID) (map[string][]*dbsqlc.GetDesiredLabelsRow, error) {
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

type Queuer struct {
	repo      queuerRepo
	tenantId  pgtype.UUID
	queueName string

	l *zerolog.Logger

	s *Scheduler

	lastReplenished *time.Time

	limit int

	resultsCh chan<- *QueueResults

	notifyQueueCh chan struct{}

	queueMu mutex

	cleanup func()

	isCleanedUp bool

	unackedMu rwMutex
	unacked   map[int64]struct{}

	unassigned   map[int64]*dbsqlc.QueueItem
	unassignedMu mutex
}

type alreadyAssigned struct {
	assignedQi      *dbsqlc.QueueItem
	assignedAtBatch int
	assignedAtCount int
}

func newQueuer(conf *sharedConfig, tenantId pgtype.UUID, queueName string, s *Scheduler, eventBuffer *buffer.BulkEventWriter, resultsCh chan<- *QueueResults) *Queuer {
	defaultLimit := 100

	if conf.singleQueueLimit > 0 {
		defaultLimit = conf.singleQueueLimit
	}

	repo, cleanupRepo := newQueueItemDbQueries(conf, tenantId, eventBuffer, queueName)

	notifyQueueCh := make(chan struct{}, 1)

	q := &Queuer{
		repo:          repo,
		tenantId:      tenantId,
		queueName:     queueName,
		l:             conf.l,
		s:             s,
		limit:         defaultLimit,
		resultsCh:     resultsCh,
		notifyQueueCh: notifyQueueCh,
		queueMu:       newMu(conf.l),
		unackedMu:     newRWMu(conf.l),
		unacked:       make(map[int64]struct{}),
		unassigned:    make(map[int64]*dbsqlc.QueueItem),
		unassignedMu:  newMu(conf.l),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cleanupMu := sync.Mutex{}
	q.cleanup = func() {
		cleanupMu.Lock()
		defer cleanupMu.Unlock()

		if q.isCleanedUp {
			return
		}

		q.isCleanedUp = true
		cleanupRepo()
		cancel()
	}

	go q.loopQueue(ctx)

	return q
}

func (q *Queuer) Cleanup() {
	q.cleanup()
}

func (q *Queuer) queue() {
	if ok := q.queueMu.TryLock(); !ok {
		return
	}

	go func() {
		defer q.queueMu.Unlock()

		q.notifyQueueCh <- struct{}{}
	}()
}

func (q *Queuer) loopQueue(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-q.notifyQueueCh:
		}

		ctx, span := telemetry.NewSpan(ctx, "queue")

		telemetry.WithAttributes(span, telemetry.AttributeKV{
			Key:   "queue",
			Value: q.queueName,
		})

		start := time.Now()
		checkpoint := start
		var err error
		qis, err := q.refillQueue(ctx)

		if err != nil {
			span.End()
			q.l.Error().Err(err).Msg("error refilling queue")
			continue
		}

		if len(qis) == 0 {
			span.End()
			continue
		}

		refillTime := time.Since(checkpoint)
		checkpoint = time.Now()

		rls, err := q.repo.GetStepRunRateLimits(ctx, qis)

		if err != nil {
			q.l.Error().Err(err).Msg("error getting rate limits")
			continue
		}

		rateLimitTime := time.Since(checkpoint)
		checkpoint = time.Now()

		stepIds := make([]pgtype.UUID, 0, len(qis))

		for _, qi := range qis {
			stepIds = append(stepIds, qi.StepId)
		}

		labels, err := q.repo.GetDesiredLabels(ctx, stepIds)

		if err != nil {
			q.l.Error().Err(err).Msg("error getting desired labels")
			continue
		}

		desiredLabelsTime := time.Since(checkpoint)
		checkpoint = time.Now()

		assignCh := q.s.tryAssign(ctx, qis, labels, rls)
		count := 0
		countMu := sync.Mutex{}
		wg := sync.WaitGroup{}

		for r := range assignCh {
			wg.Add(1)

			// asynchronously flush to database
			go func(ar *assignResults) {
				defer wg.Done()

				startFlush := time.Now()

				numFlushed := q.flushToDatabase(ctx, ar)

				countMu.Lock()
				count += numFlushed
				countMu.Unlock()

				if sinceStart := time.Since(startFlush); sinceStart > 100*time.Millisecond {
					q.l.Warn().Msgf("flushing items to database took longer than 100ms (%d items in %s)", numFlushed, time.Since(startFlush))
				}
			}(r)
		}

		assignTime := time.Since(checkpoint)
		elapsed := time.Since(start)

		if elapsed > 100*time.Millisecond {
			q.l.Warn().Dur(
				"refill_time", refillTime,
			).Dur(
				"rate_limit_time", rateLimitTime,
			).Dur(
				"desired_labels_time", desiredLabelsTime,
			).Dur(
				"assign_time", assignTime,
			).Msgf("queue %s took longer than 100ms (%s) to process %d items", q.queueName, elapsed, len(qis))
		}

		// if we processed all queue items, queue again
		prevQis := qis

		go func(originalStart time.Time) {
			wg.Wait()
			span.End()

			countMu.Lock()
			if len(prevQis) > 0 && count == len(prevQis) {
				q.queue()
			}
			countMu.Unlock()

			if sinceStart := time.Since(originalStart); sinceStart > 100*time.Millisecond {
				q.l.Warn().Dur(
					"duration", sinceStart,
				).Msgf("queue %s took longer than 100ms to process and flush %d items", q.queueName, len(prevQis))
			}
		}(start)
	}
}

func (q *Queuer) refillQueue(ctx context.Context) ([]*dbsqlc.QueueItem, error) {
	q.unackedMu.Lock()
	defer q.unackedMu.Unlock()

	q.unassignedMu.Lock()
	defer q.unassignedMu.Unlock()

	curr := make([]*dbsqlc.QueueItem, 0, len(q.unassigned))

	for _, qi := range q.unassigned {
		curr = append(curr, qi)
	}

	// determine whether we need to replenish with the following cases:
	// - we last replenished more than 1 second ago
	// - if we are at less than 50% of the limit, we always attempt to replenish
	replenish := false

	if len(curr) < q.limit {
		replenish = true
	} else if q.lastReplenished != nil {
		if time.Since(*q.lastReplenished) > 990*time.Millisecond {
			replenish = true
		}
	}

	if replenish {
		now := time.Now()
		q.lastReplenished = &now
		limit := 2 * q.limit

		var err error
		curr, err = q.repo.ListQueueItems(ctx, limit)

		if err != nil {
			return nil, err
		}
	}

	newCurr := make([]*dbsqlc.QueueItem, 0, len(curr))

	for _, qi := range curr {
		if _, ok := q.unacked[qi.ID]; !ok {
			newCurr = append(newCurr, qi)
		}
	}

	// add all newCurr to unacked so we don't assign them again
	for _, qi := range newCurr {
		q.unacked[qi.ID] = struct{}{}
	}

	return newCurr, nil
}

type QueueResults struct {
	TenantId pgtype.UUID
	Assigned []*AssignedQueueItem

	// A list of step run ids that were not assigned because they reached the scheduling
	// timeout
	SchedulingTimedOut []string
}

func (q *Queuer) ack(r *assignResults) {
	q.unackedMu.Lock()
	defer q.unackedMu.Unlock()

	q.unassignedMu.Lock()
	defer q.unassignedMu.Unlock()

	for _, assignedItem := range r.assigned {
		delete(q.unacked, assignedItem.QueueItem.ID)
		delete(q.unassigned, assignedItem.QueueItem.ID)
	}

	for _, unassignedItem := range r.unassigned {
		delete(q.unacked, unassignedItem.ID)
		q.unassigned[unassignedItem.ID] = unassignedItem
	}

	for _, schedulingTimedOutItem := range r.schedulingTimedOut {
		delete(q.unacked, schedulingTimedOutItem.ID)
		delete(q.unassigned, schedulingTimedOutItem.ID)
	}

	for _, rateLimitedItem := range r.rateLimited {
		delete(q.unacked, rateLimitedItem.qi.ID)
		q.unassigned[rateLimitedItem.qi.ID] = rateLimitedItem.qi
	}
}

func (q *Queuer) flushToDatabase(ctx context.Context, r *assignResults) int {
	// no matter what, we always ack the items in the queuer
	defer q.ack(r)

	ctx, span := telemetry.NewSpan(ctx, "flush-to-database")
	defer span.End()

	q.l.Debug().Int("assigned", len(r.assigned)).Int("unassigned", len(r.unassigned)).Int("scheduling_timed_out", len(r.schedulingTimedOut)).Msg("flushing to database")

	if len(r.assigned) == 0 && len(r.unassigned) == 0 && len(r.schedulingTimedOut) == 0 && len(r.rateLimited) == 0 {
		return 0
	}

	succeeded, failed, err := q.repo.MarkQueueItemsProcessed(ctx, r)

	if err != nil {
		q.l.Error().Err(err).Msg("error marking queue items processed")

		nackIds := make([]int, 0, len(r.assigned))

		for _, assignedItem := range r.assigned {
			nackIds = append(nackIds, assignedItem.AckId)
		}

		q.s.nack(nackIds)

		return 0
	}

	nackIds := make([]int, 0, len(failed))
	ackIds := make([]int, 0, len(succeeded))

	for _, failedItem := range failed {
		nackIds = append(nackIds, failedItem.AckId)
	}

	for _, assignedItem := range succeeded {
		ackIds = append(ackIds, assignedItem.AckId)
	}

	q.s.nack(nackIds)
	q.s.ack(ackIds)

	schedulingTimedOut := make([]string, 0, len(r.schedulingTimedOut))

	for _, id := range r.schedulingTimedOut {
		schedulingTimedOut = append(schedulingTimedOut, sqlchelpers.UUIDToStr(id.StepRunId))
	}

	q.resultsCh <- &QueueResults{
		TenantId:           q.tenantId,
		Assigned:           succeeded,
		SchedulingTimedOut: schedulingTimedOut,
	}

	q.l.Debug().Int("succeeded", len(succeeded)).Int("failed", len(failed)).Msg("flushed to database")

	return len(succeeded)
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

func getDurationIndex(s string) (int, error) {
	for i, d := range durationStrings {
		if d == s {
			return i, nil
		}
	}

	return -1, fmt.Errorf("invalid duration string: %s", s)
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
