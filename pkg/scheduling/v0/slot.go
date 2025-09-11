package v0

import (
	"slices"
	"sync"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

// slot expiry is 1 second to account for the 1 second replenish rate, plus 500 ms of buffer
// time for unacked slots to get written back to the database.
const defaultSlotExpiry = 1500 * time.Millisecond

type slot struct {
	worker  *worker
	actions []string

	// expiresAt is when the slot is no longer valid, but has not been cleaned up yet
	expiresAt *time.Time
	used      bool

	ackd bool

	additionalAcks  []func()
	additionalNacks []func()

	mu sync.RWMutex
}

func newSlot(worker *worker, actions []string) *slot {
	expires := time.Now().Add(defaultSlotExpiry)

	return &slot{
		worker:    worker,
		actions:   actions,
		expiresAt: &expires,
	}
}

func (s *slot) getWorkerId() string {
	return s.worker.ID
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
	cachedWorkerRanks map[string]int
}

func newRankedValidSlots() *rankedValidSlots {
	return &rankedValidSlots{
		ranksToSlots:      make(map[int][]*slot),
		cachedWorkerRanks: make(map[string]int),
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

		if r.ranksToSlots[rank] != nil {
			nonNegativeSlots = append(nonNegativeSlots, r.ranksToSlots[rank]...)
		}
	}

	return nonNegativeSlots
}

// getRankedSlots returns a list of valid slots sorted by preference, discarding any slots that cannot
// match the affinity conditions.
func getRankedSlots(
	qi *dbsqlc.QueueItem,
	labels []*dbsqlc.GetDesiredLabelsRow,
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
		if qi.Sticky.Valid && qi.Sticky.StickyStrategy == dbsqlc.StickyStrategyHARD {
			if qi.DesiredWorkerId.Valid && workerId == sqlchelpers.UUIDToStr(qi.DesiredWorkerId) {
				validSlots.addSlot(slot, 0)
			} else if !qi.DesiredWorkerId.Valid {
				validSlots.addSlot(slot, 0)
			}

			continue
		}

		// if this is a SOFT sticky strategy, we should prefer the desired worker, but if it is not
		// available, we can assign to any worker.
		if qi.Sticky.Valid && qi.Sticky.StickyStrategy == dbsqlc.StickyStrategySOFT {
			if qi.DesiredWorkerId.Valid && workerId == sqlchelpers.UUIDToStr(qi.DesiredWorkerId) {
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
