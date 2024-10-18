package v2

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/queueutils"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

type schedulerRepo interface {
	ListActionsForWorkers(ctx context.Context, workerIds []pgtype.UUID) ([]*dbsqlc.ListActionsForWorkersRow, error)
	ListAvailableSlotsForWorkers(ctx context.Context, params dbsqlc.ListAvailableSlotsForWorkersParams) ([]*dbsqlc.ListAvailableSlotsForWorkersRow, error)
}

type schedulerDbQueries struct {
	queries *dbsqlc.Queries
	pool    *pgxpool.Pool

	tenantId pgtype.UUID
}

func newSchedulerDbQueries(queries *dbsqlc.Queries, pool *pgxpool.Pool, tenantId pgtype.UUID) *schedulerDbQueries {
	return &schedulerDbQueries{
		queries:  queries,
		pool:     pool,
		tenantId: tenantId,
	}
}

func (d *schedulerDbQueries) ListActionsForWorkers(ctx context.Context, workerIds []pgtype.UUID) ([]*dbsqlc.ListActionsForWorkersRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "list-actions-for-workers")
	defer span.End()

	return d.queries.ListActionsForWorkers(ctx, d.pool, dbsqlc.ListActionsForWorkersParams{
		Tenantid:  d.tenantId,
		Workerids: workerIds,
	})
}

func (d *schedulerDbQueries) ListAvailableSlotsForWorkers(ctx context.Context, params dbsqlc.ListAvailableSlotsForWorkersParams) ([]*dbsqlc.ListAvailableSlotsForWorkersRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "list-available-slots-for-workers")
	defer span.End()

	return d.queries.ListAvailableSlotsForWorkers(ctx, d.pool, params)
}

// Scheduler is responsible for scheduling steps to workers as efficiently as possible.
// This is tenant-scoped, so each tenant will have its own scheduler.
type Scheduler struct {
	repo     schedulerRepo
	tenantId pgtype.UUID

	l *zerolog.Logger

	actions     map[string]*action
	actionsMu   rwMutex
	replenishMu mutex

	workersMu mutex
	workers   map[string]*worker

	assignedCount   int
	assignedCountMu mutex

	// unackedSlots are slots which have been assigned to a worker, but have not been flushed
	// to the database yet. They negatively count towards a worker's available slot count.
	unackedSlots map[int]*slot
	unackedMu    mutex

	rl *rateLimiter
}

func newScheduler(cf *sharedConfig, tenantId pgtype.UUID, rl *rateLimiter) *Scheduler {
	return &Scheduler{
		repo:            newSchedulerDbQueries(cf.queries, cf.pool, tenantId),
		tenantId:        tenantId,
		l:               cf.l,
		actions:         make(map[string]*action),
		unackedSlots:    make(map[int]*slot),
		rl:              rl,
		actionsMu:       newRWMu(cf.l),
		replenishMu:     newMu(cf.l),
		workersMu:       newMu(cf.l),
		assignedCountMu: newMu(cf.l),
		unackedMu:       newMu(cf.l),
	}
}

func (s *Scheduler) ack(ids []int) {
	s.unackedMu.Lock()
	defer s.unackedMu.Unlock()

	for _, id := range ids {
		if slot, ok := s.unackedSlots[id]; ok {
			slot.ack()
			delete(s.unackedSlots, id)
		}
	}
}

func (s *Scheduler) nack(ids []int) {
	s.unackedMu.Lock()
	defer s.unackedMu.Unlock()

	for _, id := range ids {
		if slot, ok := s.unackedSlots[id]; ok {
			slot.nack()
			delete(s.unackedSlots, id)
		}
	}
}

func (s *Scheduler) setWorkers(workers []*ListActiveWorkersResult) {
	s.workersMu.Lock()
	defer s.workersMu.Unlock()

	newWorkers := make(map[string]*worker, len(workers))

	for i := range workers {
		newWorkers[sqlchelpers.UUIDToStr(workers[i].ID)] = &worker{
			ListActiveWorkersResult: workers[i],
		}
	}

	s.workers = newWorkers
}

func (s *Scheduler) getWorkers() map[string]*worker {
	s.workersMu.Lock()
	defer s.workersMu.Unlock()

	return s.workers
}

