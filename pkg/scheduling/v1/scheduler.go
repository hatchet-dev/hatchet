package v1

import (
	"bytes"
	"context"
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

// parkedAssignRetryTimeout bounds how long an assignment that raced an
// in-flight replenish waits for the cycle to end before reporting its miss. A
// healthy replenish applies within a few milliseconds; a degraded one (slow
// database reads) must not hold assignment results hostage.
const parkedAssignRetryTimeout = 100 * time.Millisecond

// Scheduler is responsible for scheduling steps to workers as efficiently as possible.
// This is tenant-scoped, so each tenant will have its own scheduler.
//
// All mutable scheduling state is owned by a single run-loop goroutine: every
// read and write happens inside an op sent to the loop via do(). There are no
// locks and no lock-ordering rules. Database reads run outside the loop, so
// assignment is never blocked on I/O; replenish reconciles its reads against
// assignments that completed in the meantime (see ackedDuringReplenish).
type Scheduler struct {
	repo     v1.AssignmentRepository
	tenantId uuid.UUID

	l *zerolog.Logger

	rl   *rateLimiter
	exts *Extensions

	// ops is the run loop's mailbox; runDone is closed when the run loop exits.
	ops     chan func()
	runDone chan struct{}

	// ---- state below is owned by the run loop ----

	actions       map[string]*action
	pools         map[poolKey]*slotPool
	poolsByWorker map[uuid.UUID]map[string]*slotPool
	workers       map[uuid.UUID]*worker

	// unackedSlots are slots which have been assigned to a worker, but have not been flushed
	// to the database yet. They negatively count towards a worker's available slot count.
	unackedSlots  map[int]*assignedSlots
	assignedCount int

	// replenishing guards against overlapping replenish cycles. While a cycle is
	// in flight, ackedDuringReplenish counts acked slots per pool: those flushes
	// may not be visible to the cycle's availability read, so the rebuild
	// subtracts them to avoid double-counting capacity.
	replenishing         bool
	ackedDuringReplenish map[poolKey]int

	// afterReplenish holds assignment retries parked while a replenish cycle is
	// in flight; they run on the loop as soon as the cycle ends. This preserves
	// the v1 scheduler's behavior where an assignment racing a replenish waited
	// on the actions write lock and woke to fresh capacity, instead of missing
	// and paying a full queue poll interval.
	afterReplenish []func()

	// warmedSlotTypes tracks (worker, slot type) pairs whose slots have appeared in the
	// in-memory pool at least once. An empty pool is ambiguous — a worker which has not
	// been replenished yet looks identical to a fully saturated one — so utilization is
	// only derived from capacity once the pair has warmed up.
	warmedSlotTypes map[uuid.UUID]map[string]struct{}
}

func newScheduler(cf *sharedConfig, tenantId uuid.UUID, rl *rateLimiter, exts *Extensions) *Scheduler {
	l := cf.l.With().Str("tenant_id", tenantId.String()).Logger()

	return &Scheduler{
		repo:            cf.repo.Assignment(),
		tenantId:        tenantId,
		l:               &l,
		rl:              rl,
		exts:            exts,
		ops:             make(chan func(), 128),
		runDone:         make(chan struct{}),
		actions:         make(map[string]*action),
		pools:           make(map[poolKey]*slotPool),
		poolsByWorker:   make(map[uuid.UUID]map[string]*slotPool),
		workers:         make(map[uuid.UUID]*worker),
		unackedSlots:    make(map[int]*assignedSlots),
		warmedSlotTypes: make(map[uuid.UUID]map[string]struct{}),
	}
}

func (s *Scheduler) start(ctx context.Context) {
	go s.run(ctx)
	go s.loopReplenish(ctx)
	go s.loopSnapshot(ctx)
}

// run is the scheduler's event loop. It is the only goroutine that touches the
// scheduling state, which is what lets the rest of this file be plain
// single-threaded code.
func (s *Scheduler) run(ctx context.Context) {
	defer close(s.runDone)

	for {
		select {
		case <-ctx.Done():
			return
		case op := <-s.ops:
			op()
		}
	}
}

// do runs fn on the run loop and waits for it to complete. It returns false —
// and fn is guaranteed not to have run or to ever run — if the caller's context
// is done before fn could be enqueued, or the run loop has exited. Once fn is
// enqueued, do waits for it to complete regardless of the caller's context so
// results written by fn are never read while fn is still running.
func (s *Scheduler) do(ctx context.Context, fn func()) bool {
	// fast-path guard: a select with a ready send and a done context picks
	// randomly, so check cancellation explicitly first
	if ctx.Err() != nil {
		return false
	}

	done := make(chan struct{})

	select {
	case s.ops <- func() {
		defer close(done)
		fn()
	}:
	case <-ctx.Done():
		return false
	case <-s.runDone:
		return false
	}

	return s.wait(done)
}

// mustDo runs fn on the run loop, waiting for it to complete. Unlike do it is
// not cancellable: it gives up only when the run loop has exited (scheduler
// shutdown). It exists for ops that must apply once their trigger has already
// happened — dropping an ack after its flush committed to the database, or a
// worker update after the lease changed, would leave the loop's state
// permanently out of sync with the database.
func (s *Scheduler) mustDo(fn func()) bool {
	done := make(chan struct{})

	select {
	case s.ops <- func() {
		defer close(done)
		fn()
	}:
	case <-s.runDone:
		return false
	}

	return s.wait(done)
}

func (s *Scheduler) wait(done <-chan struct{}) bool {
	select {
	case <-done:
		return true
	case <-s.runDone:
		// The run loop finishes any op it has dequeued before exiting, so once
		// runDone is closed the op has either fully completed or will never run.
		select {
		case <-done:
			return true
		default:
			return false
		}
	}
}

// ack marks assignments as flushed to the database. The slots stay used until a
// later replenish observes the flush in its availability read.
func (s *Scheduler) ack(ids []int) {
	var callbacks []func()

	s.mustDo(func() {
		callbacks = s.handleAck(ids)
	})

	// rate-limit callbacks run outside the run loop so it never blocks on
	// another subsystem's locks
	for _, cb := range callbacks {
		cb()
	}
}

// nack returns assignments which failed to flush to their pools.
func (s *Scheduler) nack(ids []int) {
	var callbacks []func()

	s.mustDo(func() {
		callbacks = s.handleNack(ids)
	})

	for _, cb := range callbacks {
		cb()
	}
}

func (s *Scheduler) handleAck(ids []int) []func() {
	var callbacks []func()

	for _, id := range ids {
		assigned, ok := s.unackedSlots[id]

		if !ok {
			continue
		}

		delete(s.unackedSlots, id)

		if s.replenishing {
			for _, sl := range assigned.slots {
				s.ackedDuringReplenish[poolKey{workerId: sl.getWorkerId(), slotType: sl.slotType}]++
			}
		}

		if assigned.rateLimitAck != nil {
			callbacks = append(callbacks, assigned.rateLimitAck)
		}
	}

	return callbacks
}

func (s *Scheduler) handleNack(ids []int) []func() {
	var callbacks []func()

	for _, id := range ids {
		assigned, ok := s.unackedSlots[id]

		if !ok {
			continue
		}

		delete(s.unackedSlots, id)

		for _, sl := range assigned.slots {
			if sl.pool != nil {
				sl.pool.release(sl)
			}
		}

		if assigned.rateLimitNack != nil {
			callbacks = append(callbacks, assigned.rateLimitNack)
		}
	}

	return callbacks
}

func (s *Scheduler) setWorkers(workers []*v1.ListActiveWorkersResult) {
	s.mustDo(func() {
		newWorkers := make(map[uuid.UUID]*worker, len(workers))

		for i := range workers {
			newWorkers[workers[i].ID] = &worker{
				ListActiveWorkersResult: workers[i],
			}
		}

		s.workers = newWorkers
	})
}

func (s *Scheduler) addWorker(newWorker *v1.ListActiveWorkersResult) {
	s.mustDo(func() {
		s.workers[newWorker.ID] = &worker{
			ListActiveWorkersResult: newWorker,
		}
	})
}

// endReplenishCycle runs on the run loop when an in-flight replenish cycle
// finishes (applied, skipped as empty, or failed).
func (s *Scheduler) endReplenishCycle() {
	s.replenishing = false
	s.ackedDuringReplenish = nil

	// retry assignments that missed capacity while the cycle was in flight
	pending := s.afterReplenish
	s.afterReplenish = nil

	for _, retry := range pending {
		retry()
	}
}

// replenish loads new slots from the database and swaps them into the worker
// pools. All database reads run outside the run loop, so assignment continues
// while they are in flight.
func (s *Scheduler) replenish(ctx context.Context, mustReplenish bool) error {
	ctx, span := telemetry.NewSpan(ctx, "replenish")
	defer span.End()

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "tenant.id", Value: s.tenantId.String()},
		telemetry.AttributeKV{Key: "replenish.must_replenish", Value: mustReplenish},
	)

	// Phase 1 (run loop): skip if another cycle is in flight, snapshot the worker
	// ids, and start counting acks so the availability read below can be
	// reconciled against assignments that flush while it runs.
	var workerIds []uuid.UUID
	skipped := false

	if ok := s.do(ctx, func() {
		if s.replenishing {
			skipped = true
			return
		}

		s.replenishing = true
		s.ackedDuringReplenish = make(map[poolKey]int)

		workerIds = make([]uuid.UUID, 0, len(s.workers))
		for workerId := range s.workers {
			workerIds = append(workerIds, workerId)
		}
	}); !ok {
		return ctx.Err()
	}

	if skipped {
		s.l.Debug().Ctx(ctx).Msg("skipping replenish because another replenish is in progress")
		telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "replenish.skipped_in_progress", Value: true})
		return nil
	}

	// every exit path below must clear the replenishing flag; the apply op does
	// it on success
	applied := false

	defer func() {
		if applied {
			return
		}

		s.mustDo(func() {
			s.endReplenishCycle()
		})
	}()

	s.l.Debug().Ctx(ctx).Msg("replenishing slots")

	start := time.Now()
	checkpoint := start

	// Phase 2 (db): load the action registrations for the active workers.
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

	actionsToWorkerIds := make(map[string][]uuid.UUID)

	for _, workerActionTuple := range workersToActiveActions {
		if !workerActionTuple.ActionId.Valid {
			continue
		}

		actionId := workerActionTuple.ActionId.String
		workerId := workerActionTuple.WorkerId

		actionsToWorkerIds[actionId] = append(actionsToWorkerIds[actionId], workerId)
	}

	// Phase 3 (run loop): determine which actions should be replenished. Logic is
	// the following:
	// - action not seen before: replenish
	// - zero active slots for an action: replenish
	// - 50% or more of the last replenished slots have been used: replenish
	// - more workers available for an action than previously: replenish
	// - otherwise, do not replenish
	_, computeActionsSpan := telemetry.NewSpan(ctx, "replenish-compute-actions-to-replenish")

	actionsToReplenish := make(map[string]struct{})
	actionsScanned := 0
	activeSlotsTotal := 0

	if ok := s.do(ctx, func() {
		scanNow := time.Now()

		for actionId, workers := range actionsToWorkerIds {
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

			var replenish bool
			activeCount := storedAction.activeCount(s.poolsByWorker, scanNow)
			actionsScanned++
			activeSlotsTotal += activeCount

			switch {
			case activeCount == 0:
				replenish = true
			case activeCount <= (storedAction.lastReplenishedSlotCount / 2):
				replenish = true
			case len(workers) > storedAction.lastReplenishedWorkerCount:
				replenish = true
			}

			if replenish {
				actionsToReplenish[actionId] = struct{}{}
			}
		}
	}); !ok {
		computeActionsSpan.End()
		return ctx.Err()
	}

	telemetry.WithAttributes(computeActionsSpan,
		telemetry.AttributeKV{Key: "replenish.actions_to_replenish", Value: len(actionsToReplenish)},
		telemetry.AttributeKV{Key: "replenish.actions_scanned", Value: actionsScanned},
		telemetry.AttributeKV{Key: "replenish.active_slots", Value: activeSlotsTotal},
		telemetry.AttributeKV{Key: "replenish.unique_actions", Value: len(actionsToWorkerIds)},
	)
	computeActionsSpan.End()

	s.l.Debug().Ctx(ctx).Msgf("determining which actions to replenish took %s", time.Since(checkpoint))
	checkpoint = time.Now()

	if len(actionsToReplenish) == 0 {
		telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "replenish.skipped_empty", Value: true})
		return nil
	}

	// Phase 4 (db): load the worker-owned pool configuration and capacity.
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

	slotTypes := make([]string, 0, len(slotTypeSet))
	for slotType := range slotTypeSet {
		slotTypes = append(slotTypes, slotType)
	}

	workerUUIDs := make([]uuid.UUID, 0, len(workerIdSet))
	for workerId := range workerIdSet {
		workerUUIDs = append(workerUUIDs, workerId)
	}

	// Acks that landed before this point are visible to the availability read
	// below, so only acks from here on need to be reconciled against it. This
	// keeps the conservative double-subtract window to the single read that
	// actually matters.
	if ok := s.do(ctx, func() {
		s.ackedDuringReplenish = make(map[poolKey]int)
	}); !ok {
		return ctx.Err()
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

	// Phase 5 (run loop): build each (worker, slot type) pool once, then update
	// the action-to-worker index.
	_, buildSlotsSpan := telemetry.NewSpan(ctx, "replenish-build-slots")

	totalSlotsBuilt := 0
	maxSlotsPerPool := 0
	actionsRemoved := 0
	actionCount := 0
	unackedEntries := 0

	if ok := s.do(ctx, func() {
		// retain unacked slots in their worker-owned pools
		unackedByPool := make(map[poolKey][]*slot)
		for _, assignment := range s.unackedSlots {
			for _, assignedSlot := range assignment.slots {
				key := poolKey{workerId: assignedSlot.getWorkerId(), slotType: assignedSlot.slotType}
				unackedByPool[key] = append(unackedByPool[key], assignedSlot)
				configuredPools[key] = struct{}{}
			}
		}
		unackedEntries = len(s.unackedSlots)

		refreshedAt := time.Now()
		expiresAt := refreshedAt.Add(defaultSlotExpiry)

		nextPools := make(map[poolKey]*slotPool, len(configuredPools))
		nextPoolsByWorker := make(map[uuid.UUID]map[string]*slotPool)

		for key := range configuredPools {
			w := s.workers[key.workerId]
			if w == nil {
				continue
			}

			pool := s.pools[key]
			if pool == nil {
				pool = &slotPool{}
			}
			pool.worker = w
			pool.slotType = key.slotType

			unackedSlots := unackedByPool[key]

			// Assignments which acked while the availability read was in flight
			// are no longer in unackedSlots, but the read may still have counted
			// their slots as available; subtract them so capacity is not
			// double-counted. This can briefly under-count (an ack the read did
			// observe is subtracted again), which self-corrects on the next cycle.
			availableCount := availableByPool[key] - len(unackedSlots) - s.ackedDuringReplenish[key]
			if availableCount < 0 {
				availableCount = 0
			}

			slots := make([]*slot, 0, availableCount+len(unackedSlots))
			for i := 0; i < availableCount; i++ {
				slots = append(slots, &slot{worker: w, slotType: key.slotType})
			}
			slots = append(slots, unackedSlots...)
			pool.reset(slots, expiresAt)

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

		actionCount = len(s.actions)

		s.endReplenishCycle()
		applied = true
	}); !ok {
		buildSlotsSpan.End()
		return ctx.Err()
	}

	telemetry.WithAttributes(buildSlotsSpan,
		telemetry.AttributeKV{Key: "replenish.actions_with_new_slots", Value: actionCount},
		telemetry.AttributeKV{Key: "replenish.slots_built", Value: totalSlotsBuilt},
		telemetry.AttributeKV{Key: "replenish.max_slots_per_pool", Value: maxSlotsPerPool},
		telemetry.AttributeKV{Key: "replenish.unacked_slot_entries", Value: unackedEntries},
	)
	buildSlotsSpan.End()

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "replenish.actions_with_new_slots", Value: actionCount},
		telemetry.AttributeKV{Key: "replenish.slots_built", Value: totalSlotsBuilt},
		telemetry.AttributeKV{Key: "replenish.max_slots_per_pool", Value: maxSlotsPerPool},
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

	// toBatch indicates this queue item should be moved into the v1 batched queue table,
	// rather than scheduled to a worker slot.
	toBatch bool

	// rateLimitAck/Nack are used for non-slot outcomes (like moving to the batched queue table).
	// For slot-assigned outcomes, these are wired into the assignment and invoked on ack/nack.
	rateLimitAck  func()
	rateLimitNack func()
}

