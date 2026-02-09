package v1

import (
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// slot expiry is 1 second to account for the 1 second replenish rate, plus 500 ms of buffer
// time for unacked slots to get written back to the database.
const defaultSlotExpiry = 1500 * time.Millisecond

// slotMeta is shared across many slots to avoid duplicating
// metadata that is identical for a worker/type.
type slotMeta struct {
	slotType string
	actions  []string
}

func newSlotMeta(actions []string, slotType string) *slotMeta {
	return &slotMeta{
		actions:  actions,
		slotType: slotType,
	}
}

type slot struct {
	worker          *worker
	meta            *slotMeta
	expiresAt       *time.Time
	additionalAcks  []func()
	additionalNacks []func()
	mu              sync.RWMutex
	used            bool
	ackd            bool
}

type assignedSlots struct {
	slots         []*slot
	rateLimitAck  func()
	rateLimitNack func()
}

func (a *assignedSlots) workerId() uuid.UUID {
	if len(a.slots) == 0 {
		return uuid.Nil
	}

	return a.slots[0].getWorkerId()
}

func (a *assignedSlots) ack() {
	for _, slot := range a.slots {
		slot.ack()
	}
	if a.rateLimitAck != nil {
		a.rateLimitAck()
	}
}

func (a *assignedSlots) nack() {
	for _, slot := range a.slots {
		slot.nack()
	}
	if a.rateLimitNack != nil {
		a.rateLimitNack()
	}
}

func newSlot(worker *worker, meta *slotMeta) *slot {
	expires := time.Now().Add(defaultSlotExpiry)

	return &slot{
		worker:    worker,
		meta:      meta,
		expiresAt: &expires,
	}
}

func (s *slot) getWorkerId() uuid.UUID {
	return s.worker.ID
}

func (s *slot) getSlotType() string {
	if s.meta == nil {
		return ""
	}

	return s.meta.slotType
}

func (s *slot) extendExpiry() {
	s.mu.Lock()
	defer s.mu.Unlock()

	expires := time.Now().Add(defaultSlotExpiry)
	s.expiresAt = &expires
}

func (s *slot) active() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return !s.used && s.expiresAt != nil && s.expiresAt.After(time.Now())
}

func (s *slot) isUsed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.used
}

func (s *slot) expired() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.expiresAt == nil || s.expiresAt.Before(time.Now())
}

func (s *slot) use(additionalAcks []func(), additionalNacks []func()) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.used {
		return false
	}

	s.used = true
	s.ackd = false
	s.additionalAcks = additionalAcks
	s.additionalNacks = additionalNacks

	return true
}

func (s *slot) ack() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ackd = true

	for _, ack := range s.additionalAcks {
		if ack != nil {
			ack()
		}
	}

	s.additionalAcks = nil
	s.additionalNacks = nil
}

func (s *slot) nack() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.used = false
	s.ackd = true

	for _, nack := range s.additionalNacks {
		if nack != nil {
			nack()
		}
	}

	s.additionalAcks = nil
	s.additionalNacks = nil
}

type rankedValidSlots struct {
	ranksToSlots map[int][]*slot

	// cachedWorkerRanks is a map of worker id to rank.
	cachedWorkerRanks map[uuid.UUID]int
}

func newRankedValidSlots() *rankedValidSlots {
	return &rankedValidSlots{
		ranksToSlots:      make(map[int][]*slot),
		cachedWorkerRanks: make(map[uuid.UUID]int),
	}
}

func (r *rankedValidSlots) addFromCache(slot *slot) bool {
	workerId := slot.getWorkerId()

	if rank, ok := r.cachedWorkerRanks[workerId]; ok {
		r.addSlot(slot, rank)
		return true
	}

	return false
}

func (r *rankedValidSlots) addSlot(s *slot, rank int) {
	if _, ok := r.ranksToSlots[rank]; !ok {
		r.ranksToSlots[rank] = make([]*slot, 0)
	}

	r.ranksToSlots[rank] = append(r.ranksToSlots[rank], s)
}

func (r *rankedValidSlots) order() []*slot {
	nonNegativeSlots := make([]*slot, 0)

	sortedRanks := make([]int, 0, len(r.ranksToSlots))

	for rank := range r.ranksToSlots {
		sortedRanks = append(sortedRanks, rank)
	}

	slices.Sort(sortedRanks)

	// iterate through sortedRanks in reverse order
	for i := len(sortedRanks) - 1; i >= 0; i-- {
		rank := sortedRanks[i]

		if rank < 0 {
			// skip negative ranks, as they are not valid for scheduling
			continue
		}

		if r.ranksToSlots[rank] != nil {
			nonNegativeSlots = append(nonNegativeSlots, r.ranksToSlots[rank]...)
		}
	}

	return nonNegativeSlots
}

// getRankedSlots returns a list of valid slots sorted by preference, discarding any slots that cannot
// match the affinity conditions.
func getRankedSlots(
	qi *sqlcv1.V1QueueItem,
	labels []*sqlcv1.GetDesiredLabelsRow,
	slots []*slot,
) []*slot {
	validSlots := newRankedValidSlots()

	for _, slot := range slots {
		workerId := slot.getWorkerId()

		if validSlots.addFromCache(slot) {
			continue
		}

		// if this is a HARD sticky strategy, and there's a desired worker id, it can only be assigned to that
		// worker. if there's no desired worker id, we assign to any worker.
		if qi.Sticky == sqlcv1.V1StickyStrategyHARD {
			if qi.DesiredWorkerID != nil && workerId == *qi.DesiredWorkerID {
				validSlots.addSlot(slot, 0)
			} else if qi.DesiredWorkerID == nil {
				validSlots.addSlot(slot, 0)
			}

			continue
		}

		// if this is a SOFT sticky strategy, we should prefer the desired worker, but if it is not
		// available, we can assign to any worker.
		if qi.Sticky == sqlcv1.V1StickyStrategySOFT {
			if qi.DesiredWorkerID != nil && workerId == *qi.DesiredWorkerID {
				validSlots.addSlot(slot, 1)
			} else {
				validSlots.addSlot(slot, 0)
			}

			continue
		}

		// if this step has affinity labels, check if the worker has the desired labels, and rank by
		// the given affinity
		if len(labels) > 0 {
			weight := slot.worker.computeWeight(labels)
			validSlots.addSlot(slot, weight)
			continue
		}

		// if this step has no sticky strategy or affinity labels, add the slot with rank 0
		validSlots.addSlot(slot, 0)
	}

	return validSlots.order()
}