// replenish loads new slots from the database.
func (s *Scheduler) replenish(ctx context.Context, mustReplenish bool) error {
	if mustReplenish {
		s.replenishMu.Lock()
	} else if ok := s.replenishMu.TryLock(); !ok {
		return nil
	}

	defer s.replenishMu.Unlock()

	workers := s.getWorkers()
	workerIds := make([]pgtype.UUID, 0)

	for workerIdStr := range workers {
		workerIds = append(workerIds, sqlchelpers.UUIDFromStr(workerIdStr))
	}

	workersToActiveActions, err := s.repo.ListActionsForWorkers(ctx, workerIds)

	if err != nil {
		return err
	}

	actionsToWorkerIds := make(map[string][]string)
	workerIdsToActions := make(map[string][]string)

	for _, workerActionTuple := range workersToActiveActions {
		if !workerActionTuple.ActionId.Valid {
			continue
		}

		actionId := workerActionTuple.ActionId.String
		workerId := sqlchelpers.UUIDToStr(workerActionTuple.WorkerId)

		actionsToWorkerIds[actionId] = append(actionsToWorkerIds[actionId], workerId)
		workerIdsToActions[workerId] = append(workerIdsToActions[workerId], actionId)
	}

	// FUNCTION 1: determine which actions should be replenished. Logic is the following:
	// - zero or one slots for an action: replenish all slots
	// - some slots for an action: replenish if 50% of slots have been used, or have expired
	// - more workers available for an action than previously: fully replenish
	// - otherwise, do not replenish
	actionsToReplenish := make(map[string]bool)
	s.actionsMu.RLock()

	for actionId, workers := range actionsToWorkerIds {
		if mustReplenish {
			actionsToReplenish[actionId] = true
			continue
		}

		// if the action is not in the map, it should be replenished
		if _, ok := s.actions[actionId]; !ok {
			actionsToReplenish[actionId] = true
			continue
		}

		storedAction := s.actions[actionId]

		// determine if we match the conditions above
		var replenish bool
		activeCount := storedAction.activeCount()

		if activeCount == 0 {
			replenish = true
		} else if activeCount <= (storedAction.lastReplenishedSlotCount / 2) {
			replenish = true
		} else if len(workers) > storedAction.lastReplenishedWorkerCount {
			replenish = true
		}

		actionsToReplenish[actionId] = replenish
	}

	s.actionsMu.RUnlock()

	// FUNCTION 2: for each action which should be replenished, load the available slots
	uniqueWorkerIds := make(map[string]bool)

	for actionId, replenish := range actionsToReplenish {
		if !replenish {
			continue
		}

		workerIds := actionsToWorkerIds[actionId]

		for _, workerId := range workerIds {
			uniqueWorkerIds[workerId] = true
		}
	}

	workerUUIDs := make([]pgtype.UUID, 0, len(uniqueWorkerIds))

	for workerId := range uniqueWorkerIds {
		workerUUIDs = append(workerUUIDs, sqlchelpers.UUIDFromStr(workerId))
	}

	availableSlots, err := s.repo.ListAvailableSlotsForWorkers(ctx, dbsqlc.ListAvailableSlotsForWorkersParams{
		Tenantid:  s.tenantId,
		Workerids: workerUUIDs,
	})

	if err != nil {
		return err
	}

	// FUNCTION 3: list unacked slots (so they're not counted towards the worker slot count)
	workersToUnackedSlots := make(map[string][]*slot)

	s.unackedMu.Lock()
	defer s.unackedMu.Unlock()

	for _, unackedSlot := range s.unackedSlots {
		s := unackedSlot
		workerId := s.getWorkerId()

		if _, ok := workersToUnackedSlots[workerId]; !ok {
			workersToUnackedSlots[workerId] = make([]*slot, 0)
		}

		workersToUnackedSlots[workerId] = append(workersToUnackedSlots[workerId], s)
	}

	// FUNCTION 4: write the new slots to the scheduler and clean up expired slots
	actionsToNewSlots := make(map[string][]*slot)
	actionsToTotalSlots := make(map[string]int)

	for _, worker := range availableSlots {
		workerId := sqlchelpers.UUIDToStr(worker.ID)
		actions := workerIdsToActions[workerId]
		unackedSlots := workersToUnackedSlots[workerId]

		// create a slot for each available slot
		slots := make([]*slot, 0)

		for i := 0; i < int(worker.AvailableSlots)-len(unackedSlots); i++ {
			slots = append(slots, newSlot(workers[workerId], actions))
		}

		// extend expiry of all unacked slots
		for _, unackedSlot := range unackedSlots {
			unackedSlot.extendExpiry()
		}

		slots = append(slots, unackedSlots...)

		for _, actionId := range actions {
			actionsToNewSlots[actionId] = append(actionsToNewSlots[actionId], slots...)
			actionsToTotalSlots[actionId] += len(slots)
		}
	}

	s.actionsMu.Lock()
	defer s.actionsMu.Unlock()

	// (we don't need cryptographically secure randomness)
	randSource := rand.New(rand.NewSource(time.Now().UnixNano())) // nolint: gosec

	// first pass: write all actions with new slots to the scheduler
	for actionId, newSlots := range actionsToNewSlots {
		// randomly sort the slots
		randSource.Shuffle(len(newSlots), func(i, j int) { newSlots[i], newSlots[j] = newSlots[j], newSlots[i] })

		if _, ok := s.actions[actionId]; !ok {
			s.actions[actionId] = &action{
				slots:                      newSlots,
				lastReplenishedSlotCount:   len(newSlots),
				lastReplenishedWorkerCount: len(actionsToWorkerIds[actionId]),
			}
		} else {
			// we overwrite the slots for the action
			s.actions[actionId].slots = newSlots
			s.actions[actionId].lastReplenishedSlotCount = actionsToTotalSlots[actionId]
			s.actions[actionId].lastReplenishedWorkerCount = len(actionsToWorkerIds[actionId])
		}
	}

	// second pass: clean up used and expired slots
	for i := range s.actions {
		storedAction := s.actions[i]

		newSlots := make([]*slot, 0, len(storedAction.slots))

		for i := range storedAction.slots {
			slot := storedAction.slots[i]

			if slot.active() {
				newSlots = append(newSlots, slot)
			}
		}

		storedAction.slots = newSlots
	}

	// third pass: remove any actions which have no slots
	for actionId, storedAction := range s.actions {
		if storedAction.activeCount() == 0 {
			delete(s.actions, actionId)
		}
	}

	return nil
}