type batchedQueueItemResult struct {
	qi            *sqlcv1.V1QueueItem
	rateLimitAck  func()
	rateLimitNack func()
}

func (s *Scheduler) tryAssignBatch(
	ctx context.Context,
	actionId string,
	qis []*sqlcv1.V1QueueItem,
	stepIdsToLabels map[uuid.UUID][]*sqlcv1.GetDesiredLabelsRow,
	stepIdsToRequests map[uuid.UUID]map[string]int32,
	taskIdsToRateLimits map[int64]map[string]int32,
	stepIdsToBatchConfig map[string]bool,
	taskIdsToLabelOverrides map[int64][]*sqlcv1.GetDesiredLabelsRow,
) (
	res []*assignSingleResult, err error,
) {
	s.l.Debug().Ctx(ctx).Msgf("trying to assign %d queue items", len(qis))

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

	// rate limits are checked outside the run loop: the rate limiter has its own
	// synchronization and may not block scheduling for other actions
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

		// store for non-slot outcomes (e.g. moving to batched queue table)
		r.rateLimitAck = rateLimitAck
		r.rateLimitNack = rateLimitNack
	}

	// After rate limits are evaluated, mark batch candidates to be moved to the batched queue table.
	// This replaces the old DB trigger-based redirect. Batch-eligible items do not need a worker
	// slot here — they are moved to v1_batched_queue_item and the batch scheduler handles worker
	// assignment — so they keep their rate-limit reservation and are skipped below.
	for i := range res {
		qi := res[i].qi
		if qi == nil {
			continue
		}

		if stepIdsToBatchConfig == nil || !stepIdsToBatchConfig[qi.StepID.String()] {
			continue
		}

		res[i].toBatch = true
	}
	rateLimitSpan.End()

	assignDone := make(chan struct{})

	// finished is only touched on the run loop; once true, res belongs to the
	// caller again and no parked retry or timeout may touch it.
	finished := false
	finish := func() {
		if finished {
			return
		}
		finished = true
		close(assignDone)
	}

	var attempt func(isRetry bool)
	attempt = func(isRetry bool) {
		s.handleAssignBatch(actionId, qis, res, rlAcks, rlNacks, stepIdsToLabels, stepIdsToRequests, taskIdsToLabelOverrides)

		// If a replenish cycle is in flight, capacity may be milliseconds away:
		// park the missed items and retry once when the cycle ends, instead of
		// reporting noSlots and paying a full queue poll interval. This mirrors
		// the v1 scheduler, where an assignment racing a replenish blocked on
		// the actions write lock and woke to fresh capacity — but unlike v1 the
		// wait is bounded by parkedAssignRetryTimeout, so a slow replenish
		// (e.g. degraded database reads) cannot stall assignment results.
		if !isRetry && s.replenishing && batchHasMisses(res) {
			s.afterReplenish = append(s.afterReplenish, func() {
				if finished {
					return
				}

				// clear the miss markers from the first attempt before retrying
				for i := range res {
					if res[i].rateLimitResult == nil && !res[i].toBatch && !res[i].succeeded {
						res[i].noSlots = false
					}
				}

				attempt(true)
			})

			time.AfterFunc(parkedAssignRetryTimeout, func() {
				s.mustDo(finish)
			})

			return
		}

		finish()
	}

	enqueued := ctx.Err() == nil
	if enqueued {
		select {
		case s.ops <- func() { attempt(false) }:
		case <-ctx.Done():
			enqueued = false
		case <-s.runDone:
			enqueued = false
		}
	}

	if !enqueued || !s.wait(assignDone) {
		// the scheduler is shutting down; treat the batch as unassignable
		for i := range res {
			if res[i].rateLimitResult == nil && !res[i].toBatch && !res[i].succeeded {
				res[i].noSlots = true
			}
		}
	}

	// release rate-limit reservations for items that did not get assigned
	for i := range res {
		if res[i].rateLimitResult == nil && !res[i].succeeded && !res[i].toBatch {
			rlNacks[i]()
		}
	}

	return res, nil
}

