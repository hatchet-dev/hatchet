package v2

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

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
	return d.queries.ListActionsForWorkers(ctx, d.pool, dbsqlc.ListActionsForWorkersParams{
		Tenantid:  d.tenantId,
		Workerids: workerIds,
	})
}

func (d *schedulerDbQueries) ListAvailableSlotsForWorkers(ctx context.Context, params dbsqlc.ListAvailableSlotsForWorkersParams) ([]*dbsqlc.ListAvailableSlotsForWorkersRow, error) {
	return d.queries.ListAvailableSlotsForWorkers(ctx, d.pool, params)
}

// Scheduler is responsible for scheduling steps to workers as efficiently as possible.
// This is tenant-scoped, so each tenant will have its own scheduler.
type Scheduler struct {
	repo     schedulerRepo
	tenantId pgtype.UUID

	l *zerolog.Logger

	actions     map[string]*action
	actionsMu   sync.RWMutex
	replenishMu sync.Mutex

	workersMu sync.Mutex
	workers   map[string]*worker

	assignedCount   int
	assignedCountMu sync.Mutex

	// unackedSlots are slots which have been assigned to a worker, but have not been flushed
	// to the database yet. They negatively count towards a worker's available slot count.
	unackedSlots map[int]*slot
	unackedMu    sync.Mutex

	rl *rateLimiter
}