func (s *Scheduler) loopReplenish(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := s.replenish(ctx, true)

			if err != nil {
				s.l.Error().Err(err).Msg("error replenishing slots")
			}
		}
	}
}

func (s *Scheduler) start(ctx context.Context) {
	go s.loopReplenish(ctx)
}

type scheduleRateLimitResult struct {
	*rateLimitResult

	qi *dbsqlc.QueueItem
}

type assignSingleResult struct {
	qi *dbsqlc.QueueItem

	workerId pgtype.UUID
	ackId    int

	noSlots   bool
	succeeded bool

	rateLimitResult *scheduleRateLimitResult
}

func (s *Scheduler) tryAssignBatch(
	ctx context.Context,
	actionId string,
	qis []*dbsqlc.QueueItem,
	ringOffset int,
	stepIdsToLabels map[string][]*dbsqlc.GetDesiredLabelsRow,
	stepRunIdsToRateLimits map[string]map[string]int32,
) (
	res []*assignSingleResult, newRingOffset int, err error,
) {
	newRingOffset = ringOffset

	ctx, span := telemetry.NewSpan(ctx, "try-assign-batch")
	defer span.End()

	res = make([]*assignSingleResult, len(qis))

	for i := range qis {
		res[i] = &assignSingleResult{
			qi: qis[i],
		}
	}

	rlAcks := make([]func(), len(qis))
	rlNacks := make([]func(), len(qis))

	// first, check rate limits for each of the queue items
	for i := range res {
		r := res[i]
		qi := qis[i]
		var rateLimitAck func()
		var rateLimitNack func()

		rls := make(map[string]int32)

		if stepRunIdsToRateLimits != nil {
			if _, ok := stepRunIdsToRateLimits[sqlchelpers.UUIDToStr(qi.StepRunId)]; ok {
				rls = stepRunIdsToRateLimits[sqlchelpers.UUIDToStr(qi.StepRunId)]
			}
		}

		// check rate limits
		if len(rls) > 0 {
			rlResult := s.rl.use(ctx, sqlchelpers.UUIDToStr(qi.StepRunId), rls)

			if !rlResult.succeeded {
				r.rateLimitResult = &scheduleRateLimitResult{
					rateLimitResult: &rlResult,
					qi:              qi,
				}
			} else {
				rateLimitAck = rlResult.ack
				rateLimitNack = rlResult.nack
			}
		}

		rlAcks[i] = rateLimitAck
		rlNacks[i] = rateLimitNack
	}

	// next, if we're not rate-limited, lock the actions map and try to assign the batch of queue items
	// increment the ring offset for each call when we're not rate limited
	s.actionsMu.RLock()

	if _, ok := s.actions[actionId]; !ok {
		s.actionsMu.RUnlock()

		// if the action is not in the map, then we have no slots to assign to
		for i := range res {
			res[i].noSlots = true
		}

		return res, newRingOffset, nil
	}

	candidateSlots := s.actions[actionId].slots

	for i := range res {
		if res[i].rateLimitResult != nil {
			continue
		}

		qi := qis[i]

		singleRes, err := s.tryAssignSingleton(
			ctx,
			qi,
			candidateSlots,
			ringOffset,
			stepIdsToLabels[sqlchelpers.UUIDToStr(qi.StepId)],
			rlAcks[i],
			rlNacks[i],
		)

		if err != nil {
			s.l.Error().Err(err).Msg("error assigning queue item")
			return res, newRingOffset, err
		}

		res[i] = &singleRes
		res[i].qi = qi
		newRingOffset++
	}

	s.actionsMu.RUnlock()

	return res, newRingOffset, nil
}

