package scheduler

import (
	"context"
	"errors"
	"sort"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingDispatcherIdLookup answers from a fixed worker to dispatcher map and records the worker ids
// of every query it receives.
type recordingDispatcherIdLookup struct {
	dispatchers map[uuid.UUID]uuid.UUID
	queries     [][]uuid.UUID
	err         error
}

func (l *recordingDispatcherIdLookup) lookup(_ context.Context, _ uuid.UUID, workerIds []uuid.UUID) (map[uuid.UUID]uuid.UUID, map[uuid.UUID]struct{}, error) {
	sorted := append([]uuid.UUID(nil), workerIds...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].String() < sorted[j].String() })
	l.queries = append(l.queries, sorted)

	if l.err != nil {
		return nil, nil, l.err
	}

	found := make(map[uuid.UUID]uuid.UUID)

	for _, workerId := range workerIds {
		if dispatcherId, ok := l.dispatchers[workerId]; ok {
			found[workerId] = dispatcherId
		}
	}

	return found, map[uuid.UUID]struct{}{}, nil
}

func sortedIds(ids ...uuid.UUID) []uuid.UUID {
	sorted := append([]uuid.UUID(nil), ids...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].String() < sorted[j].String() })
	return sorted
}

func TestDispatcherIdCache(t *testing.T) {
	ctx := context.Background()
	tenantId := uuid.New()
	worker1, worker2, worker3 := uuid.New(), uuid.New(), uuid.New()
	dispatcher1, dispatcher2 := uuid.New(), uuid.New()

	newLookup := func() *recordingDispatcherIdLookup {
		return &recordingDispatcherIdLookup{dispatchers: map[uuid.UUID]uuid.UUID{
			worker1: dispatcher1,
			worker2: dispatcher2,
			worker3: dispatcher1,
		}}
	}

	t.Run("a second call within the TTL issues no query for cached workers", func(t *testing.T) {
		lookup := newLookup()
		cache := newDispatcherIdCache(lookup.lookup)

		first, _, err := cache.get(ctx, tenantId, []uuid.UUID{worker1, worker2, worker1})
		require.NoError(t, err)
		assert.Equal(t, map[uuid.UUID]uuid.UUID{worker1: dispatcher1, worker2: dispatcher2}, first)
		require.Len(t, lookup.queries, 1)
		assert.Equal(t, sortedIds(worker1, worker2), lookup.queries[0], "duplicate worker ids are queried once")

		second, _, err := cache.get(ctx, tenantId, []uuid.UUID{worker1, worker2})
		require.NoError(t, err)
		assert.Equal(t, first, second)
		assert.Len(t, lookup.queries, 1, "the second call is answered from the cache")
	})

	t.Run("a mixed call queries only the misses", func(t *testing.T) {
		lookup := newLookup()
		cache := newDispatcherIdCache(lookup.lookup)

		_, _, err := cache.get(ctx, tenantId, []uuid.UUID{worker1})
		require.NoError(t, err)

		result, _, err := cache.get(ctx, tenantId, []uuid.UUID{worker1, worker2, worker3})
		require.NoError(t, err)
		assert.Equal(t, map[uuid.UUID]uuid.UUID{worker1: dispatcher1, worker2: dispatcher2, worker3: dispatcher1}, result)
		require.Len(t, lookup.queries, 2)
		assert.Equal(t, sortedIds(worker2, worker3), lookup.queries[1])

		_, _, err = cache.get(ctx, tenantId, []uuid.UUID{worker1, worker2, worker3})
		require.NoError(t, err)
		assert.Len(t, lookup.queries, 2, "every worker is cached after the mixed call")
	})

	t.Run("a worker without a dispatcher is not cached and is queried again", func(t *testing.T) {
		lookup := newLookup()
		cache := newDispatcherIdCache(lookup.lookup)
		unknown := uuid.New()

		result, _, err := cache.get(ctx, tenantId, []uuid.UUID{worker1, unknown})
		require.NoError(t, err)
		assert.Equal(t, map[uuid.UUID]uuid.UUID{worker1: dispatcher1}, result)

		_, _, err = cache.get(ctx, tenantId, []uuid.UUID{worker1, unknown})
		require.NoError(t, err)
		require.Len(t, lookup.queries, 2)
		assert.Equal(t, []uuid.UUID{unknown}, lookup.queries[1])
	})

	t.Run("a lookup error is returned and nothing is cached", func(t *testing.T) {
		lookup := newLookup()
		lookup.err = errors.New("boom")
		cache := newDispatcherIdCache(lookup.lookup)

		_, _, err := cache.get(ctx, tenantId, []uuid.UUID{worker1})
		require.ErrorIs(t, err, lookup.err)

		lookup.err = nil

		result, _, err := cache.get(ctx, tenantId, []uuid.UUID{worker1})
		require.NoError(t, err)
		assert.Equal(t, map[uuid.UUID]uuid.UUID{worker1: dispatcher1}, result)
		assert.Len(t, lookup.queries, 2)
	})
}
