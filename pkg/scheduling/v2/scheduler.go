package v2

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sasha-s/go-deadlock"

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
	actionsMu   deadlock.RWMutex
	replenishMu deadlock.Mutex

	workerIdsMu deadlock.Mutex
	workerIds   []pgtype.UUID

	// unackedSlots are slots which have been assigned to a worker, but have not been flushed
	// to the database yet. They negatively count towards a worker's available slot count.
	unackedSlots map[int]*slot
	unackedMu    deadlock.Mutex

	assignedCount   int
	assignedCountMu deadlock.Mutex
}

func newScheduler(cf *sharedConfig, tenantId pgtype.UUID) *Scheduler {
	return &Scheduler{
		repo:         newSchedulerDbQueries(cf.queries, cf.pool, tenantId),
		tenantId:     tenantId,
		l:            cf.l,
		actions:      make(map[string]*action),
		unackedSlots: make(map[int]*slot),
	}
}

type action struct {
	lastReplenishedSlotCount   int
	lastReplenishedWorkerCount int

	// note that slots can be used across multiple actions, hence the pointer
	slots []*slot
}

func (a *action) activeCount() int {
	count := 0

	for _, slot := range a.slots {
		if slot.active() {
			count++
		}
	}

	return count
}

type slot struct {
	workerId string
	actions  []string

	// expiresAt is when the slot is no longer valid, but has not been cleaned up yet
	expiresAt *time.Time
	used      bool

	ackd bool

	mu deadlock.RWMutex
}

func (s *slot) active() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return !s.used && s.expiresAt != nil && s.expiresAt.After(time.Now())
}

func (s *slot) use() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.used {
		return false
	}

	s.used = true
	s.ackd = false

	return true
}

func (s *slot) ack() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ackd = true
}

func (s *slot) nack() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.used = false
	s.ackd = false
}

// TODO: ACK IN BULK!
func (s *Scheduler) ack(id int) {
	s.unackedMu.Lock()
	defer s.unackedMu.Unlock()

	if slot, ok := s.unackedSlots[id]; ok {
		slot.ack()
		delete(s.unackedSlots, id)
	}
}

// TODO: NACK IN BULK!
func (s *Scheduler) nack(id int) {
	s.unackedMu.Lock()
	defer s.unackedMu.Unlock()

	if slot, ok := s.unackedSlots[id]; ok {
		slot.nack()
		delete(s.unackedSlots, id)
	}
}

func (s *Scheduler) setWorkerIds(workerIds []pgtype.UUID) {
	s.workerIdsMu.Lock()
	defer s.workerIdsMu.Unlock()

	s.workerIds = workerIds
}

func (s *Scheduler) getWorkerIds() []pgtype.UUID {
	s.workerIdsMu.Lock()
	defer s.workerIdsMu.Unlock()

	return s.workerIds
}

// replenish loads new slots from the database.
func (s *Scheduler) replenish(ctx context.Context) error {
	if ok := s.replenishMu.TryLock(); !ok {
		return nil
	}

	defer s.replenishMu.Unlock()

	workersToActiveActions, err := s.repo.ListActionsForWorkers(ctx, s.getWorkerIds())

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

	// TODO: CHECK CORRECTNESS OF UNACKED SLOTS, PERHAPS WE SHOULD LOCK THIS FOR THE ENTIRE REPLENISH?
	s.unackedMu.Lock()
	defer s.unackedMu.Unlock()

	for _, unackedSlot := range s.unackedSlots {
		s := unackedSlot
		if _, ok := workersToUnackedSlots[s.workerId]; !ok {
			workersToUnackedSlots[s.workerId] = make([]*slot, 0)
		}

		workersToUnackedSlots[s.workerId] = append(workersToUnackedSlots[s.workerId], s)
	}

	// FUNCTION 4: write the new slots to the scheduler and clean up expired slots
	now := time.Now()
	expires := now.Add(5 * time.Second)
	actionsToNewSlots := make(map[string][]*slot)
	actionsToTotalSlots := make(map[string]int)

	for _, worker := range availableSlots {
		workerId := sqlchelpers.UUIDToStr(worker.ID)
		actions := workerIdsToActions[workerId]
		unackedSlots := workersToUnackedSlots[workerId]

		// create a slot for each available slot
		slots := make([]*slot, 0, int(worker.AvailableSlots))

		for i := 0; i < int(worker.AvailableSlots)-len(unackedSlots); i++ {
			slots = append(slots, &slot{
				actions:   actions,
				workerId:  workerId,
				expiresAt: &expires,
			})
		}

		slots = append(slots, unackedSlots...)

		for _, actionId := range actions {
			actionsToNewSlots[actionId] = append(actionsToNewSlots[actionId], slots...)
			actionsToTotalSlots[actionId] += len(slots)
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
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := s.replenish(ctx)

			if err != nil {
				s.l.Error().Err(err).Msg("error replenishing slots")
			}
		}
	}
}

