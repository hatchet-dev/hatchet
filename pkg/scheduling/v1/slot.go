package v1

import (
	"github.com/google/uuid"
)

// slot is a unit of schedulable capacity on a worker. Slots are owned by a
// slotPool and are only ever touched from the scheduler's run loop, so they
// carry no synchronization.
type slot struct {
	worker   *worker
	pool     *slotPool
	slotType string
	idx      int
	used     bool
}

func (s *slot) getWorkerId() uuid.UUID {
	return s.worker.ID
}

// assignedSlots tracks the slots reserved for a single assignment, plus the
// rate-limit callbacks which must fire when the assignment is acked or nacked.
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
