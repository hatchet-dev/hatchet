package v1

import (
	"context"
	"fmt"
	"math/rand"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/queueutils"
	"github.com/hatchet-dev/hatchet/pkg/randomticker"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

const rateLimitedRequeueAfterThreshold = 2 * time.Second

// Scheduler is responsible for scheduling steps to workers as efficiently as possible.
// This is tenant-scoped, so each tenant will have its own scheduler.
type Scheduler struct {
	repo     v1.AssignmentRepository
	tenantId uuid.UUID

	l *zerolog.Logger

	actions     map[string]*action
	actionsMu   rwMutex
	replenishMu mutex

	workersMu mutex
	workers   map[uuid.UUID]*worker

	assignedCount   int
	assignedCountMu mutex

	// unackedSlots are slots which have been assigned to a worker, but have not been flushed
	// to the database yet. They negatively count towards a worker's available slot count.
	unackedSlots map[int]*assignedSlots
	unackedMu    mutex

	rl   *rateLimiter
	exts *Extensions
}

func newScheduler(cf *sharedConfig, tenantId uuid.UUID, rl *rateLimiter, exts *Extensions) *Scheduler {
	l := cf.l.With().Str("tenant_id", tenantId.String()).Logger()

	return &Scheduler{
		repo:            cf.repo.Assignment(),
		tenantId:        tenantId,
		l:               &l,
		actions:         make(map[string]*action),
		unackedSlots:    make(map[int]*assignedSlots),
		rl:              rl,
		actionsMu:       newRWMu(cf.l),
		replenishMu:     newMu(cf.l),
		workers:         map[uuid.UUID]*worker{},
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
		if assigned, ok := s.unackedSlots[id]; ok {
			assigned.ack()
			delete(s.unackedSlots, id)
		}
	}
}

func (s *Scheduler) nack(ids []int) {
	s.unackedMu.Lock()
	defer s.unackedMu.Unlock()

	for _, id := range ids {
		if assigned, ok := s.unackedSlots[id]; ok {
			assigned.nack()
			delete(s.unackedSlots, id)
		}
	}
}

func (s *Scheduler) setWorkers(workers []*v1.ListActiveWorkersResult) {
	s.workersMu.Lock()
	defer s.workersMu.Unlock()

	newWorkers := make(map[uuid.UUID]*worker, len(workers))

	for i := range workers {
		newWorkers[workers[i].ID] = &worker{
			ListActiveWorkersResult: workers[i],
		}
	}

	s.workers = newWorkers
}

func (s *Scheduler) addWorker(newWorker *v1.ListActiveWorkersResult) {
	s.workersMu.Lock()
	defer s.workersMu.Unlock()

	s.workers[newWorker.ID] = &worker{
		ListActiveWorkersResult: newWorker,
	}
}

func (s *Scheduler) copyWorkers() map[uuid.UUID]*worker {
	s.workersMu.Lock()
	defer s.workersMu.Unlock()

	copied := make(map[uuid.UUID]*worker, len(s.workers))

	for k, v := range s.workers {
		copied[k] = v
	}

	return copied
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

	workers := s.copyWorkers()
	workerIds := make([]uuid.UUID, 0)

	for workerId := range workers {
		workerIds = append(workerIds, workerId)
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

	actionsToWorkerIds := make(map[string][]uuid.UUID)
	workerIdsToActions := make(map[uuid.UUID][]string)

	for _, workerActionTuple := range workersToActiveActions {
		if !workerActionTuple.ActionId.Valid {
			continue
		}

		actionId := workerActionTuple.ActionId.String
		workerId := workerActionTuple.WorkerId

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
				actionId:               actionId,
				slotsByTypeAndWorkerId: make(map[string]map[uuid.UUID][]*slot),
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
	workerSlotConfigs, err := s.repo.ListWorkerSlotConfigs(ctx, s.tenantId, workerIds)
	if err != nil {
		return err
	}

	workerSlotTypes := make(map[uuid.UUID]map[string]bool, len(workerSlotConfigs))
	slotTypeToWorkerIds := make(map[string]map[uuid.UUID]bool)

	for _, config := range workerSlotConfigs {
		if _, ok := workerSlotTypes[config.WorkerID]; !ok {
			workerSlotTypes[config.WorkerID] = make(map[string]bool)
		}

		workerSlotTypes[config.WorkerID][config.SlotType] = true

		if _, ok := slotTypeToWorkerIds[config.SlotType]; !ok {
			slotTypeToWorkerIds[config.SlotType] = make(map[uuid.UUID]bool)
		}

		slotTypeToWorkerIds[config.SlotType][config.WorkerID] = true
	}

	// We may update slots for any action that is active on a worker with slot capacity.
	// Since tryAssignBatch can hold action.mu without holding actionsMu, we must lock every
	// action we might write to here (not just the subset that triggered a replenish).
	actionsToLock := make(map[string]*action)
	for _, workerSet := range slotTypeToWorkerIds {
		for workerId := range workerSet {
			for _, actionId := range workerIdsToActions[workerId] {
				if a := s.actions[actionId]; a != nil {
					actionsToLock[actionId] = a
				}
			}
		}
	}

	orderedLock(actionsToLock)
	unlock := orderedUnlock(actionsToLock)
	defer unlock()

	s.unackedMu.Lock()
	defer s.unackedMu.Unlock()

	availableSlotsByType := make(map[string]map[uuid.UUID]int, len(slotTypeToWorkerIds))

	slotTypes := make([]string, 0, len(slotTypeToWorkerIds))
	workerUUIDSet := make(map[uuid.UUID]struct{})

	for slotType, workerSet := range slotTypeToWorkerIds {
		slotTypes = append(slotTypes, slotType)

		// Preserve the prior behavior of creating a map entry per slot type even if it ends up empty.
		if _, ok := availableSlotsByType[slotType]; !ok {
			availableSlotsByType[slotType] = make(map[uuid.UUID]int, len(workerSet))
		}

		for workerId := range workerSet {
			workerUUIDSet[workerId] = struct{}{}
		}
	}

	if len(slotTypes) > 0 && len(workerUUIDSet) > 0 {
		workerUUIDs := make([]uuid.UUID, 0, len(workerUUIDSet))
		for workerId := range workerUUIDSet {
			workerUUIDs = append(workerUUIDs, workerId)
		}

		availableSlots, err := s.repo.ListAvailableSlotsForWorkersAndTypes(ctx, s.tenantId, sqlcv1.ListAvailableSlotsForWorkersAndTypesParams{
			Tenantid:  s.tenantId,
			Workerids: workerUUIDs,
			Slottypes: slotTypes,
		})
		if err != nil {
			return err
		}

		for _, row := range availableSlots {
			if _, ok := availableSlotsByType[row.SlotType]; !ok {
				availableSlotsByType[row.SlotType] = make(map[uuid.UUID]int)
			}

			availableSlotsByType[row.SlotType][row.ID] = int(row.AvailableSlots)
		}
	}

	s.l.Debug().Msgf("loading available slots took %s", time.Since(checkpoint))

	// FUNCTION 3: list unacked slots (so they're not counted towards the worker slot count)
	workersToUnackedSlots := make(map[uuid.UUID]map[string][]*slot)

	for _, unackedSlot := range s.unackedSlots {
		for _, assignedSlot := range unackedSlot.slots {
			workerId := assignedSlot.getWorkerId()

			slotType, err := assignedSlot.getSlotType()
			if err != nil {
				return fmt.Errorf("could not get slot type for unacked slot: %w", err)
			}

			if _, ok := workersToUnackedSlots[workerId]; !ok {
				workersToUnackedSlots[workerId] = make(map[string][]*slot)
			}

			workersToUnackedSlots[workerId][slotType] = append(workersToUnackedSlots[workerId][slotType], assignedSlot)
		}
	}

	// FUNCTION 4: write the new slots to the scheduler and clean up expired slots
	actionsToNewSlots := make(map[string][]*slot)
	actionsToTotalSlots := make(map[string]int)
	actionsToSlotsByType := make(map[string]map[string]map[uuid.UUID][]*slot)

	// metaCache interns slotMeta so slots with identical metadata share the same pointer.
	// Key format: slotType + "\x00" + strings.Join(sortedUniqueActions, "\x00")
	metaCache := make(map[string]*slotMeta)

	for slotType, availableSlotsByWorker := range availableSlotsByType {
		for workerId, availableSlots := range availableSlotsByWorker {
			actions := workerIdsToActions[workerId]
			unackedSlots := workersToUnackedSlots[workerId][slotType]

			// create a slot for each available slot
			slots := make([]*slot, 0)
			availableCount := availableSlots - len(unackedSlots)
			if availableCount < 0 {
				availableCount = 0
			}

			// Canonicalize actions to increase cache hits across workers.
			// Order doesn't matter for correctness anywhere in scheduling.
			if len(actions) > 1 {
				slices.Sort(actions)
				actions = slices.Compact(actions)
			}

			metaKey := slotType
			if len(actions) > 0 {
				metaKey = slotType + "\x00" + strings.Join(actions, "\x00")
			}

			meta := metaCache[metaKey]
			if meta == nil {
				meta = newSlotMeta(actions, slotType)
				metaCache[metaKey] = meta
			}

			for i := 0; i < availableCount; i++ {
				slots = append(slots, newSlot(workers[workerId], meta))
			}

			// extend expiry of all unacked slots
			for _, unackedSlot := range unackedSlots {
				unackedSlot.extendExpiry()
			}

			s.l.Debug().Msgf("worker %s has %d total slots (%s), %d unacked slots", workerId, availableSlots, slotType, len(unackedSlots))

			slots = append(slots, unackedSlots...)

			for _, actionId := range actions {
				if s.actions[actionId] == nil {
					continue
				}

				actionsToNewSlots[actionId] = append(actionsToNewSlots[actionId], slots...)
				actionsToTotalSlots[actionId] += len(slots)

				if _, ok := actionsToSlotsByType[actionId]; !ok {
					actionsToSlotsByType[actionId] = make(map[string]map[uuid.UUID][]*slot)
				}
				if _, ok := actionsToSlotsByType[actionId][slotType]; !ok {
					actionsToSlotsByType[actionId][slotType] = make(map[uuid.UUID][]*slot)
				}
				// Reuse the per-worker/per-type slice for each action on that worker.
				actionsToSlotsByType[actionId][slotType][workerId] = slots
			}
		}
	}

	// (we don't need cryptographically secure randomness)
	randSource := rand.New(rand.NewSource(time.Now().UnixNano())) // nolint: gosec

	// first pass: write all actions with new slots to the scheduler
	for actionId, newSlots := range actionsToNewSlots {
		storedAction := actionsToLock[actionId]
		if storedAction == nil {
			// Defensive: actionsToNewSlots should only contain actions for workers we locked above.
			continue
		}

		// randomly sort the slots
		randSource.Shuffle(len(newSlots), func(i, j int) { newSlots[i], newSlots[j] = newSlots[j], newSlots[i] })

		// we overwrite the slots for the action. we know that the action is in the map because we checked
		// for it in the first pass.
		storedAction.slots = newSlots
		storedAction.slotsByTypeAndWorkerId = actionsToSlotsByType[actionId]
		storedAction.lastReplenishedSlotCount = actionsToTotalSlots[actionId]
		storedAction.lastReplenishedWorkerCount = len(actionsToWorkerIds[actionId])

		s.l.Debug().Msgf("before cleanup, action %s has %d slots", actionId, len(newSlots))
	}

	// second pass: clean up expired slots
	for _, storedAction := range actionsToReplenish {
		newSlots := make([]*slot, 0, len(storedAction.slots))

		for i := range storedAction.slots {
			slotItem := storedAction.slots[i]

			if !slotItem.expired() {
				newSlots = append(newSlots, slotItem)
			}
		}

		storedAction.slots = newSlots
		storedAction.slotsByTypeAndWorkerId = make(map[string]map[uuid.UUID][]*slot)

		for _, slotItem := range newSlots {
			slotType, err := slotItem.getSlotType()
			if err != nil {
				return fmt.Errorf("could not get slot type during cleanup: %w", err)
			}

			workerId := slotItem.getWorkerId()

			if _, ok := storedAction.slotsByTypeAndWorkerId[slotType]; !ok {
				storedAction.slotsByTypeAndWorkerId[slotType] = make(map[uuid.UUID][]*slot)
			}

			storedAction.slotsByTypeAndWorkerId[slotType][workerId] = append(storedAction.slotsByTypeAndWorkerId[slotType][workerId], slotItem)
		}

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
			innerCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			err := s.replenish(innerCtx, true)

			if err != nil {
				s.l.Error().Err(err).Msg("error replenishing slots")
			}
			cancel()
		}

	}
}

func (s *Scheduler) loopSnapshot(ctx context.Context) {
	ticker := randomticker.NewRandomTicker(10*time.Millisecond, 90*time.Millisecond)
	defer ticker.Stop()

	count := 0
	for {

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

			s.exts.ReportSnapshot(s.tenantId, in)

			count++
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

	workerId uuid.UUID
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
	stepIdsToLabels map[uuid.UUID][]*sqlcv1.GetDesiredLabelsRow,
	stepIdsToRequests map[uuid.UUID]map[string]int32,
	taskIdsToRateLimits map[int64]map[string]int32,
) (
	res []*assignSingleResult, newRingOffset int, err error,
) {
	s.l.Debug().Msgf("trying to assign %d queue items", len(qis))

	newRingOffset = ringOffset

	ctx, span := telemetry.NewSpan(ctx, "try-assign-batch")
	defer span.End()

	if len(qis) > 0 {
		uniqueTenantIds := telemetry.CollectUniqueTenantIDs(qis, func(qi *sqlcv1.V1QueueItem) string {
			return qi.TenantID.String()
		})
		telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "tenant.id", Value: uniqueTenantIds})
	}

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
	s.actionsMu.RUnlock()

	if !ok || action == nil {
		s.l.Debug().Msgf("no action %s", actionId)

		// Treat missing action as "no slots" for any non-rate-limited queue item.
		for i := range res {
			if res[i].rateLimitResult != nil {
				continue
			}
			res[i].noSlots = true
			rlNacks[i]()
		}

		return res, newRingOffset, nil
	}

	action.mu.RLock()
	if len(action.slots) == 0 {
		action.mu.RUnlock()

		s.l.Debug().Msgf("no slots for action %s", actionId)

		// if the action is not in the map, then we have no slots to assign to
		for i := range res {
			if res[i].rateLimitResult != nil {
				continue
			}
			res[i].noSlots = true
			rlNacks[i]()
		}

		return res, newRingOffset, nil
	}
	action.mu.RUnlock()

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

		labels := []*sqlcv1.GetDesiredLabelsRow(nil)
		if stepIdsToLabels != nil {
			labels = stepIdsToLabels[qi.StepID]
		}

		// Backwards-compatible default: if no slot requests are provided for a step,
		// assume it needs 1 default slot.
		requests := map[string]int32{v1.SlotTypeDefault: 1}
		if stepIdsToRequests != nil {
			if r, ok := stepIdsToRequests[qi.StepID]; ok && len(r) > 0 {
				requests = r
			}
		}

		singleRes, err := s.tryAssignSingleton(
			ctx,
			qi,
			action,
			candidateSlots,
			childRingOffset,
			labels,
			requests,
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

func findAssignableSlots(
	candidateSlots []*slot,
	action *action,
	requests map[string]int32,
	rateLimitAck func(),
	rateLimitNack func(),
) *assignedSlots {
	// NOTE: the caller must hold action.mu (RLock or Lock) while calling this
	// function. We read from action.slots, which is replaced during replenish
	// under action.mu.
	seenWorkers := make(map[uuid.UUID]struct{})

	for _, candidateSlot := range candidateSlots {
		if !candidateSlot.active() {
			continue
		}

		workerId := candidateSlot.getWorkerId()
		if _, seen := seenWorkers[workerId]; seen {
			continue
		}
		seenWorkers[workerId] = struct{}{}

		selected, ok := selectSlotsForWorker(action.slotsByTypeAndWorkerId, workerId, requests)
		if !ok {
			continue
		}

		usedSlots, ok := useSelectedSlots(selected)
		if !ok {
			continue
		}

		// Rate limit callbacks are stored at assignedSlots level,
		// not on individual slots. They're called once when the
		// entire assignment is acked/nacked.
		return &assignedSlots{
			slots:         usedSlots,
			rateLimitAck:  rateLimitAck,
			rateLimitNack: rateLimitNack,
		}
	}

	return nil
}

// useSelectedSlots attempts to reserve each slot in order. If any slot cannot be
// reserved, it rolls back by nacking any slots already reserved in this call.
func useSelectedSlots(selected []*slot) ([]*slot, bool) {
	usedSlots := make([]*slot, 0, len(selected))

	for _, sl := range selected {
		if !sl.use(nil, nil) {
			for _, used := range usedSlots {
				used.nack()
			}
			return nil, false
		}
		usedSlots = append(usedSlots, sl)
	}

	return usedSlots, true
}

func selectSlotsForWorker(
	slotsByType map[string]map[uuid.UUID][]*slot,
	workerId uuid.UUID,
	requests map[string]int32,
) ([]*slot, bool) {
	// Pre-size the selection slice to the total number of requested units.
	totalNeeded := 0
	for _, units := range requests {
		if units > 0 {
			totalNeeded += int(units)
		}
	}

	selected := make([]*slot, 0, totalNeeded)

	for slotType, units := range requests {
		if units <= 0 {
			continue
		}

		slotsByWorker, ok := slotsByType[slotType]
		if !ok {
			return nil, false
		}

		workerSlots := slotsByWorker[workerId]
		if len(workerSlots) == 0 {
			return nil, false
		}

		needed := int(units)
		found := 0

		for _, s := range workerSlots {
			if !s.active() {
				continue
			}
			selected = append(selected, s)
			found++
			if found >= needed {
				break
			}
		}

		if found < needed {
			return nil, false
		}
	}

	return selected, true
}

// tryAssignSingleton attempts to assign a singleton step to a worker.
func (s *Scheduler) tryAssignSingleton(
	ctx context.Context,
	qi *sqlcv1.V1QueueItem,
	action *action,
	candidateSlots []*slot,
	ringOffset int,
	labels []*sqlcv1.GetDesiredLabelsRow,
	requests map[string]int32,
	rateLimitAck func(),
	rateLimitNack func(),
) (
	res assignSingleResult, err error,
) {
	// NOTE: the caller must hold action.mu (RLock or Lock) while calling this
	// function. We read from action.slots, which is replaced during replenish
	// under action.mu.

	ctx, span := telemetry.NewSpan(ctx, "try-assign-singleton") // nolint: ineffassign
	defer span.End()

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "tenant.id", Value: qi.TenantID.String()})

	ringOffset = ringOffset % len(candidateSlots)

	if (qi.Sticky != sqlcv1.V1StickyStrategyNONE) || len(labels) > 0 {
		candidateSlots = getRankedSlots(qi, labels, candidateSlots)
		ringOffset = 0
	}

	assignedSlot := findAssignableSlots(candidateSlots[ringOffset:], action, requests, rateLimitAck, rateLimitNack)

	if assignedSlot == nil {
		assignedSlot = findAssignableSlots(candidateSlots[:ringOffset], action, requests, rateLimitAck, rateLimitNack)
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

	res.workerId = assignedSlot.workerId()
	if res.workerId == uuid.Nil {
		s.l.Error().Msgf("assigned slot %d has no worker id, skipping assignment", res.ackId)
		res.noSlots = true
		return res, nil
	}

	res.succeeded = true

	return res, nil
}

type assignedQueueItem struct {
	AckId    int
	WorkerId uuid.UUID

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
	stepIdsToLabels map[uuid.UUID][]*sqlcv1.GetDesiredLabelsRow,
	stepIdsToRequests map[uuid.UUID]map[string]int32,
	taskIdsToRateLimits map[int64]map[string]int32,
) <-chan *assignResults {
	ctx, span := telemetry.NewSpan(ctx, "try-assign")

	if len(qis) > 0 {
		uniqueTenantIds := telemetry.CollectUniqueTenantIDs(qis, func(qi *sqlcv1.V1QueueItem) string {
			return qi.TenantID.String()
		})
		telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "tenant.id", Value: uniqueTenantIds})
	}

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

					results, newRingOffset, err := s.tryAssignBatch(ctx, actionId, batchQis, ringOffset, stepIdsToLabels, stepIdsToRequests, taskIdsToRateLimits)

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

		s.exts.PostAssign(s.tenantId, s.getExtensionInput(extensionResults))

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

	workers := s.copyWorkers()

	res := &SnapshotInput{
		Workers: make(map[uuid.UUID]*WorkerCp),
	}

	for workerId, worker := range workers {
		res.Workers[workerId] = &WorkerCp{
			WorkerId: workerId,
			Labels:   worker.Labels,
			Name:     worker.Name,
		}
	}

	// NOTE: these locks are important because we must acquire locks in the same order as the replenish and tryAssignBatch
	// functions. we always acquire actionsMu first and then the specific action's lock.
	actionKeys := make([]string, 0, len(s.actions))

	for actionId := range s.actions {
		actionKeys = append(actionKeys, actionId)
	}

	uniqueSlots := make(map[*slot]bool)

	workerSlotUtilization := make(map[uuid.UUID]*SlotUtilization)

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
