package scheduler

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/golang-lru/v2/expirable"

	"github.com/hatchet-dev/hatchet/internal/listutils"
)

const (
	dispatcherIdCacheSize = 10000
	// The dispatcher publishes a worker's new dispatcher id when the worker connects to it, so an entry
	// is replaced on the event. The TTL is the safety net for a lost publish and matches the worker
	// liveness window, after which a worker that stopped heartbeating is no longer assigned to anyway.
	dispatcherIdCacheTTL = 5 * time.Second
)

type dispatcherIdLookup func(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) (map[uuid.UUID]uuid.UUID, map[uuid.UUID]struct{}, error)

// dispatcherIdCache answers worker to dispatcher lookups from entries seen within the TTL and queries
// only the workers it holds no entry for. A worker that moves to another dispatcher is written through
// set when the dispatcher's change message arrives; otherwise its entry is replaced once it expires.
type dispatcherIdCache struct {
	ids    *expirable.LRU[uuid.UUID, uuid.UUID]
	lookup dispatcherIdLookup
}

func newDispatcherIdCache(lookup dispatcherIdLookup) *dispatcherIdCache {
	return &dispatcherIdCache{
		ids:    expirable.NewLRU[uuid.UUID, uuid.UUID](dispatcherIdCacheSize, nil, dispatcherIdCacheTTL),
		lookup: lookup,
	}
}

// set records the dispatcher a worker is now connected to, replacing whatever the cache held for it.
func (c *dispatcherIdCache) set(workerId, dispatcherId uuid.UUID) {
	c.ids.Add(workerId, dispatcherId)
}

func (c *dispatcherIdCache) get(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) (map[uuid.UUID]uuid.UUID, map[uuid.UUID]struct{}, error) {
	workerIdToDispatcherId := make(map[uuid.UUID]uuid.UUID, len(workerIds))
	uncached := make([]uuid.UUID, 0)

	for _, workerId := range listutils.Uniq(workerIds) {
		if dispatcherId, ok := c.ids.Get(workerId); ok {
			workerIdToDispatcherId[workerId] = dispatcherId
		} else {
			uncached = append(uncached, workerId)
		}
	}

	if len(uncached) == 0 {
		return workerIdToDispatcherId, map[uuid.UUID]struct{}{}, nil
	}

	found, workersWithoutDispatchers, err := c.lookup(ctx, tenantId, uncached)

	if err != nil {
		return nil, nil, err
	}

	for workerId, dispatcherId := range found {
		c.ids.Add(workerId, dispatcherId)
		workerIdToDispatcherId[workerId] = dispatcherId
	}

	return workerIdToDispatcherId, workersWithoutDispatchers, nil
}
