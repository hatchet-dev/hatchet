package v1

import (
	"time"

	"github.com/google/uuid"
)

// slot expiry is 1.5 seconds to account for the 1 second replenish rate, plus 500 ms of
// buffer time for unacked slots to get written back to the database.
const defaultSlotExpiry = 1500 * time.Millisecond

type poolKey struct {
	slotType string
	workerId uuid.UUID
}

// slotPool is the single owner of scheduling capacity for one worker and slot
// type. Actions index workers; they do not copy these slot slices.
//
// NOTE: slotPool is not concurrency-safe on its own. It is only ever read or
// written from the Scheduler's run loop goroutine (via ops sent through do /
// mustDo), which is what makes the lock-free freelist sound. Do not touch a
// pool from any other goroutine.
//
// Expiry is pool-level: every slot in a pool is built from the same replenish
// read, so the whole pool goes stale together. The freelist is exact — only the
// run loop mutates it, and expiry invalidates the pool as a whole rather than
// draining individual slots — so free counts never drift.
type slotPool struct {
	worker    *worker
	slotType  string
	expiresAt time.Time

	slots []*slot

	// free is a stack of indexes into slots that are not in use.
	free []int
}

func (p *slotPool) staleAt(now time.Time) bool {
	return p == nil || !p.expiresAt.After(now)
}

// freeCountAt is the exact number of assignable slots; zero when stale.
func (p *slotPool) freeCountAt(now time.Time) int {
	if p.staleAt(now) {
		return 0
	}

	return len(p.free)
}

// take pops a free slot and marks it used. Callers must have checked staleness
// and free count first.
func (p *slotPool) take() *slot {
	n := len(p.free)
	if n == 0 {
		return nil
	}

	idx := p.free[n-1]
	p.free = p.free[:n-1]

	sl := p.slots[idx]
	sl.used = true

	return sl
}

// release returns a used slot to the freelist (assignment nack).
func (p *slotPool) release(sl *slot) {
	if !sl.used {
		return
	}

	sl.used = false
	p.free = append(p.free, sl.idx)
}

// reset replaces the pool's slots and rebuilds the freelist. Used slots
// (unacked assignments retained across a replenish) stay out of the freelist.
func (p *slotPool) reset(slots []*slot, expiresAt time.Time) {
	p.slots = slots
	p.expiresAt = expiresAt
	p.free = p.free[:0]

	for i, sl := range slots {
		sl.pool = p
		sl.idx = i

		if !sl.used {
			p.free = append(p.free, i)
		}
	}
}
