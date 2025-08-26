package v1

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/queueutils"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/scheduling/v0/randomticker"
)

const rateLimitedRequeueAfterThreshold = 2 * time.Second

// Scheduler is responsible for scheduling steps to workers as efficiently as possible.
// This is tenant-scoped, so each tenant will have its own scheduler.
type Scheduler struct {
	repo     v1.AssignmentRepository
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

	rl   *rateLimiter
	exts *Extensions
}

func newScheduler(cf *sharedConfig, tenantId pgtype.UUID, rl *rateLimiter, exts *Extensions) *Scheduler {
	l := cf.l.With().Str("tenant_id", sqlchelpers.UUIDToStr(tenantId)).Logger()

	return &Scheduler{
		repo:            cf.repo.Assignment(),
		tenantId:        tenantId,
		l:               &l,
		actions:         make(map[string]*action),
		unackedSlots:    make(map[int]*slot),
		rl:              rl,
		actionsMu:       newRWMu(cf.l),
		replenishMu:     newMu(cf.l),
		workersMu:       newMu(cf.l),
		assignedCountMu: newMu(cf.l),
		unackedMu:       newMu(cf.l),
		exts:            exts,
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

func (s *Scheduler) setWorkers(workers []*v1.ListActiveWorkersResult) {
	s.workersMu.Lock()
	defer s.workersMu.Unlock()

	newWorkers := make(map[string]*worker, len(workers))

	for i := range workers {
		newWorkers[workers[i].ID] = &worker{
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
	if ok := s.replenishMu.TryLock(); !ok {
		s.l.Debug().Msg("skipping replenish because another replenish is in progress")
		return nil
	}

	defer s.replenishMu.Unlock()

	// we get a lock on the actions mutexes here because we want to acquire the locks in the same order
	// as the tryAssignBatch function. otherwise, we could deadlock when tryAssignBatch has a lock
	// on the actionsMu and tries to acquire the unackedMu lock.
	// additionally, we have to acquire a lock this early (before the database read) to prevent slots
	// from being assigned while we read slots from the database.
	if mustReplenish {
		s.actionsMu.Lock()
	} else if ok := s.actionsMu.TryLock(); !ok {
		s.l.Debug().Msg("skipping replenish because we can't acquire the actions mutex")
		return nil
	}

	defer s.actionsMu.Unlock()

	s.l.Debug().Msg("replenishing slots")

	workers := s.getWorkers()
	workerIds := make([]pgtype.UUID, 0)

	for workerIdStr := range workers {
		workerIds = append(workerIds, sqlchelpers.UUIDFromStr(workerIdStr))
	}

	start := time.Now()
	checkpoint := start

	workersToActiveActions, err := s.repo.ListActionsForWorkers(ctx, s.tenantId, workerIds)

	if err != nil {
		return err
	}

	if sinceStart := time.Since(start); sinceStart > 100*time.Millisecond {
		s.l.Warn().Msgf("listing actions for workers took %s for %d workers", time.Since(checkpoint), len(workerIds))
	} else {
		s.l.Debug().Msgf("listing actions for workers took %s", time.Since(checkpoint))
	}

	checkpoint = time.Now()

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
	actionsToReplenish := make(map[string]*action)

	for actionId, workers := range actionsToWorkerIds {
		// if the action is not in the map, it should be replenished
		if _, ok := s.actions[actionId]; !ok {
			newAction := &action{
				actionId: actionId,
			}

			actionsToReplenish[actionId] = newAction

			s.actions[actionId] = newAction

			continue
		}

		if mustReplenish {
			actionsToReplenish[actionId] = s.actions[actionId]

			continue
		}

		storedAction := s.actions[actionId]

		// determine if we match the conditions above
		var replenish bool
		activeCount := storedAction.activeCount()

		switch {
		case activeCount == 0:
			s.l.Debug().Msgf("replenishing all slots for action %s because activeCount is 0", actionId)
			replenish = true
		case activeCount <= (storedAction.lastReplenishedSlotCount / 2):
			s.l.Debug().Msgf("replenishing slots for action %s because 50%% of slots have been used", actionId)
			replenish = true
		case len(workers) > storedAction.lastReplenishedWorkerCount:
			s.l.Debug().Msgf("replenishing slots for action %s because more workers are available", actionId)
			replenish = true
		}

		if replenish {
			actionsToReplenish[actionId] = s.actions[actionId]
		}
	}

	// if there are any workers which have additional actions not in the actionsToReplenish map, we need
	// to add them to the actionsToReplenish map
	for actionId := range actionsToReplenish {
		for _, workerId := range actionsToWorkerIds[actionId] {
			for _, actions := range workerIdsToActions[workerId] {
				if _, ok := actionsToReplenish[actions]; !ok {
					actionsToReplenish[actions] = s.actions[actions]
				}
			}
		}
	}

	s.l.Debug().Msgf("determining which actions to replenish took %s", time.Since(checkpoint))
	checkpoint = time.Now()

	// FUNCTION 2: for each action which should be replenished, load the available slots
	uniqueWorkerIds := make(map[string]bool)

	for actionId := range actionsToReplenish {
		workerIds := actionsToWorkerIds[actionId]

		for _, workerId := range workerIds {
			uniqueWorkerIds[workerId] = true
		}
	}

	workerUUIDs := make([]pgtype.UUID, 0, len(uniqueWorkerIds))

	for workerId := range uniqueWorkerIds {
		workerUUIDs = append(workerUUIDs, sqlchelpers.UUIDFromStr(workerId))
	}

	orderedLock(actionsToReplenish)
	unlock := orderedUnlock(actionsToReplenish)
	defer unlock()

	s.unackedMu.Lock()
	defer s.unackedMu.Unlock()

	availableSlots, err := s.repo.ListAvailableSlotsForWorkers(ctx, s.tenantId, sqlcv1.ListAvailableSlotsForWorkersParams{
		Tenantid:  s.tenantId,
		Workerids: workerUUIDs,
	})

	if err != nil {
		return err
	}

	s.l.Debug().Msgf("loading available slots took %s", time.Since(checkpoint))

	// FUNCTION 3: list unacked slots (so they're not counted towards the worker slot count)
	workersToUnackedSlots := make(map[string][]*slot)

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

		s.l.Debug().Msgf("worker %s has %d total slots, %d unacked slots", workerId, worker.AvailableSlots, len(unackedSlots))

		slots = append(slots, unackedSlots...)

		for _, actionId := range actions {
			actionsToNewSlots[actionId] = append(actionsToNewSlots[actionId], slots...)
			actionsToTotalSlots[actionId] += len(slots)
		}
	}

	// (we don't need cryptographically secure randomness)
	randSource := rand.New(rand.NewSource(time.Now().UnixNano())) // nolint: gosec

	// first pass: write all actions with new slots to the scheduler
	for actionId, newSlots := range actionsToNewSlots {
		// randomly sort the slots
		randSource.Shuffle(len(newSlots), func(i, j int) { newSlots[i], newSlots[j] = newSlots[j], newSlots[i] })

		// we overwrite the slots for the action. we know that the action is in the map because we checked
		// for it in the first pass.
		s.actions[actionId].slots = newSlots
		s.actions[actionId].lastReplenishedSlotCount = actionsToTotalSlots[actionId]
		s.actions[actionId].lastReplenishedWorkerCount = len(actionsToWorkerIds[actionId])

		s.l.Debug().Msgf("before cleanup, action %s has %d slots", actionId, len(newSlots))
	}

	// second pass: clean up expired slots
	for _, storedAction := range actionsToReplenish {
		newSlots := make([]*slot, 0, len(storedAction.slots))

		for i := range storedAction.slots {
			slot := storedAction.slots[i]

			if !slot.expired() {
				newSlots = append(newSlots, slot)
			}
		}

		storedAction.slots = newSlots

		s.l.Debug().Msgf("after cleanup, action %s has %d slots", storedAction.actionId, len(newSlots))
	}

	// third pass: remove any actions which have no slots
	for actionId, storedAction := range actionsToReplenish {
		if len(storedAction.slots) == 0 {
			s.l.Debug().Msgf("removing action %s because it has no slots", actionId)
			delete(s.actions, actionId)
		}
	}

	if sinceStart := time.Since(start); sinceStart > 100*time.Millisecond {
		s.l.Warn().Dur("duration", sinceStart).Msg("replenishing slots took longer than 100ms")
	} else {
		s.l.Debug().Dur("duration", sinceStart).Msgf("finished replenishing slots")
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

func (s *Scheduler) loopSnapshot(ctx context.Context) {
	ticker := randomticker.NewRandomTicker(10*time.Millisecond, 90*time.Millisecond)
	defer ticker.Stop()

	for {
		count := 0

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// require that 1 out of every 20 snapshots is taken
			must := count%20 == 0

			in, ok := s.getSnapshotInput(must)

			if !ok {
				continue
			}

			s.exts.ReportSnapshot(sqlchelpers.UUIDToStr(s.tenantId), in)
		}
	}
}

func (s *Scheduler) start(ctx context.Context) {
	go s.loopReplenish(ctx)
	go s.loopSnapshot(ctx)
}

type scheduleRateLimitResult struct {
	*rateLimitResult

	qi *sqlcv1.V1QueueItem
}

// shouldRemoveFromQueue returns true if the queue item is being rate limited and should be removed from the queue
// until the rate limit is reset.
// we only do this if the requeue_after time is at least 2 seconds in the future, to avoid thrashing
func (s *scheduleRateLimitResult) shouldRemoveFromQueue() bool {
	if s.rateLimitResult == nil {
		return false
	}

	nextRefillAt := s.nextRefillAt

	return nextRefillAt != nil && nextRefillAt.UTC().After(time.Now().UTC().Add(rateLimitedRequeueAfterThreshold))
}

type assignSingleResult struct {
	qi *sqlcv1.V1QueueItem

	workerId pgtype.UUID
	ackId    int

	noSlots   bool
	succeeded bool

	rateLimitResult *scheduleRateLimitResult
}

func (s *Scheduler) tryAssignBatch(
	ctx context.Context,
	actionId string,
	qis []*sqlcv1.V1QueueItem,
	// ringOffset is a hint for where to start the search for a slot. The search will wraparound the ring if necessary.
	// If a slot is assigned, the caller should increment this value for the next call to tryAssignSingleton.
	// Note that this is not guaranteed to be the actual offset of the latest assigned slot, since many actions may be scheduling
	// slots concurrently.
	ringOffset int,
	stepIdsToLabels map[string][]*sqlcv1.GetDesiredLabelsRow,
	taskIdsToRateLimits map[int64]map[string]int32,
) (
	res []*assignSingleResult, newRingOffset int, err error,
) {
	s.l.Debug().Msgf("trying to assign %d queue items", len(qis))

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

	noop := func() {}

	// first, check rate limits for each of the queue items
	// TODO: REVERT
	for i := range res {
		r := res[i]
		qi := qis[i]

		rateLimitAck := noop
		rateLimitNack := noop

		rls := make(map[string]int32)

		if taskIdsToRateLimits != nil {
			if _, ok := taskIdsToRateLimits[qi.TaskID]; ok {
				rls = taskIdsToRateLimits[qi.TaskID]
			}
		}

		// check rate limits
		if len(rls) > 0 {
			rlResult := s.rl.use(ctx, qi.TaskID, rls)

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

	// lock the actions map and try to assign the batch of queue items.
	// NOTE: if we change the position of this lock, make sure that we are still acquiring locks in the same
	// order as the replenish() function, otherwise we may deadlock.
	s.actionsMu.RLock()

	action, ok := s.actions[actionId]

	if !ok || len(action.slots) == 0 {
		s.actionsMu.RUnlock()

		s.l.Debug().Msgf("no slots for action %s", actionId)

		// if the action is not in the map, then we have no slots to assign to
		for i := range res {
			res[i].noSlots = true
			rlNacks[i]()
		}

		return res, newRingOffset, nil
	}

	s.actionsMu.RUnlock()

	action.mu.Lock()
	defer action.mu.Unlock()

	candidateSlots := action.slots

	for i := range res {
		if res[i].rateLimitResult != nil {
			continue
		}

		denom := len(candidateSlots)

		if denom == 0 {
			res[i].noSlots = true
			rlNacks[i]()

			continue
		}

		childRingOffset := newRingOffset % denom

		qi := qis[i]

		singleRes, err := s.tryAssignSingleton(
			ctx,
			qi,
			candidateSlots,
			childRingOffset,
			stepIdsToLabels[sqlchelpers.UUIDToStr(qi.StepID)],
			rlAcks[i],
			rlNacks[i],
		)

		if err != nil {
			s.l.Error().Err(err).Msg("error assigning queue item")
		}

		if !singleRes.succeeded {
			rlNacks[i]()
		}

		res[i] = &singleRes
		res[i].qi = qi

		newRingOffset++
	}

	return res, newRingOffset, nil
}

func findSlot(
	candidateSlots []*slot,
	rateLimitAck func(),
	rateLimitNack func(),
) *slot {
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

	return assignedSlot
}

// tryAssignSingleton attempts to assign a singleton step to a worker.
func (s *Scheduler) tryAssignSingleton(
	ctx context.Context,
	qi *sqlcv1.V1QueueItem,
	candidateSlots []*slot,
	ringOffset int,
	labels []*sqlcv1.GetDesiredLabelsRow,
	rateLimitAck func(),
	rateLimitNack func(),
) (
	res assignSingleResult, err error,
) {
	ctx, span := telemetry.NewSpan(ctx, "try-assign-singleton") // nolint: ineffassign
	defer span.End()

	ringOffset = ringOffset % len(candidateSlots)

	if (qi.Sticky != sqlcv1.V1StickyStrategyNONE) || len(labels) > 0 {
		candidateSlots = getRankedSlots(qi, labels, candidateSlots)
		ringOffset = 0
	}

	assignedSlot := findSlot(candidateSlots[ringOffset:], rateLimitAck, rateLimitNack)

	if assignedSlot == nil {
		assignedSlot = findSlot(candidateSlots[:ringOffset], rateLimitAck, rateLimitNack)
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

type assignedQueueItem struct {
	AckId    int
	WorkerId pgtype.UUID

	QueueItem *sqlcv1.V1QueueItem
}

type assignResults struct {
	assigned           []*assignedQueueItem
	unassigned         []*sqlcv1.V1QueueItem
	schedulingTimedOut []*sqlcv1.V1QueueItem
	rateLimited        []*scheduleRateLimitResult
	rateLimitedToMove  []*scheduleRateLimitResult
}

func (s *Scheduler) tryAssign(
	ctx context.Context,
	qis []*sqlcv1.V1QueueItem,
	stepIdsToLabels map[string][]*sqlcv1.GetDesiredLabelsRow,
	taskIdsToRateLimits map[int64]map[string]int32,
) <-chan *assignResults {
	ctx, span := telemetry.NewSpan(ctx, "try-assign")

	// split into groups based on action ids, and process each action id in parallel
	actionIdToQueueItems := make(map[string][]*sqlcv1.V1QueueItem)

	for i := range qis {
		qi := qis[i]

		actionId := qi.ActionID

		if _, ok := actionIdToQueueItems[actionId]; !ok {
			actionIdToQueueItems[actionId] = make([]*sqlcv1.V1QueueItem, 0)
		}

		actionIdToQueueItems[actionId] = append(actionIdToQueueItems[actionId], qi)
	}

	resultsCh := make(chan *assignResults, len(actionIdToQueueItems))

	go func() {
		wg := sync.WaitGroup{}
		startTotal := time.Now()

		extensionResults := make([]*assignResults, 0)
		extensionResultsMu := sync.Mutex{}

		// process each action id in parallel
		for actionId, qis := range actionIdToQueueItems {
			wg.Add(1)

			go func(actionId string, qis []*sqlcv1.V1QueueItem) {
				defer wg.Done()

				ringOffset := 0

				batched := make([]*sqlcv1.V1QueueItem, 0)
				schedulingTimedOut := make([]*sqlcv1.V1QueueItem, 0, len(qis))

				for i := range qis {
					qi := qis[i]

					if isTimedOut(qi) {
						schedulingTimedOut = append(schedulingTimedOut, qi)
						continue
					}

					batched = append(batched, qi)
				}

				resultsCh <- &assignResults{
					schedulingTimedOut: schedulingTimedOut,
				}

				err := queueutils.BatchLinear(50, batched, func(batchQis []*sqlcv1.V1QueueItem) error {
					batchAssigned := make([]*assignedQueueItem, 0, len(batchQis))
					batchRateLimited := make([]*scheduleRateLimitResult, 0, len(batchQis))
					batchRateLimitedToMove := make([]*scheduleRateLimitResult, 0, len(batchQis))
					batchUnassigned := make([]*sqlcv1.V1QueueItem, 0, len(batchQis))

					batchStart := time.Now()

					results, newRingOffset, err := s.tryAssignBatch(ctx, actionId, batchQis, ringOffset, stepIdsToLabels, taskIdsToRateLimits)

					if err != nil {
						return err
					}

					ringOffset = newRingOffset

					for _, singleRes := range results {
						if !singleRes.succeeded {
							if singleRes.rateLimitResult != nil {
								if singleRes.rateLimitResult.shouldRemoveFromQueue() {

									batchRateLimitedToMove = append(batchRateLimitedToMove, singleRes.rateLimitResult)
								} else {
									batchRateLimited = append(batchRateLimited, singleRes.rateLimitResult)
								}
							} else {
								batchUnassigned = append(batchUnassigned, singleRes.qi)

								if !singleRes.noSlots {
									s.l.Error().Msgf("scheduling failed for queue item %d: expected assignment to fail with either no slots or rate limit exceeded, but failed with neither", singleRes.qi.ID)
								}
							}

							continue
						}

						batchAssigned = append(batchAssigned, &assignedQueueItem{
							WorkerId:  singleRes.workerId,
							QueueItem: singleRes.qi,
							AckId:     singleRes.ackId,
						})
					}

					if sinceStart := time.Since(batchStart); sinceStart > 100*time.Millisecond {
						s.l.Warn().Dur("duration", sinceStart).Msgf("processing batch of %d queue items took longer than 100ms", len(batchQis))
					}

					r := &assignResults{
						assigned:          batchAssigned,
						rateLimited:       batchRateLimited,
						rateLimitedToMove: batchRateLimitedToMove,
						unassigned:        batchUnassigned,
					}

					extensionResultsMu.Lock()
					extensionResults = append(extensionResults, r)
					extensionResultsMu.Unlock()

					resultsCh <- r

					return nil
				})

				if err != nil {
					s.l.Error().Err(err).Msg("error assigning queue items")
				}
			}(actionId, qis)
		}

		wg.Wait()
		span.End()
		close(resultsCh)

		s.exts.PostAssign(sqlchelpers.UUIDToStr(s.tenantId), s.getExtensionInput(extensionResults))

		if sinceStart := time.Since(startTotal); sinceStart > 100*time.Millisecond {
			s.l.Warn().Dur("duration", sinceStart).Msgf("assigning queue items took longer than 100ms")
		}
	}()

	return resultsCh
}

func (s *Scheduler) getExtensionInput(results []*assignResults) *PostAssignInput {
	unassigned := make([]*sqlcv1.V1QueueItem, 0)

	for _, res := range results {
		unassigned = append(unassigned, res.unassigned...)
	}

	return &PostAssignInput{
		HasUnassignedStepRuns: len(unassigned) > 0,
	}
}

func (s *Scheduler) getSnapshotInput(mustSnapshot bool) (*SnapshotInput, bool) {
	if mustSnapshot {
		s.actionsMu.RLock()
	} else {
		if ok := s.actionsMu.TryRLock(); !ok {
			return nil, false
		}
	}

	defer s.actionsMu.RUnlock()

	workers := s.getWorkers()

	res := &SnapshotInput{
		Workers: make(map[string]*WorkerCp),
	}

	for workerId, worker := range workers {
		res.Workers[workerId] = &WorkerCp{
			WorkerId: workerId,
			MaxRuns:  worker.MaxRuns,
			Labels:   worker.Labels,
		}
	}

	// NOTE: these locks are important because we must acquire locks in the same order as the replenish and tryAssignBatch
	// functions. we always acquire actionsMu first and then the specific action's lock.
	actionKeys := make([]string, 0, len(s.actions))

	for actionId := range s.actions {
		actionKeys = append(actionKeys, actionId)
	}

	uniqueSlots := make(map[*slot]bool)

	workerSlotUtilization := make(map[string]*SlotUtilization)

	for workerId := range workers {
		workerSlotUtilization[workerId] = &SlotUtilization{
			UtilizedSlots:    0,
			NonUtilizedSlots: 0,
		}
	}

	for _, actionId := range actionKeys {
		action, ok := s.actions[actionId]

		if !ok || action == nil {
			continue
		}

		action.mu.RLock()
		for _, slot := range action.slots {
			if _, ok := uniqueSlots[slot]; ok {
				continue
			}

			workerId := slot.worker.ID

			if _, ok := workerSlotUtilization[workerId]; !ok {
				// initialize the worker slot utilization
				workerSlotUtilization[workerId] = &SlotUtilization{
					UtilizedSlots:    0,
					NonUtilizedSlots: 0,
				}
			}

			uniqueSlots[slot] = true

			if slot.isUsed() {
				workerSlotUtilization[workerId].UtilizedSlots++
			} else {
				workerSlotUtilization[workerId].NonUtilizedSlots++
			}
		}
		action.mu.RUnlock()
	}

	res.WorkerSlotUtilization = workerSlotUtilization

	return res, true
}

func isTimedOut(qi *sqlcv1.V1QueueItem) bool {
	// if the current time is after the scheduleTimeoutAt, then mark this as timed out
	now := time.Now().UTC().UTC()
	scheduleTimeoutAt := qi.ScheduleTimeoutAt.Time

	// timed out if the scheduleTimeoutAt is set and the current time is after the scheduleTimeoutAt
	isTimedOut := !scheduleTimeoutAt.IsZero() && scheduleTimeoutAt.Before(now)

	return isTimedOut
}
