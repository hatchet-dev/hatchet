package v2

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/sasha-s/go-deadlock"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/buffer"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

type queuerRepo interface {
	ListQueueItems(ctx context.Context) ([]*dbsqlc.QueueItem, error)
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

	limit  pgtype.Int4
	gtId   pgtype.Int8
	gtIdMu deadlock.RWMutex

	eventBuffer              *buffer.BulkEventWriter
	cachedStepIdHasRateLimit *cache.Cache
}

func newQueueItemDbQueries(cf *sharedConfig, tenantId pgtype.UUID, eventBuffer *buffer.BulkEventWriter, queueName string, limit int32,
) (*queuerDbQueries, func()) {
	c := cache.New(5 * time.Minute)
	return &queuerDbQueries{
		tenantId:    tenantId,
		queueName:   queueName,
		queries:     cf.queries,
		pool:        cf.pool,
		l:           cf.l,
		eventBuffer: eventBuffer,
		limit: pgtype.Int4{
			Int32: limit,
			Valid: true,
		},
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

func (d *queuerDbQueries) ListQueueItems(ctx context.Context) ([]*dbsqlc.QueueItem, error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, d.pool, d.l, 5000)

	if err != nil {
		return nil, err
	}

	defer rollback()

	qis, err := d.queries.ListQueueItemsForQueue(ctx, tx, dbsqlc.ListQueueItemsForQueueParams{
		Tenantid: d.tenantId,
		Queue:    d.queueName,
		GtId:     d.getMinId(),
		Limit:    d.limit,
	})

	if err != nil {
		return nil, err
	}

	qis, err = d.removeInvalidStepRuns(ctx, tx, qis)

	if err != nil {
		return nil, err
	}

	if err := commit(ctx); err != nil {
		return nil, err
	}

	return qis, nil
}

// removeInvalidStepRuns removes all duplicate step runs and step runs which are in a finalized state from
// the queue. It returns the remaining queue items and an error if one occurred.
func (s *queuerDbQueries) removeInvalidStepRuns(ctx context.Context, tx pgx.Tx, qis []*dbsqlc.QueueItem) ([]*dbsqlc.QueueItem, error) {
	if len(qis) == 0 {
		return qis, nil
	}

	currStepRunIds := make([]pgtype.UUID, len(qis))

	for i, qi := range qis {
		currStepRunIds[i] = qi.StepRunId
	}

	// remove duplicates
	encountered := map[string]bool{}
	remaining1 := make([]*dbsqlc.QueueItem, 0, len(qis))
	cancelled := make([]int64, 0, len(qis))

	for _, v := range qis {
		stepRunId := sqlchelpers.UUIDToStr(v.StepRunId)

		if encountered[stepRunId] {
			cancelled = append(cancelled, v.ID)
			continue
		}

		encountered[stepRunId] = true
		remaining1 = append(remaining1, v)
	}

	finalizedStepRuns, err := s.queries.GetFinalizedStepRuns(ctx, tx, currStepRunIds)

	if err != nil {
		return nil, err
	}

	finalizedStepRunsMap := make(map[string]bool, len(finalizedStepRuns))

	for _, sr := range finalizedStepRuns {
		s.l.Warn().Msgf("step run %s is in state %s, skipping queueing", sqlchelpers.UUIDToStr(sr.ID), string(sr.Status))
		finalizedStepRunsMap[sqlchelpers.UUIDToStr(sr.ID)] = true
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

	if len(cancelled) > 0 {
		err = s.queries.BulkQueueItems(ctx, tx, cancelled)

		if err != nil {
			return nil, err
		}
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
	rateLimits []*rateLimitResult,
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
	start := time.Now()

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, d.pool, d.l, 5000)

	if err != nil {
		return nil, nil, err
	}

	defer rollback()

	timeAfterPrepare := time.Since(start)

	// d.queries.UpdateStepRunsToAssigned
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

	// TODO: ADD UNIQUE CONSTRAINT TO SEMAPHORES WITH ON CONFLICT DO NOTHING, THEN DON'T
	// QUEUE ITEMS THAT ALREADY HAVE SEMAPHORES
	err = d.queries.UpdateStepRunsToAssigned(ctx, tx, dbsqlc.UpdateStepRunsToAssignedParams{
		Steprunids:      stepRunIds,
		Workerids:       workerIds,
		Stepruntimeouts: stepTimeouts,
		Tenantid:        d.tenantId,
	})

	if err != nil {
		return nil, nil, err
	}

	timeAfterUpdateStepRuns := time.Since(start)

	err = d.queries.BulkQueueItems(ctx, tx, idsToUnqueue)

	if err != nil {
		return nil, nil, err
	}

	timeAfterBulkQueueItems := time.Since(start)

	dispatcherIdWorkerIds, err := d.queries.ListDispatcherIdsForWorkers(ctx, tx, dbsqlc.ListDispatcherIdsForWorkersParams{
		Tenantid:  d.tenantId,
		Workerids: sqlchelpers.UniqueSet(workerIds),
	})

	if err := commit(ctx); err != nil {
		return nil, nil, err
	}

	// if we committed, we can update the min id
	go d.updateMinId()

	d.bulkStepRunsAssigned(sqlchelpers.UUIDToStr(d.tenantId), time.Now().UTC(), stepRunIds, workerIds)
	d.bulkStepRunsUnassigned(sqlchelpers.UUIDToStr(d.tenantId), unassignedStepRunIds)
	d.bulkStepRunsRateLimited(sqlchelpers.UUIDToStr(d.tenantId), r.rateLimited)

	workerIdToDispatcherId := make(map[string]pgtype.UUID, len(dispatcherIdWorkerIds))

	for _, dispatcherIdWorkerId := range dispatcherIdWorkerIds {
		workerIdToDispatcherId[sqlchelpers.UUIDToStr(dispatcherIdWorkerId.WorkerId)] = dispatcherIdWorkerId.DispatcherId
	}

	succeeded = make([]*AssignedQueueItem, 0, len(r.assigned))
	failed = make([]*AssignedQueueItem, 0, len(r.assigned))

	for _, assignedItem := range r.assigned {
		dispatcherId, ok := workerIdToDispatcherId[sqlchelpers.UUIDToStr(assignedItem.WorkerId)]

		if !ok {
			failed = append(failed, assignedItem)
			continue
		}

		assignedItem.DispatcherId = &dispatcherId
		succeeded = append(succeeded, assignedItem)
	}

	if sinceStart := time.Since(start); sinceStart > 100*time.Millisecond {
		d.l.Warn().Msgf(
			"marking queue items processed took longer than 100ms (%s) (prepare=%s, update=%s, bulkqueue=%s)",
			sinceStart, timeAfterPrepare, timeAfterUpdateStepRuns, timeAfterBulkQueueItems,
		)
	}

	return succeeded, failed, nil
}

func (d *queuerDbQueries) GetStepRunRateLimits(ctx context.Context, queueItems []*dbsqlc.QueueItem) (map[string]map[string]int32, error) {
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
	labels, err := d.queries.GetDesiredLabels(ctx, d.pool, stepIds)

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

	queueMu deadlock.Mutex

	cleanup func()

	isCleanedUp bool
}

func newQueuer(conf *sharedConfig, tenantId pgtype.UUID, queueName string, s *Scheduler, eventBuffer *buffer.BulkEventWriter, resultsCh chan<- *QueueResults) *Queuer {
	defaultLimit := 100

	if conf.singleQueueLimit > 0 {
		defaultLimit = conf.singleQueueLimit
	}

	repo, cleanupRepo := newQueueItemDbQueries(conf, tenantId, eventBuffer, queueName, int32(defaultLimit)) // nolint: gosec

	notifyQueueCh := make(chan struct{})

	q := &Queuer{
		repo:          repo,
		tenantId:      tenantId,
		queueName:     queueName,
		l:             conf.l,
		s:             s,
		limit:         100,
		resultsCh:     resultsCh,
		notifyQueueCh: notifyQueueCh,
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
	qis := make([]*dbsqlc.QueueItem, 0)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-q.notifyQueueCh:
		}

		start := time.Now()

		qis, err := q.refillQueue(ctx, qis)

		if err != nil {
			q.l.Error().Err(err).Msg("error refilling queue")
			continue
		}

		rls, err := q.repo.GetStepRunRateLimits(ctx, qis)

		if err != nil {
			q.l.Error().Err(err).Msg("error getting rate limits")
			continue
		}

		stepIds := make([]pgtype.UUID, 0, len(qis))

		for _, qi := range qis {
			stepIds = append(stepIds, qi.StepId)
		}

		labels, err := q.repo.GetDesiredLabels(ctx, stepIds)

		if err != nil {
			q.l.Error().Err(err).Msg("error getting desired labels")
			continue
		}

		timeToRefill := time.Since(start)

		assignCh := q.s.tryAssign(ctx, qis, labels, rls)
		count := 0
		countMu := sync.Mutex{}

		wg := sync.WaitGroup{}

		for r := range assignCh {
			wg.Add(1)

			go func() {
				defer wg.Done()

				startFlush := time.Now()

				numFlushed := q.flushToDatabase(ctx, r)

				countMu.Lock()
				count += numFlushed
				countMu.Unlock()

				if sinceStart := time.Since(startFlush); sinceStart > 100*time.Millisecond {
					q.l.Warn().Msgf("flushing items to database took longer than 100ms (%d items in %s)", numFlushed, time.Since(startFlush))
				}
			}()
		}

		wg.Wait()

		elapsed := time.Since(start)

		if elapsed > 100*time.Millisecond {
			q.l.Warn().Msgf("queue %s took longer than 100ms (%s) to process %d items (time to refill %s)", q.queueName, elapsed, len(qis), timeToRefill)
		}

		// if we processed all queue items, queue again
		if len(qis) > 0 && count == len(qis) {
			go q.queue()
		}
	}
}

func (q *Queuer) refillQueue(ctx context.Context, curr []*dbsqlc.QueueItem) ([]*dbsqlc.QueueItem, error) {
	// determine whether we need to replenish with the following cases:
	// - we last replenished more than 1 second ago
	// - if we are at less than 50% of the limit, we always attempt to replenish
	replenish := false
	now := time.Now()

	if len(curr) < q.limit/2 {
		replenish = true
	}

	if q.lastReplenished != nil {
		if time.Since(*q.lastReplenished) > 990*time.Millisecond {
			replenish = true
		}
	}

	if !replenish {
		return curr, nil
	}

	q.lastReplenished = &now

	return q.repo.ListQueueItems(ctx)
}

type QueueResults struct {
	TenantId pgtype.UUID
	Assigned []*AssignedQueueItem

	// A list of step run ids that were not assigned because they reached the scheduling
	// timeout
	SchedulingTimedOut []string
}

func (q *Queuer) flushToDatabase(ctx context.Context, r *assignResults) int {
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
