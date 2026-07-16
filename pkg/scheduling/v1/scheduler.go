package v1

import (
	"bytes"
	"context"
	"fmt"
	"slices"
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

	actions       map[string]*action
	pools         map[poolKey]*slotPool
	poolsByWorker map[uuid.UUID]map[string]*slotPool
	actionsMu     rwMutex
	replenishMu   mutex

	workersMu mutex
	workers   map[uuid.UUID]*worker

	assignedCount   int
	assignedCountMu mutex

	// unackedSlots are slots which have been assigned to a worker, but have not been flushed
	// to the database yet. They negatively count towards a worker's available slot count.
	unackedSlots map[int]*assignedSlots
	unackedMu    mutex

	// warmedSlotTypes tracks (worker, slot type) pairs whose slots have appeared in the
	// in-memory pool at least once. An empty pool is ambiguous — a worker which has not
	// been replenished yet looks identical to a fully saturated one — so utilization is
	// only derived from capacity once the pair has warmed up. Accessed exclusively from
	// the snapshot loop goroutine (via getSnapshotInput).
	warmedSlotTypes map[uuid.UUID]map[string]struct{}

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
		pools:           make(map[poolKey]*slotPool),
		poolsByWorker:   make(map[uuid.UUID]map[string]*slotPool),
		unackedSlots:    make(map[int]*assignedSlots),
		warmedSlotTypes: make(map[uuid.UUID]map[string]struct{}),
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
		s.l.Debug().Ctx(ctx).Msg("skipping replenish because another replenish is in progress")
		return nil
	}

	defer s.replenishMu.Unlock()

	// NOTE: the span starts before the actionsMu acquisition so that lock wait time is
	// visible in the trace; the acquire-actions-mu child span isolates that wait.
	ctx, span := telemetry.NewSpan(ctx, "replenish")
	defer span.End()

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "tenant.id", Value: s.tenantId.String()},
		telemetry.AttributeKV{Key: "replenish.must_replenish", Value: mustReplenish},
	)

	// we get a lock on the actions mutexes here because we want to acquire the locks in the same order
	// as the tryAssignBatch function. otherwise, we could deadlock when tryAssignBatch has a lock
	// on the actionsMu and tries to acquire the unackedMu lock.
	// additionally, we have to acquire a lock this early (before the database read) to prevent slots
	// from being assigned while we read slots from the database.
	_, lockSpan := telemetry.NewSpan(ctx, "replenish-acquire-actions-mu")

	if mustReplenish {
		s.actionsMu.Lock()
	} else if ok := s.actionsMu.TryLock(); !ok {
		lockSpan.End()
		s.l.Debug().Ctx(ctx).Msg("skipping replenish because we can't acquire the actions mutex")
		return nil
	}

	lockSpan.End()

	defer s.actionsMu.Unlock()

	s.l.Debug().Ctx(ctx).Msg("replenishing slots")

	workers := s.copyWorkers()
	workerIds := make([]uuid.UUID, 0)

	for workerId := range workers {
		workerIds = append(workerIds, workerId)
	}

	start := time.Now()
	checkpoint := start

	listActionsCtx, listActionsSpan := telemetry.NewSpan(ctx, "replenish-list-actions-for-workers")
	telemetry.WithAttributes(listActionsSpan, telemetry.AttributeKV{Key: "replenish.worker_count", Value: len(workerIds)})

	workersToActiveActions, err := s.repo.ListActionsForWorkers(listActionsCtx, s.tenantId, workerIds)

	if err != nil {
		listActionsSpan.End()
		return err
	}

	telemetry.WithAttributes(listActionsSpan, telemetry.AttributeKV{Key: "replenish.worker_action_rows", Value: len(workersToActiveActions)})
	listActionsSpan.End()

	if sinceStart := time.Since(start); sinceStart > 100*time.Millisecond {
		s.l.Warn().Ctx(ctx).Msgf("listing actions for workers took %s for %d workers", time.Since(checkpoint), len(workerIds))
	} else {
		s.l.Debug().Ctx(ctx).Msgf("listing actions for workers took %s", time.Since(checkpoint))
	}

	checkpoint = time.Now()

	_, computeActionsSpan := telemetry.NewSpan(ctx, "replenish-compute-actions-to-replenish")

	actionsToWorkerIds := make(map[string][]uuid.UUID)

	for _, workerActionTuple := range workersToActiveActions {
		if !workerActionTuple.ActionId.Valid {
			continue
		}

		actionId := workerActionTuple.ActionId.String
		workerId := workerActionTuple.WorkerId

		actionsToWorkerIds[actionId] = append(actionsToWorkerIds[actionId], workerId)
	}

	// FUNCTION 1: determine which actions should be replenished. Logic is the following:
	// - zero or one slots for an action: replenish all slots
	// - some slots for an action: replenish if 50% of slots have been used, or have expired
	// - more workers available for an action than previously: fully replenish
	// - otherwise, do not replenish
	actionsToReplenish := make(map[string]struct{})

	// Isolate the activeCount scans: on large tenants this dominates replenish wall time
	// while actionsMu is held, which blocks tryAssignBatch.
	_, scanActiveSpan := telemetry.NewSpan(ctx, "replenish-scan-active-counts")
	actionsScanned := 0
	activeSlotsTotal := 0
	scanNow := time.Now()

	for actionId, workers := range actionsToWorkerIds {
		// if the action is not in the map, it should be replenished
		if _, ok := s.actions[actionId]; !ok {
			actionsToReplenish[actionId] = struct{}{}
			s.actions[actionId] = new(action)

			continue
		}

		if mustReplenish {
			actionsToReplenish[actionId] = struct{}{}

			continue
		}

		storedAction := s.actions[actionId]

		// determine if we match the conditions above
		var replenish bool
		activeCount := storedAction.activeCountFromPools(s.poolsByWorker, scanNow)
		actionsScanned++
		activeSlotsTotal += activeCount

		switch {
		case activeCount == 0:
			s.l.Debug().Ctx(ctx).Msgf("replenishing all slots for action %s because activeCount is 0", actionId)
			replenish = true
		case activeCount <= (storedAction.lastReplenishedSlotCount / 2):
			s.l.Debug().Ctx(ctx).Msgf("replenishing slots for action %s because 50%% of slots have been used", actionId)
			replenish = true
		case len(workers) > storedAction.lastReplenishedWorkerCount:
			s.l.Debug().Ctx(ctx).Msgf("replenishing slots for action %s because more workers are available", actionId)
			replenish = true
		}

		if replenish {
			actionsToReplenish[actionId] = struct{}{}
		}
	}

	telemetry.WithAttributes(scanActiveSpan,
		telemetry.AttributeKV{Key: "replenish.actions_scanned", Value: actionsScanned},
		telemetry.AttributeKV{Key: "replenish.active_slots", Value: activeSlotsTotal},
		telemetry.AttributeKV{Key: "replenish.unique_actions", Value: len(actionsToWorkerIds)},
	)
	scanActiveSpan.End()

	telemetry.WithAttributes(computeActionsSpan,
		telemetry.AttributeKV{Key: "replenish.actions_to_replenish", Value: len(actionsToReplenish)},
		telemetry.AttributeKV{Key: "replenish.actions_scanned", Value: actionsScanned},
		telemetry.AttributeKV{Key: "replenish.active_slots", Value: activeSlotsTotal},
	)
	computeActionsSpan.End()

	s.l.Debug().Ctx(ctx).Msgf("determining which actions to replenish took %s", time.Since(checkpoint))
	checkpoint = time.Now()

	if len(actionsToReplenish) == 0 {
		telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "replenish.skipped_empty", Value: true})
		return nil
	}

	// FUNCTION 2: load the worker-owned pool configuration and capacity.
	listConfigsCtx, listConfigsSpan := telemetry.NewSpan(ctx, "replenish-list-worker-slot-configs")

	workerSlotConfigs, err := s.repo.ListWorkerSlotConfigs(listConfigsCtx, s.tenantId, workerIds)

	listConfigsSpan.End()

	if err != nil {
		return err
	}

	configuredPools := make(map[poolKey]struct{}, len(workerSlotConfigs))
	slotTypeSet := make(map[string]struct{})
	workerIdSet := make(map[uuid.UUID]struct{})

	for _, config := range workerSlotConfigs {
		configuredPools[poolKey{workerId: config.WorkerID, slotType: config.SlotType}] = struct{}{}
		slotTypeSet[config.SlotType] = struct{}{}
		workerIdSet[config.WorkerID] = struct{}{}
	}

	s.unackedMu.Lock()
	defer s.unackedMu.Unlock()

	slotTypes := make([]string, 0, len(slotTypeSet))
	for slotType := range slotTypeSet {
		slotTypes = append(slotTypes, slotType)
	}

	workerUUIDs := make([]uuid.UUID, 0, len(workerIdSet))
	for workerId := range workerIdSet {
		workerUUIDs = append(workerUUIDs, workerId)
	}

	availableByPool := make(map[poolKey]int, len(configuredPools))
	if len(slotTypes) > 0 && len(workerUUIDs) > 0 {
		listSlotsCtx, listSlotsSpan := telemetry.NewSpan(ctx, "replenish-list-available-slots")

		availableSlots, err := s.repo.ListAvailableSlotsForWorkersAndTypes(listSlotsCtx, s.tenantId, sqlcv1.ListAvailableSlotsForWorkersAndTypesParams{
			Tenantid:  s.tenantId,
			Workerids: workerUUIDs,
			Slottypes: slotTypes,
		})

		listSlotsSpan.End()

		if err != nil {
			return err
		}

		for _, row := range availableSlots {
			availableByPool[poolKey{workerId: row.ID, slotType: row.SlotType}] = int(row.AvailableSlots)
		}
	}

	s.l.Debug().Ctx(ctx).Msgf("loading available slots took %s", time.Since(checkpoint))

	// FUNCTION 3: retain unacked slots in their worker-owned pools.
	unackedByPool := make(map[poolKey][]*slot)
	for _, assignment := range s.unackedSlots {
		for _, assignedSlot := range assignment.slots {
			slotType, err := assignedSlot.getSlotType()
			if err != nil {
				return fmt.Errorf("could not get slot type for unacked slot: %w", err)
			}

			key := poolKey{workerId: assignedSlot.getWorkerId(), slotType: slotType}
			unackedByPool[key] = append(unackedByPool[key], assignedSlot)
			configuredPools[key] = struct{}{}
		}
	}

	// FUNCTION 4: build each (worker, slot type) pool once, then update the
	// action-to-worker index. Slot objects are no longer copied into actions.
	_, buildSlotsSpan := telemetry.NewSpan(ctx, "replenish-build-slots")

	nextPools := make(map[poolKey]*slotPool, len(configuredPools))
	nextPoolsByWorker := make(map[uuid.UUID]map[string]*slotPool)
	totalSlotsBuilt := 0
	maxSlotsPerPool := 0

	for key := range configuredPools {
		w := workers[key.workerId]
		if w == nil {
			continue
		}

		pool := s.pools[key]
		if pool == nil {
			pool = &slotPool{worker: w, slotType: key.slotType}
		} else {
			pool.worker = w
			pool.slotType = key.slotType
		}

		unackedSlots := unackedByPool[key]
		availableCount := availableByPool[key] - len(unackedSlots)
		if availableCount < 0 {
			availableCount = 0
		}

		// Align pool refreshedAt with every slot's expiresAt so staleAt and
		// active() flip together. Otherwise unusedCount can still report
		// capacity after individual slots have expired.
		refreshedAt := time.Now()
		expiresAt := refreshedAt.Add(defaultSlotExpiry)

		slots := make([]*slot, 0, availableCount+len(unackedSlots))
		meta := newSlotMeta(nil, key.slotType)
		for index := 0; index < availableCount; index++ {
			slots = append(slots, newSlotWithExpiry(w, meta, expiresAt))
		}
		for _, unackedSlot := range unackedSlots {
			unackedSlot.setExpiry(expiresAt)
		}
		slots = append(slots, unackedSlots...)
		pool.resetSlotsAt(slots, refreshedAt)

		nextPools[key] = pool
		if nextPoolsByWorker[key.workerId] == nil {
			nextPoolsByWorker[key.workerId] = make(map[string]*slotPool)
		}
		nextPoolsByWorker[key.workerId][key.slotType] = pool

		totalSlotsBuilt += len(slots)
		if len(slots) > maxSlotsPerPool {
			maxSlotsPerPool = len(slots)
		}
	}

	s.pools = nextPools
	s.poolsByWorker = nextPoolsByWorker

	actionsRemoved := 0
	for actionId, storedAction := range s.actions {
		actionWorkerIds := actionsToWorkerIds[actionId]
		if len(actionWorkerIds) > 1 {
			slices.SortFunc(actionWorkerIds, func(left, right uuid.UUID) int {
				return bytes.Compare(left[:], right[:])
			})
			actionWorkerIds = slices.Compact(actionWorkerIds)
		}

		totalSlots := 0
		for _, workerId := range actionWorkerIds {
			for _, pool := range nextPoolsByWorker[workerId] {
				totalSlots += len(pool.slots)
			}
		}

		if totalSlots == 0 {
			delete(s.actions, actionId)
			actionsRemoved++
			continue
		}

		storedAction.workerIds = actionWorkerIds
		storedAction.lastReplenishedSlotCount = totalSlots
		storedAction.lastReplenishedWorkerCount = len(actionWorkerIds)
	}

	telemetry.WithAttributes(buildSlotsSpan,
		telemetry.AttributeKV{Key: "replenish.actions_with_new_slots", Value: len(s.actions)},
		telemetry.AttributeKV{Key: "replenish.slots_built", Value: totalSlotsBuilt},
		telemetry.AttributeKV{Key: "replenish.max_slots_per_pool", Value: maxSlotsPerPool},
		telemetry.AttributeKV{Key: "replenish.pool_count", Value: len(nextPools)},
		telemetry.AttributeKV{Key: "replenish.unacked_slot_entries", Value: len(s.unackedSlots)},
	)
	buildSlotsSpan.End()

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "replenish.actions_with_new_slots", Value: len(s.actions)},
		telemetry.AttributeKV{Key: "replenish.slots_built", Value: totalSlotsBuilt},
		telemetry.AttributeKV{Key: "replenish.max_slots_per_pool", Value: maxSlotsPerPool},
		telemetry.AttributeKV{Key: "replenish.pool_count", Value: len(nextPools)},
		telemetry.AttributeKV{Key: "replenish.actions_removed", Value: actionsRemoved},
	)

	if sinceStart := time.Since(start); sinceStart > 100*time.Millisecond {
		s.l.Warn().Ctx(ctx).Dur("duration", sinceStart).Msg("replenishing slots took longer than 100ms")
	} else {
		s.l.Debug().Ctx(ctx).Dur("duration", sinceStart).Msgf("finished replenishing slots")
	}

	return nil
}