// batchHasMisses runs on the run loop.
func batchHasMisses(res []*assignSingleResult) bool {
	for i := range res {
		if res[i].rateLimitResult == nil && !res[i].toBatch && !res[i].succeeded {
			return true
		}
	}
	return false
}

// handleAssignBatch runs on the run loop.
func (s *Scheduler) handleAssignBatch(
	actionId string,
	qis []*sqlcv1.V1QueueItem,
	res []*assignSingleResult,
	rlAcks []func(),
	rlNacks []func(),
	stepIdsToLabels map[uuid.UUID][]*sqlcv1.GetDesiredLabelsRow,
	stepIdsToRequests map[uuid.UUID]map[string]int32,
	taskIdsToLabelOverrides map[int64][]*sqlcv1.GetDesiredLabelsRow,
) {
	action, ok := s.actions[actionId]

	if !ok || action == nil || len(action.workerIds) == 0 {
		s.l.Debug().Msgf("no slots for action %s", actionId)

		// Treat missing action as "no slots" for non-rate-limited, non-batch queue items.
		// Batch-eligible items (toBatch=true) do NOT need a worker slot here — they are
		// moved to v1_batched_queue_item and the batch scheduler handles worker assignment.
		// Marking them noSlots here would strand them in v1_queue_item indefinitely if the
		// action isn't yet present in s.actions (e.g. replenish hasn't fired yet).
		for i := range res {
			if res[i].rateLimitResult == nil && !res[i].toBatch {
				res[i].noSlots = true
			}
		}

		return
	}

	now := time.Now()

	for i := range res {
		r := res[i]

		// Batch candidates are moved to v1_batched_queue_item in the queuer flush path.
		// They should not consume a worker slot here.
		if r.toBatch {
			continue
		}

		if r.rateLimitResult != nil {
			continue
		}

		// already assigned by a previous attempt (parked-retry path)
		if r.succeeded {
			continue
		}

		qi := qis[i]

		labels := []*sqlcv1.GetDesiredLabelsRow(nil)

		if stepIdsToLabels != nil {
			labels = stepIdsToLabels[qi.StepID]
		}

		if labelOverrides, ok := taskIdsToLabelOverrides[qi.TaskID]; ok {
			labels = labelOverrides
		}

		// Backwards-compatible default: if no slot requests are provided for a step,
		// assume it needs 1 default slot.
		requests := map[string]int32{v1.SlotTypeDefault: 1}
		if stepIdsToRequests != nil {
			if req, ok := stepIdsToRequests[qi.StepID]; ok && len(req) > 0 {
				requests = req
			}
		}

		s.assignSingleton(action, qi, r, labels, requests, rlAcks[i], rlNacks[i], now)
	}
}

