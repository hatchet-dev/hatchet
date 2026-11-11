package scheduler

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/golang-lru/v2/expirable"

	"github.com/hatchet-dev/hatchet/internal/enumerables"
)

const (
	dispatcherIdCacheSize = 10000
	// only a fallback for a lost worker-dispatcher-changed message
	dispatcherIdCacheTTL = 5 * time.Second
)

type dispatcherIdLookup func(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) (map[uuid.UUID]uuid.UUID, map[uuid.UUID]struct{}, error)

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

func (c *dispatcherIdCache) set(workerId, dispatcherId uuid.UUID) {
	c.ids.Add(workerId, dispatcherId)
}

func (c *dispatcherIdCache) get(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) (map[uuid.UUID]uuid.UUID, map[uuid.UUID]struct{}, error) {
	workerIdToDispatcherId := make(map[uuid.UUID]uuid.UUID, len(workerIds))
	uncached := make([]uuid.UUID, 0)

	for _, workerId := range enumerables.Uniq(workerIds) {
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