func newScheduler(cf *sharedConfig, tenantId pgtype.UUID, rl *rateLimiter) *Scheduler {
	return &Scheduler{
		repo:         newSchedulerDbQueries(cf.queries, cf.pool, tenantId),
		tenantId:     tenantId,
		l:            cf.l,
		actions:      make(map[string]*action),
		unackedSlots: make(map[int]*slot),
		rl:           rl,
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

		if worker.AvailableSlots > 0 {

			workerId := sqlchelpers.UUIDToStr(worker.ID)
			actions := workerIdsToActions[workerId]
			unackedSlots := workersToUnackedSlots[workerId]

			// create a slot for each available slot
			slots := make([]*slot, 0, int(worker.AvailableSlots))

			for i := 0; i < int(worker.AvailableSlots)-len(unackedSlots); i++ {
				slots = append(slots, &slot{
					actions: actions,
					worker:  workers[workerId],
				})
			}

			slots = append(slots, unackedSlots...)

			for _, actionId := range actions {
				actionsToNewSlots[actionId] = append(actionsToNewSlots[actionId], slots...)
				actionsToTotalSlots[actionId] += len(slots)
			}
		} else {
			s.l.Warn().Msgf("worker %s has no available slots", sqlchelpers.UUIDToStr(worker.ID))
		}
	}

	s.actionsMu.Lock()
	defer s.actionsMu.Unlock()

	// first pass: write all actions with new slots to the scheduler
	for actionId, newSlots := range actionsToNewSlots {
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
		if len(storedAction.slots) == 0 {
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

type assignSingleResult struct {
	workerId pgtype.UUID
	ackId    int

	noSlots   bool
	succeeded bool

	rateLimitResult *rateLimitResult
}

// tryAssignSingleton attempts to assign a singleton step to a worker.
func (s *Scheduler) tryAssignSingleton(
	ctx context.Context,
	qi *dbsqlc.QueueItem,
	// ringOffset is a hint for where to start the search for a slot. The search will wraparound the ring if necessary.
	// If a slot is assigned, the caller should increment this value for the next call to tryAssignSingleton.
	// Note that this is not guaranteed to be the actual offset of the latest assigned slot, since many actions may be scheduling
	// slots concurrently.
	ringOffset int,
	labels []*dbsqlc.GetDesiredLabelsRow,
	rls map[string]int32,
) (
	res assignSingleResult, err error,
) {
	if !qi.ActionId.Valid {
		return res, fmt.Errorf("queue item does not have a valid action id")
	}

	actionId := qi.ActionId.String

	var rateLimitAck func()
	var rateLimitNack func()

	// check rate limits
	if len(rls) > 0 {
		rlResult := s.rl.use(ctx, sqlchelpers.UUIDToStr(qi.StepRunId), rls)

		if !rlResult.succeeded {
			res.rateLimitResult = &rlResult
			return res, nil
		}
	}

	// pick a worker to assign the slot to
	var assignedSlot *slot

	s.actionsMu.RLock()
	if _, ok := s.actions[actionId]; !ok {
		s.actionsMu.RUnlock()
		res.noSlots = true
		return res, nil
	}

	candidateSlots := s.actions[actionId].slots

	ringOffset %= len(candidateSlots)

	// rotate the ring to the offset
	candidateSlots = append(candidateSlots[ringOffset:], candidateSlots[:ringOffset]...)

	s.actionsMu.RUnlock()

	candidateSlots = getRankedSlots(qi, labels, candidateSlots)

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

type erroredQueueItem struct {
	QueueItem *dbsqlc.QueueItem
	Err       error
}

type assignResults struct {
	assigned           []*AssignedQueueItem
	errored            []*erroredQueueItem
	unassigned         []*dbsqlc.QueueItem
	schedulingTimedOut []*dbsqlc.QueueItem
	rateLimited        []*rateLimitResult
}

func (s *Scheduler) tryAssign(
	ctx context.Context,
	qis []*dbsqlc.QueueItem,
	stepIdsToLabels map[string][]*dbsqlc.GetDesiredLabelsRow,
	stepRunIdsToRateLimits map[string]map[string]int32,
) <-chan *assignResults {
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
				errored := make([]*erroredQueueItem, 0, len(qis))
				unassigned := make([]*dbsqlc.QueueItem, 0, len(qis))
				schedulingTimedOut := make([]*dbsqlc.QueueItem, 0, len(qis))
				rateLimited := make([]*rateLimitResult, 0, len(qis))

				startAssignment := time.Now()

				ringOffset := 0

				for i := range qis {
					qi := qis[i]

					if isTimedOut(qi) {
						schedulingTimedOut = append(schedulingTimedOut, qi)
						continue
					}

					labels := make([]*dbsqlc.GetDesiredLabelsRow, 0)

					if stepIdsToLabels != nil {
						if _, ok := stepIdsToLabels[sqlchelpers.UUIDToStr(qi.StepId)]; ok {
							labels = stepIdsToLabels[sqlchelpers.UUIDToStr(qi.StepId)]
						}
					}

					rls := make(map[string]int32)

					if stepRunIdsToRateLimits != nil {
						if _, ok := stepRunIdsToRateLimits[sqlchelpers.UUIDToStr(qi.StepRunId)]; ok {
							rls = stepRunIdsToRateLimits[sqlchelpers.UUIDToStr(qi.StepRunId)]
						}
					}

					singleRes, err := s.tryAssignSingleton(ctx, qi, ringOffset, labels, rls)

					if err != nil {
						s.l.Error().Err(err).Msg("error assigning queue item")
					}

					if !singleRes.succeeded {
						if singleRes.rateLimitResult != nil {
							rateLimited = append(rateLimited, singleRes.rateLimitResult)
							continue
						} else if singleRes.noSlots {
							unassigned = append(unassigned, qi)
							break
						}
					}

					ringOffset++

					assigned = append(assigned, &AssignedQueueItem{
						WorkerId:  singleRes.workerId,
						QueueItem: qi,
						AckId:     singleRes.ackId,
					})
				}

				endAssignment := time.Now()

				if sinceStart := endAssignment.Sub(startAssignment); sinceStart > 100*time.Millisecond {
					s.l.Warn().Msgf("assignment of %d queue items took longer than 100ms (%v)", len(qis), sinceStart.String())
				}

				resultsCh <- &assignResults{
					assigned:           assigned,
					errored:            errored,
					unassigned:         unassigned,
					schedulingTimedOut: schedulingTimedOut,
					rateLimited:        rateLimited,
				}
			}(actionId, qis)
		}

		wg.Wait()
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
