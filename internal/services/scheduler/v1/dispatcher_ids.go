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
	dispatcherIdCacheTTL  = time.Second
)

type dispatcherIdLookup func(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) (map[uuid.UUID]uuid.UUID, map[uuid.UUID]struct{}, error)

// dispatcherIdCache answers worker to dispatcher lookups from entries seen within the last second and
// queries only the workers it holds no entry for. Entries expire on the TTL alone: a worker that moves
// to another dispatcher is seen within a second, the same window its liveness check allows.
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