// tryAssignSingleton attempts to assign a singleton step to a worker.
func (s *Scheduler) tryAssignSingleton(
	ctx context.Context,
	qi *dbsqlc.QueueItem,
	candidateSlots []*slot,
	// ringOffset is a hint for where to start the search for a slot. The search will wraparound the ring if necessary.
	// If a slot is assigned, the caller should increment this value for the next call to tryAssignSingleton.
	// Note that this is not guaranteed to be the actual offset of the latest assigned slot, since many actions may be scheduling
	// slots concurrently.
	ringOffset int,
	labels []*dbsqlc.GetDesiredLabelsRow,
	rateLimitAck func(),
	rateLimitNack func(),
) (
	res assignSingleResult, err error,
) {
	ctx, span := telemetry.NewSpan(ctx, "try-assign-singleton") // nolint: ineffassign
	defer span.End()

	ringOffset %= len(candidateSlots)

	// rotate the ring to the offset
	candidateSlots = append(candidateSlots[ringOffset:], candidateSlots[:ringOffset]...)

	if qi.Sticky.Valid || len(labels) > 0 {
		candidateSlots = getRankedSlots(qi, labels, candidateSlots)
	}

	var assignedSlot *slot

	for _, slot := range candidateSlots {
		if !slot.active() {
			continue
		}

		if !slot.use([]func(){rateLimitAck}, []func(){rateLimitNack}) {
			continue
		}

		assignedSlot = slot
		break
	}

	if assignedSlot == nil {
		res.noSlots = true
		return res, nil
	}

	s.assignedCountMu.Lock()
	s.assignedCount++
	res.ackId = s.assignedCount
	s.assignedCountMu.Unlock()

	s.unackedMu.Lock()
	s.unackedSlots[res.ackId] = assignedSlot
	s.unackedMu.Unlock()

	res.workerId = sqlchelpers.UUIDFromStr(assignedSlot.getWorkerId())
	res.succeeded = true

	return res, nil
}

type AssignedQueueItem struct {
	AckId    int
	WorkerId pgtype.UUID

	QueueItem *dbsqlc.QueueItem

	// DispatcherId only gets set after a successful flush to the database
	DispatcherId *pgtype.UUID
}

