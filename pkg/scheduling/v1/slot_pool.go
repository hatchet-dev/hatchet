package v1

import (
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

type poolKey struct {
	slotType string
	workerId uuid.UUID
}

// slotPool is the single owner of scheduling capacity for one worker and slot
// type. Actions index workers; they do not copy these slot slices.
type slotPool struct {
	refreshedAt time.Time
	worker      *worker
	slotType    string
	slots       []*slot
	unused      atomic.Int64
}

func (p *slotPool) unusedCount() int {
	if p == nil {
		return 0
	}
	return int(p.unused.Load())
}

func (p *slotPool) resetSlots(slots []*slot) {
	p.slots = slots
	p.refreshedAt = time.Now()

	unused := int64(0)
	for _, sl := range slots {
		sl.pool = p
		if !sl.isUsed() {
			unused++
		}
	}
	p.unused.Store(unused)
}

func (p *slotPool) staleAt(now time.Time) bool {
	return p == nil || p.refreshedAt.IsZero() || !p.refreshedAt.Add(defaultSlotExpiry).After(now)
}