func (s *Scheduler) start(ctx context.Context) {
	go s.loopReplenish(ctx)
}

var ErrNoSlotsAvailable = fmt.Errorf("no slots available")
var ErrAlreadyAssigned = fmt.Errorf("queue item already assigned")

// tryAssignSingleton attempts to assign a singleton step to a worker.
func (s *Scheduler) tryAssignSingleton(ctx context.Context, qi *dbsqlc.QueueItem, skip int) (
	workerId pgtype.UUID, ackId int, err error,
) {
	if !qi.ActionId.Valid {
		return workerId, ackId, fmt.Errorf("queue item does not have a valid action id")
	}

	actionId := qi.ActionId.String

	// TODO: check rate limits

	// pick a worker to assign the slot to
	var assignedSlot *slot

	s.actionsMu.RLock()
	if _, ok := s.actions[actionId]; !ok {
		s.actionsMu.RUnlock()
		return workerId, ackId, ErrNoSlotsAvailable
	}

	candidateSlots := s.actions[actionId].slots
	s.actionsMu.RUnlock()

	startIterating := time.Now()

	for i := skip; i < len(candidateSlots); i++ {
		slot := candidateSlots[i]

		if !slot.active() {
			continue
		}

		if !slot.use() {
			continue
		}

		assignedSlot = slot
		break
	}

	if assignedSlot == nil {
		return workerId, ackId, ErrNoSlotsAvailable
	}

	endIterating := time.Now()

	s.l.Warn().Msgf("iteration took %v", endIterating.Sub(startIterating))

	s.assignedCountMu.Lock()
	s.assignedCount++
	ackId = s.assignedCount
	s.assignedCountMu.Unlock()

	s.unackedMu.Lock()
	s.unackedSlots[ackId] = assignedSlot
	s.unackedMu.Unlock()

	res := sqlchelpers.UUIDFromStr(assignedSlot.workerId)

	return res, ackId, nil
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

	// TODO: return rate limited slots
	// rateLimited []*dbsqlc.QueueItem
}

func (s *Scheduler) tryAssign(ctx context.Context, qis []*dbsqlc.QueueItem) <-chan *assignResults {
	// fmt.Println("TRY ASSIGN", len(qis))

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

				startAssignment := time.Now()

				// TODO: ADD A NOTE ABOUT SKIPPING SINCE THERE MIGHT BE CONTENTION ON THE CANDIDATE SLOTS
				skip := 0

				for i := range qis {
					qi := qis[i]

					if isTimedOut(qi) {
						schedulingTimedOut = append(schedulingTimedOut, qi)
						continue
					}

					workerId, ackId, err := s.tryAssignSingleton(ctx, qi, skip)

					if err != nil {
						if err == ErrNoSlotsAvailable {
							unassigned = append(unassigned, qi)
						} else {
							errored = append(errored, &erroredQueueItem{
								QueueItem: qi,
								Err:       err,
							})
						}

						// if we can't assign, we break out of the loop
						// TODO: change this once we have rate limits!
						break
					}

					skip += 1

					assigned = append(assigned, &AssignedQueueItem{
						WorkerId:  workerId,
						QueueItem: qi,
						AckId:     ackId,
					})
				}

				endAssignment := time.Now()

				s.l.Warn().Msgf("assignment of %d queue items took %v", len(qis), endAssignment.Sub(startAssignment))

				resultsCh <- &assignResults{
					assigned:           assigned,
					errored:            errored,
					unassigned:         unassigned,
					schedulingTimedOut: schedulingTimedOut,
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