type assignResults struct {
	assigned           []*AssignedQueueItem
	unassigned         []*dbsqlc.QueueItem
	schedulingTimedOut []*dbsqlc.QueueItem
	rateLimited        []*scheduleRateLimitResult
}

func (s *Scheduler) tryAssign(
	ctx context.Context,
	qis []*dbsqlc.QueueItem,
	stepIdsToLabels map[string][]*dbsqlc.GetDesiredLabelsRow,
	stepRunIdsToRateLimits map[string]map[string]int32,
) <-chan *assignResults {
	ctx, span := telemetry.NewSpan(ctx, "try-assign")

	// split into groups based on action ids, and process each action id in parallel
	actionIdToQueueItems := make(map[string][]*dbsqlc.QueueItem)

	for i := range qis {
		qi := qis[i]

		actionId := qi.ActionId.String

		if _, ok := actionIdToQueueItems[actionId]; !ok {
			actionIdToQueueItems[actionId] = make([]*dbsqlc.QueueItem, 0)
		}

		actionIdToQueueItems[actionId] = append(actionIdToQueueItems[actionId], qi)
	}

	resultsCh := make(chan *assignResults, len(actionIdToQueueItems))

	go func() {
		wg := sync.WaitGroup{}

		// process each action id in parallel
		for actionId, qis := range actionIdToQueueItems {
			wg.Add(1)

			go func(actionId string, qis []*dbsqlc.QueueItem) {
				defer wg.Done()
				assigned := make([]*AssignedQueueItem, 0, len(qis))
				unassigned := make([]*dbsqlc.QueueItem, 0, len(qis))
				schedulingTimedOut := make([]*dbsqlc.QueueItem, 0, len(qis))
				rateLimited := make([]*scheduleRateLimitResult, 0, len(qis))

				ringOffset := 0

				batched := make([]*dbsqlc.QueueItem, 0)

				for i := range qis {
					qi := qis[i]

					if isTimedOut(qi) {
						schedulingTimedOut = append(schedulingTimedOut, qi)
						continue
					}

					batched = append(batched, qi)
				}

				err := queueutils.MakeBatched(50, batched, func(batchQis []*dbsqlc.QueueItem) error {
					batchStart := time.Now()

					results, newRingOffset, err := s.tryAssignBatch(ctx, actionId, batchQis, ringOffset, stepIdsToLabels, stepRunIdsToRateLimits)

					if err != nil {
						return err
					}

					ringOffset = newRingOffset

					for _, singleRes := range results {
						if !singleRes.succeeded {
							if singleRes.rateLimitResult != nil {
								rateLimited = append(rateLimited, singleRes.rateLimitResult)
							} else if singleRes.noSlots {
								unassigned = append(unassigned, singleRes.qi)
							}

							continue
						}

						assigned = append(assigned, &AssignedQueueItem{
							WorkerId:  singleRes.workerId,
							QueueItem: singleRes.qi,
							AckId:     singleRes.ackId,
						})
					}

					if sinceStart := time.Since(batchStart); sinceStart > 100*time.Millisecond {
						s.l.Warn().Dur("duration", sinceStart).Msgf("processing batch of %d queue items took longer than 100ms", len(batchQis))
					}

					return nil
				})

				if err != nil {
					s.l.Error().Err(err).Msg("error assigning queue items")
					return
				}

				resultsCh <- &assignResults{
					assigned:           assigned,
					unassigned:         unassigned,
					schedulingTimedOut: schedulingTimedOut,
					rateLimited:        rateLimited,
				}
			}(actionId, qis)
		}

		wg.Wait()
		span.End()
		close(resultsCh)
	}()

	return resultsCh
}

func isTimedOut(qi *dbsqlc.QueueItem) bool {
	// if the current time is after the scheduleTimeoutAt, then mark this as timed out
	now := time.Now().UTC().UTC()
	scheduleTimeoutAt := qi.ScheduleTimeoutAt.Time

	// timed out if the scheduleTimeoutAt is set and the current time is after the scheduleTimeoutAt
	isTimedOut := !scheduleTimeoutAt.IsZero() && scheduleTimeoutAt.Before(now)

	return isTimedOut
}