// assignSingleton runs on the run loop.
func (s *Scheduler) assignSingleton(
	a *action,
	qi *sqlcv1.V1QueueItem,
	r *assignSingleResult,
	labels []*sqlcv1.GetDesiredLabelsRow,
	requests map[string]int32,
	rateLimitAck func(),
	rateLimitNack func(),
	now time.Time,
) {
	candidates := a.workerIds
	offset := a.ringOffset
	a.ringOffset++
	topRankCount := len(candidates)

	if qi.Sticky != sqlcv1.V1StickyStrategyNONE || len(labels) > 0 {
		candidates, topRankCount = s.rankWorkerIds(qi, labels, a.workerIds)
	}

	if len(candidates) == 0 || topRankCount == 0 {
		r.noSlots = true
		return
	}

	offset %= topRankCount

	var selected []*slot

	// NOTE: ringOffset increments each assign, so we start at that worker and
	// wrap with % if it has no free slot, instead of always packing onto the
	// first worker in the tied group.
	for i := 0; i < topRankCount; i++ {
		workerId := candidates[(offset+i)%topRankCount]

		if sel, ok := selectSlotsFromPools(s.poolsByWorker[workerId], requests, now); ok {
			selected = sel
			break
		}
	}

	if selected == nil {
		// Lower ranks are only considered after the top-rank group has no slots.
		for i := topRankCount; i < len(candidates); i++ {
			workerId := candidates[i]

			if sel, ok := selectSlotsFromPools(s.poolsByWorker[workerId], requests, now); ok {
				selected = sel
				break
			}
		}
	}

	if selected == nil {
		r.noSlots = true
		return
	}

	s.assignedCount++
	r.ackId = s.assignedCount

	s.unackedSlots[r.ackId] = &assignedSlots{
		slots:         selected,
		rateLimitAck:  rateLimitAck,
		rateLimitNack: rateLimitNack,
	}

	r.workerId = selected[0].getWorkerId()
	r.succeeded = true
}

