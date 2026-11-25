package dispatcher

import (
	"bytes"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"

	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

type workerHeartbeatKey struct {
	tenantId uuid.UUID
	workerId uuid.UUID
}

// pendingHeartbeats holds the latest received heartbeat per (tenant, worker) while
// the database is unavailable. The zero value is ready to use.
type pendingHeartbeats struct {
	mu      sync.Mutex
	entries map[workerHeartbeatKey]time.Time
}

func (p *pendingHeartbeats) record(tenantId, workerId uuid.UUID, at time.Time) int {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.entries == nil {
		p.entries = make(map[workerHeartbeatKey]time.Time)
	}

	key := workerHeartbeatKey{tenantId: tenantId, workerId: workerId}
	if cur, ok := p.entries[key]; !ok || at.After(cur) {
		p.entries[key] = at
	}

	return len(p.entries)
}

func (p *pendingHeartbeats) snapshot() []v1.WorkerHeartbeat {
	p.mu.Lock()
	if len(p.entries) == 0 {
		p.mu.Unlock()
		return nil
	}

	batch := make([]v1.WorkerHeartbeat, 0, len(p.entries))
	for k, at := range p.entries {
		batch = append(batch, v1.WorkerHeartbeat{
			TenantId:        k.tenantId,
			WorkerId:        k.workerId,
			LastHeartbeatAt: at,
		})
	}
	p.mu.Unlock()

	// Order rows deterministically so concurrent flushes from different dispatchers
	// acquire row locks in the same order.
	slices.SortFunc(batch, func(a, b v1.WorkerHeartbeat) int {
		if c := bytes.Compare(a.WorkerId[:], b.WorkerId[:]); c != 0 {
			return c
		}

		return bytes.Compare(a.TenantId[:], b.TenantId[:])
	})

	return batch
}

// removeFlushed drops entries whose timestamp did not advance while the flush was in flight.
func (p *pendingHeartbeats) removeFlushed(batch []v1.WorkerHeartbeat) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, hb := range batch {
		key := workerHeartbeatKey{tenantId: hb.TenantId, workerId: hb.WorkerId}
		if cur, ok := p.entries[key]; ok && !cur.After(hb.LastHeartbeatAt) {
			delete(p.entries, key)
		}
	}
}