func (s *Scheduler) loopReplenish(ctx context.Context) {
	ticker := randomticker.NewRandomTicker(1000*time.Millisecond, 1500*time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			innerCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			err := s.replenish(innerCtx, true)

			if err != nil {
				s.l.Error().Ctx(ctx).Err(err).Msg("error replenishing slots")
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

			// only advance the counter when a snapshot was actually taken, so
			// the "must" cadence counts real snapshots rather than skipped ticks
			if s.snapshot(ctx, must) {
				count++
			}
		}
	}
}

// snapshot builds a point-in-time view of the tenant's slot utilization and
// reports it to the registered extensions. It returns false when the snapshot
// was skipped because the scheduler was busy (non-must path).
func (s *Scheduler) snapshot(ctx context.Context, mustSnapshot bool) bool {
	ctx, span := telemetry.NewSpan(ctx, "snapshot")
	defer span.End()

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "tenant.id", Value: s.tenantId.String()},
		telemetry.AttributeKV{Key: "snapshot.must", Value: mustSnapshot},
	)

	in, ok := s.getSnapshotInput(ctx, mustSnapshot)

	if !ok {
		telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "snapshot.skipped", Value: true})
		return false
	}

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "snapshot.worker_count", Value: len(in.Workers)})

	s.exts.ReportSnapshot(ctx, s.tenantId, in)

	return true
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
	taskIdsToLabelOverrides map[int64][]*sqlcv1.GetDesiredLabelsRow,
) (
	res []*assignSingleResult, newRingOffset int, err error,
) {
	s.l.Debug().Ctx(ctx).Msgf("trying to assign %d queue items", len(qis))

	newRingOffset = ringOffset

	ctx, span := telemetry.NewSpan(ctx, "try-assign-batch")
	defer span.End()

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "batch.item_count", Value: len(qis)},
		telemetry.AttributeKV{Key: "action.id", Value: actionId},
	)

	if len(qis) > 0 {
		uniqueTenantIds := telemetry.CollectUniqueTenantIDs(qis, func(qi *sqlcv1.V1QueueItem) string {
			return qi.TenantID.String()
		})
		telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "tenant.id", Value: uniqueTenantIds})

		uniqueQueueNames := telemetry.CollectUniqueTenantIDs(qis, func(qi *sqlcv1.V1QueueItem) string {
			return qi.Queue
		})
		telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "queue.name", Value: uniqueQueueNames})
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
	_, rateLimitSpan := telemetry.NewSpan(ctx, "try-assign-batch-rate-limits")
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
	rateLimitSpan.End()

	// Hold the read lock for the entire batch so replenish cannot replace the
	// action-to-worker index or worker pools while assignment is using them.
	// Replenish acquires the write lock before unackedMu; assignment preserves
	// that lock order when it records selected slots.
	_, actionsMuSpan := telemetry.NewSpan(ctx, "try-assign-batch-acquire-actions-mu")
	s.actionsMu.RLock()
	defer s.actionsMu.RUnlock()

	action, ok := s.actions[actionId]
	actionsMuSpan.End()

	if !ok || action == nil {
		telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "action.missing", Value: true})
		s.l.Debug().Ctx(ctx).Msgf("no action %s", actionId)

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

	workerCount := len(action.workerIds)
	if workerCount == 0 {
		telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "action.worker_count", Value: 0})

		s.l.Debug().Ctx(ctx).Msgf("no slots for action %s", actionId)

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

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "action.worker_count", Value: workerCount},
		telemetry.AttributeKV{Key: "action.slot_count", Value: action.lastReplenishedSlotCount},
	)

	for i := range res {
		if res[i].rateLimitResult != nil {
			continue
		}

		denom := workerCount

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

		labelOverrides, ok := taskIdsToLabelOverrides[qi.TaskID]

		if ok {
			labels = labelOverrides
		}

		// Backwards-compatible default: if no slot requests are provided for a step,
		// assume it needs 1 default slot.
		requests := map[string]int32{v1.SlotTypeDefault: 1}
		if stepIdsToRequests != nil {
			if r, ok := stepIdsToRequests[qi.StepID]; ok && len(r) > 0 {
				requests = r
			}
		}

		singleRes, err := s.tryAssignSingletonFromPools(
			ctx,
			qi,
			action.workerIds,
			childRingOffset,
			labels,
			requests,
			rlAcks[i],
			rlNacks[i],
		)

		if err != nil {
			s.l.Error().Ctx(ctx).Err(err).Msg("error assigning queue item")
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

func (s *Scheduler) findAssignableWorkerPools(
	workerIds []uuid.UUID,
	requests map[string]int32,
	rateLimitAck func(),
	rateLimitNack func(),
) *assignedSlots {
	for _, workerId := range workerIds {
		selected, ok := selectSlotsFromPools(s.poolsByWorker[workerId], requests)
		if !ok {
			continue
		}

		usedSlots, ok := useSelectedSlots(selected)
		if !ok {
			continue
		}

		return &assignedSlots{
			slots:         usedSlots,
			rateLimitAck:  rateLimitAck,
			rateLimitNack: rateLimitNack,
		}
	}

	return nil
}

func selectSlotsFromPools(poolsByType map[string]*slotPool, requests map[string]int32) ([]*slot, bool) {
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

		pool := poolsByType[slotType]
		if pool == nil {
			return nil, false
		}

		// Do not gate on unusedCount: expiry does not decrement the counter, so it
		// can drift above (or, after concurrent use/nack, below) the number of
		// active slots. Selection must follow active() only.
		found := 0
		for _, sl := range pool.slots {
			if !sl.active() {
				continue
			}
			selected = append(selected, sl)
			found++
			if found == int(units) {
				break
			}
		}
		if found < int(units) {
			return nil, false
		}
	}

	return selected, true
}

func (s *Scheduler) rankWorkerIds(
	qi *sqlcv1.V1QueueItem,
	labels []*sqlcv1.GetDesiredLabelsRow,
	workerIds []uuid.UUID,
) []uuid.UUID {
	type rankedWorker struct {
		id   uuid.UUID
		rank int
	}

	ranked := make([]rankedWorker, 0, len(workerIds))
	for _, workerId := range workerIds {
		w := s.poolsByWorker[workerId]
		var worker *worker
		for _, pool := range w {
			worker = pool.worker
			break
		}
		if worker == nil {
			continue
		}

		rank := 0
		switch qi.Sticky {
		case sqlcv1.V1StickyStrategyHARD:
			if qi.DesiredWorkerID != nil && workerId != *qi.DesiredWorkerID {
				continue
			}
		case sqlcv1.V1StickyStrategySOFT:
			if qi.DesiredWorkerID != nil && workerId == *qi.DesiredWorkerID {
				rank = 1
			}
		default:
			if len(labels) > 0 {
				rank = worker.computeWeight(labels)
				if rank < 0 {
					continue
				}
			}
		}

		ranked = append(ranked, rankedWorker{id: workerId, rank: rank})
	}

	slices.SortStableFunc(ranked, func(left, right rankedWorker) int {
		return right.rank - left.rank
	})

	result := make([]uuid.UUID, len(ranked))
	for index := range ranked {
		result[index] = ranked[index].id
	}
	return result
}

func (s *Scheduler) tryAssignSingletonFromPools(
	ctx context.Context,
	qi *sqlcv1.V1QueueItem,
	workerIds []uuid.UUID,
	ringOffset int,
	labels []*sqlcv1.GetDesiredLabelsRow,
	requests map[string]int32,
	rateLimitAck func(),
	rateLimitNack func(),
) (res assignSingleResult, err error) {
	ctx, span := telemetry.NewSpan(ctx, "try-assign-singleton") // nolint: ineffassign
	defer span.End()

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "tenant.id", Value: qi.TenantID.String()},
		telemetry.AttributeKV{Key: "queue.name", Value: qi.Queue},
	)

	candidates := workerIds
	if qi.Sticky != sqlcv1.V1StickyStrategyNONE || len(labels) > 0 {
		candidates = s.rankWorkerIds(qi, labels, workerIds)
		ringOffset = 0
	}
	if len(candidates) == 0 {
		res.noSlots = true
		return res, nil
	}

	ringOffset %= len(candidates)
	assigned := s.findAssignableWorkerPools(candidates[ringOffset:], requests, rateLimitAck, rateLimitNack)
	if assigned == nil {
		assigned = s.findAssignableWorkerPools(candidates[:ringOffset], requests, rateLimitAck, rateLimitNack)
	}
	if assigned == nil {
		res.noSlots = true
		return res, nil
	}

	s.assignedCountMu.Lock()
	s.assignedCount++
	res.ackId = s.assignedCount
	s.assignedCountMu.Unlock()

	s.unackedMu.Lock()
	s.unackedSlots[res.ackId] = assigned
	s.unackedMu.Unlock()

	res.workerId = assigned.workerId()
	if res.workerId == uuid.Nil {
		s.l.Error().Ctx(ctx).Msgf("assigned slot %d has no worker id, skipping assignment", res.ackId)
		res.noSlots = true
		return res, nil
	}

	res.succeeded = true
	return res, nil
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
	taskIdsToLabelOverrides map[int64][]*sqlcv1.GetDesiredLabelsRow,
) <-chan *assignResults {
	ctx, span := telemetry.NewSpan(ctx, "try-assign")

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "batch.item_count", Value: len(qis)})

	if len(qis) > 0 {
		uniqueTenantIds := telemetry.CollectUniqueTenantIDs(qis, func(qi *sqlcv1.V1QueueItem) string {
			return qi.TenantID.String()
		})
		telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "tenant.id", Value: uniqueTenantIds})

		uniqueQueueNames := telemetry.CollectUniqueTenantIDs(qis, func(qi *sqlcv1.V1QueueItem) string {
			return qi.Queue
		})
		telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "queue.name", Value: uniqueQueueNames})
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

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "batch.action_count", Value: len(actionIdToQueueItems)})

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

					results, newRingOffset, err := s.tryAssignBatch(ctx, actionId, batchQis, ringOffset, stepIdsToLabels, stepIdsToRequests, taskIdsToRateLimits, taskIdsToLabelOverrides)

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
									s.l.Error().Ctx(ctx).Msgf("scheduling failed for queue item %d: expected assignment to fail with either no slots or rate limit exceeded, but failed with neither", singleRes.qi.ID)
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
						s.l.Warn().Ctx(ctx).Dur("duration", sinceStart).Msgf("processing batch of %d queue items took longer than 100ms", len(batchQis))
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
					s.l.Error().Ctx(ctx).Err(err).Msg("error assigning queue items")
				}
			}(actionId, qis)
		}

		wg.Wait()
		span.End()
		close(resultsCh)

		s.exts.PostAssign(s.tenantId, s.getExtensionInput(extensionResults))

		if sinceStart := time.Since(startTotal); sinceStart > 100*time.Millisecond {
			s.l.Warn().Ctx(ctx).Dur("duration", sinceStart).Msgf("assigning queue items took longer than 100ms")
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

func (s *Scheduler) getSnapshotInput(ctx context.Context, mustSnapshot bool) (*SnapshotInput, bool) {
	ctx, span := telemetry.NewSpan(ctx, "get-snapshot-input")
	defer span.End()

	// isolate the lock-acquire wait in its own child span; on the non-must path
	// a contended lock makes us skip the snapshot entirely.
	_, lockSpan := telemetry.NewSpan(ctx, "get-snapshot-input-acquire-actions-mu")
	if mustSnapshot {
		s.actionsMu.RLock()
	} else {
		if ok := s.actionsMu.TryRLock(); !ok {
			lockSpan.End()
			telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "snapshot.lock_contended", Value: true})
			return nil, false
		}
	}
	lockSpan.End()

	defer s.actionsMu.RUnlock()

	workers := s.copyWorkers()

	res := &SnapshotInput{
		Workers:                     make(map[uuid.UUID]*WorkerCp, len(workers)),
		WorkerSlotUtilization:       make(map[uuid.UUID]*SlotUtilization, len(workers)),
		WorkerSlotUtilizationByType: make(map[uuid.UUID]map[string]*SlotUtilization, len(workers)),
	}

	for workerId, worker := range workers {
		totalSlots := 0

		for _, units := range worker.TotalSlotsByType {
			totalSlots += units
		}

		res.Workers[workerId] = &WorkerCp{
			WorkerId: workerId,
			Labels:   worker.Labels,
			Name:     worker.Name,
			MaxRuns:  totalSlots,
		}
	}

	utilizationByType := make(map[uuid.UUID]map[string]*SlotUtilization)

	for workerId := range workers {
		utilizationByType[workerId] = make(map[string]*SlotUtilization)
	}

	_, walkSpan := telemetry.NewSpan(ctx, "get-snapshot-input-walk-slots")
	uniqueSlotCount := 0
	for key, pool := range s.pools {
		if _, active := workers[key.workerId]; !active {
			continue
		}
		byType := utilizationByType[key.workerId]
		if byType == nil {
			byType = make(map[string]*SlotUtilization)
			utilizationByType[key.workerId] = byType
		}
		utilization := byType[key.slotType]
		if utilization == nil {
			utilization = &SlotUtilization{}
			byType[key.slotType] = utilization
		}

		for _, slot := range pool.slots {
			uniqueSlotCount++
			if slot.isUsed() {
				utilization.UtilizedSlots++
			} else {
				utilization.NonUtilizedSlots++
			}
		}
	}
	telemetry.WithAttributes(walkSpan,
		telemetry.AttributeKV{Key: "snapshot.action_count", Value: len(s.actions)},
		telemetry.AttributeKV{Key: "snapshot.unique_slots", Value: uniqueSlotCount},
	)
	walkSpan.End()

	// prune warm state for workers which are no longer registered
	for workerId := range s.warmedSlotTypes {
		if _, ok := workers[workerId]; !ok {
			delete(s.warmedSlotTypes, workerId)
		}
	}

	// The in-memory pool only holds slots which have not been assigned (plus assigned slots
	// which are not yet flushed to the database), so the used counts walked above miss any
	// slot consumed by a running task. Derive the true used count per slot type from the
	// worker's slot capacity instead: everything that is not free is in use.
	for workerId, byType := range utilizationByType {
		var capacities map[string]int

		if worker, ok := workers[workerId]; ok {
			capacities = worker.TotalSlotsByType
		}

		warmed := s.warmedSlotTypes[workerId]

		for slotType, utilization := range byType {
			if utilization.UtilizedSlots+utilization.NonUtilizedSlots > 0 {
				if warmed == nil {
					warmed = make(map[string]struct{})
					s.warmedSlotTypes[workerId] = warmed
				}

				warmed[slotType] = struct{}{}
			}
		}

		// slot types with capacity but no walked slots still get reported: once the
		// type has warmed up, an empty in-memory pool means all of its slots are in use
		for slotType := range capacities {
			if _, ok := byType[slotType]; !ok {
				byType[slotType] = &SlotUtilization{}
			}
		}

		aggregate := &SlotUtilization{}

		for slotType, utilization := range byType {
			_, isWarmed := warmed[slotType]

			// Only derive from capacity once the slot type has had slots in the pool:
			// a never-replenished worker would otherwise report full utilization
			// between registration and its first replenish. Un-warmed types report
			// zero slots, which extensions treat as a transient state.
			if capacity := capacities[slotType]; capacity > 0 && isWarmed {
				used := capacity - utilization.NonUtilizedSlots
				if used < 0 {
					used = 0
				}

				utilization.UtilizedSlots = used
			}
			// no capacity known for this slot type; fall back to the walked counts

			aggregate.UtilizedSlots += utilization.UtilizedSlots
			aggregate.NonUtilizedSlots += utilization.NonUtilizedSlots
		}

		res.WorkerSlotUtilizationByType[workerId] = byType
		res.WorkerSlotUtilization[workerId] = aggregate
	}

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "snapshot.worker_count", Value: len(workers)},
		telemetry.AttributeKV{Key: "snapshot.action_count", Value: len(s.actions)},
	)

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