// tryAssignBatchQueueItem assigns a single representative queue item to obtain one worker slot
// for an entire batch flush (v1_batched_queue_item). This is intentionally separate from the
// regular chunk assignment flow.
func (s *Scheduler) tryAssignBatchQueueItem(
	ctx context.Context,
	qi *sqlcv1.V1QueueItem,
	labels []*sqlcv1.GetDesiredLabelsRow,
) (
	res assignSingleResult, err error,
) {
	ctx, span := telemetry.NewSpan(ctx, "try-assign-batch-queue-item")
	defer span.End()

	if qi == nil {
		return res, nil
	}

	if err := ctx.Err(); err != nil {
		return res, err
	}

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "tenant.id", Value: qi.TenantID.String()})

	res.qi = qi

	if isTimedOut(qi) {
		res.noSlots = true
		return res, nil
	}

	// NOTE: batch flush scheduling should not re-evaluate rate limits. Rate limits are evaluated per
	// task before redirecting into the batched queue table.
	noop := func() {}

	if ok := s.do(ctx, func() {
		action, ok := s.actions[qi.ActionID]

		if !ok || action == nil || len(action.workerIds) == 0 {
			res.noSlots = true
			return
		}

		// Default to 1 standard slot — same fallback as tryAssignBatch — since batch flush
		// scheduling skips the regular slot-request lookup path.
		requests := map[string]int32{v1.SlotTypeDefault: 1}

		s.assignSingleton(action, qi, &res, labels, requests, noop, noop, time.Now())
	}); !ok {
		res.noSlots = true
	}

	return res, nil
}

