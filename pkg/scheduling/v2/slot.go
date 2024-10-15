package v2

import (
	"sort"

	"github.com/sasha-s/go-deadlock"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

type slot struct {
	worker  *worker
	actions []string

	used bool

	ackd bool

	additionalAcks  []func()
	additionalNacks []func()

	mu deadlock.RWMutex
}

func (s *slot) getWorkerId() string {
	return sqlchelpers.UUIDToStr(s.worker.ID)
}

func (s *slot) active() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return !s.used
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

func (s *slot) nack(additionalNacks ...func()) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.used = false
	s.ackd = false

	for _, nack := range additionalNacks {
		if nack != nil {
			nack()
		}
	}

	s.additionalAcks = nil
	s.additionalNacks = nil
}

type rankedValidSlots struct {
	validSlots      []*slot
	workerSeenCount map[string]int

	// slotRanking is a map of slot index to rank. Ranks of -1 will not count in the final ranking.
	slotRanking map[int]int

	// workerSlotCountRank is used as a secondary ranking for workers with the same rank. It stores a map of
	// slot index to the number of times that worker was seen.
	workerSlotCountRank map[int]int

	// cachedWorkerRanks is a map of worker id to rank.
	cachedWorkerRanks map[string]int
}

func newRankedValidSlots() *rankedValidSlots {
	return &rankedValidSlots{
		validSlots:          make([]*slot, 0),
		workerSeenCount:     make(map[string]int),
		cachedWorkerRanks:   make(map[string]int),
		workerSlotCountRank: make(map[int]int),
		slotRanking:         make(map[int]int),
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

func (r *rankedValidSlots) addSlot(slot *slot, rank int) {
	workerId := slot.getWorkerId()
	index := len(r.validSlots)

	r.validSlots = append(r.validSlots, slot)
	r.slotRanking[index] = rank
	r.workerSlotCountRank[index] = r.workerSeenCount[workerId]
	r.cachedWorkerRanks[workerId] = rank
	r.workerSeenCount[workerId]++
}

func (r *rankedValidSlots) less(i, j int) bool {
	// if we have the same rank, sort by worker seen count
	if r.slotRanking[i] == r.slotRanking[j] {
		return r.workerSlotCountRank[i] > r.workerSlotCountRank[j]
	}

	return r.slotRanking[i] > r.slotRanking[j]
}

func (r *rankedValidSlots) order() []*slot {
	nonNegativeSlots := make([]*slot, 0)

	// remove any slots with a negative rank
	for i, rank := range r.slotRanking {
		if rank >= 0 {
			nonNegativeSlots = append(nonNegativeSlots, r.validSlots[i])
		}
	}

	// sort the slots by rank
	sort.Slice(nonNegativeSlots, r.less)

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

		// if this is a sticky strategy, it can only be assigned to the desired worker if the desired
		// worker id is set. otherwise, it can be assigned to any worker.
		if qi.Sticky.Valid && qi.Sticky.StickyStrategy == dbsqlc.StickyStrategyHARD {
			if qi.DesiredWorkerId.Valid && workerId == sqlchelpers.UUIDToStr(qi.DesiredWorkerId) {
				validSlots.addSlot(slot, 0)
				continue
			}
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