// selectSlotsFromPools reserves the requested units from a single worker's
// pools, or nothing at all. Freelist counts are exact, so the reservation can
// be verified up front and never needs to roll back.
func selectSlotsFromPools(poolsByType map[string]*slotPool, requests map[string]int32, now time.Time) ([]*slot, bool) {
	totalNeeded := 0

	for slotType, units := range requests {
		if units <= 0 {
			continue
		}

		if poolsByType[slotType].freeCountAt(now) < int(units) {
			return nil, false
		}

		totalNeeded += int(units)
	}

	selected := make([]*slot, 0, totalNeeded)

	for slotType, units := range requests {
		for j := int32(0); j < units; j++ {
			selected = append(selected, poolsByType[slotType].take())
		}
	}

	return selected, true
}

// rankWorkerIds runs on the run loop (it reads s.workers for label affinity).
// The returned int is the length of the highest-rank prefix, so assignment can
// round-robin ties without skipping a better match.
func (s *Scheduler) rankWorkerIds(
	qi *sqlcv1.V1QueueItem,
	labels []*sqlcv1.GetDesiredLabelsRow,
	workerIds []uuid.UUID,
) ([]uuid.UUID, int) {
	type rankedWorker struct {
		id   uuid.UUID
		rank int
	}

	ranked := make([]rankedWorker, 0, len(workerIds))
	for _, workerId := range workerIds {
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
				// Label affinity reads worker metadata from s.workers. Do not
				// require a poolsByWorker entry — candidates can be listed on
				// the action before pools are populated.
				worker := s.workers[workerId]
				if worker == nil {
					continue
				}
				rank = worker.computeWeight(labels)
				if rank < 0 {
					continue
				}
			}
		}

		ranked = append(ranked, rankedWorker{id: workerId, rank: rank})
	}

	if len(ranked) == 0 {
		return []uuid.UUID{}, 0
	}

	topRankCount := len(ranked)
	for i := 1; i < len(ranked); i++ {
		if ranked[i].rank != ranked[0].rank {
			slices.SortStableFunc(ranked, func(left, right rankedWorker) int {
				return right.rank - left.rank
			})

			topRankCount = 1
			topRank := ranked[0].rank
			for topRankCount < len(ranked) && ranked[topRankCount].rank == topRank {
				topRankCount++
			}
			break
		}
	}

	result := make([]uuid.UUID, len(ranked))
	for index := range ranked {
		result[index] = ranked[index].id
	}
	return result, topRankCount
}

type assignedQueueItem struct {
	AckId    int
	WorkerId uuid.UUID

	QueueItem *sqlcv1.V1QueueItem
}

type assignResults struct {
	assigned           []*assignedQueueItem
	buffered           []*assignedQueueItem
	batched            []*batchedQueueItemResult
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
	stepIdsToBatchConfig map[string]bool,
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
					batchBuffered := make([]*assignedQueueItem, 0, len(batchQis))
					batchBatched := make([]*batchedQueueItemResult, 0, len(batchQis))

					batchRateLimited := make([]*scheduleRateLimitResult, 0, len(batchQis))
					batchRateLimitedToMove := make([]*scheduleRateLimitResult, 0, len(batchQis))
					batchUnassigned := make([]*sqlcv1.V1QueueItem, 0, len(batchQis))

					batchStart := time.Now()

					results, err := s.tryAssignBatch(ctx, actionId, batchQis, stepIdsToLabels, stepIdsToRequests, taskIdsToRateLimits, stepIdsToBatchConfig, taskIdsToLabelOverrides)

					if err != nil {
						return err
					}

					for _, singleRes := range results {
						if singleRes.toBatch {
							batchBatched = append(batchBatched, &batchedQueueItemResult{
								qi:            singleRes.qi,
								rateLimitAck:  singleRes.rateLimitAck,
								rateLimitNack: singleRes.rateLimitNack,
							})
							continue
						}

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
						buffered:          batchBuffered,
						batched:           batchBatched,
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

func isTimedOut(qi *sqlcv1.V1QueueItem) bool {
	// if the current time is after the scheduleTimeoutAt, then mark this as timed out
	now := time.Now().UTC()
	scheduleTimeoutAt := qi.ScheduleTimeoutAt.Time

	// timed out if the scheduleTimeoutAt is set and the current time is after the scheduleTimeoutAt
	isTimedOut := !scheduleTimeoutAt.IsZero() && scheduleTimeoutAt.Before(now)

	return isTimedOut
}
