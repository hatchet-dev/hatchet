package scheduler

import (
	"context"
	"errors"
	"sort"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
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

func TestDispatcherIdCacheSet(t *testing.T) {
	ctx := context.Background()
	tenantId := uuid.New()
	worker := uuid.New()
	oldDispatcher, newDispatcher := uuid.New(), uuid.New()

	lookup := &recordingDispatcherIdLookup{dispatchers: map[uuid.UUID]uuid.UUID{worker: oldDispatcher}}
	cache := newDispatcherIdCache(lookup.lookup)

	result, _, err := cache.get(ctx, tenantId, []uuid.UUID{worker})
	require.NoError(t, err)
	assert.Equal(t, map[uuid.UUID]uuid.UUID{worker: oldDispatcher}, result)

	cache.set(worker, newDispatcher)

	result, _, err = cache.get(ctx, tenantId, []uuid.UUID{worker})
	require.NoError(t, err)
	assert.Equal(t, map[uuid.UUID]uuid.UUID{worker: newDispatcher}, result, "set replaces the cached entry")
	assert.Len(t, lookup.queries, 1, "the replaced entry is answered from the cache, not re-queried")
}

// TestHandleWorkerDispatcherChangedReplacesCachedId routes a worker-dispatcher-changed message through
// the scheduler's message handler and checks the cached dispatcher id is replaced before the TTL.
func TestHandleWorkerDispatcherChangedReplacesCachedId(t *testing.T) {
	ctx := context.Background()
	tenantId := uuid.New()
	worker, otherWorker := uuid.New(), uuid.New()
	oldDispatcher, newDispatcher := uuid.New(), uuid.New()

	lookup := &recordingDispatcherIdLookup{dispatchers: map[uuid.UUID]uuid.UUID{worker: oldDispatcher, otherWorker: oldDispatcher}}
	l := zerolog.Nop()
	s := &Scheduler{l: &l, dispatcherIds: newDispatcherIdCache(lookup.lookup)}

	result, _, err := s.dispatcherIds.get(ctx, tenantId, []uuid.UUID{worker, otherWorker})
	require.NoError(t, err)
	assert.Equal(t, map[uuid.UUID]uuid.UUID{worker: oldDispatcher, otherWorker: oldDispatcher}, result)

	msg, err := tasktypes.NotifyWorkerDispatcherChanged(tenantId, worker, newDispatcher)
	require.NoError(t, err)
	assert.Equal(t, msgqueue.MsgIDWorkerDispatcherChanged, msg.ID)

	require.NoError(t, s.handleTask(ctx, msg))

	result, _, err = s.dispatcherIds.get(ctx, tenantId, []uuid.UUID{worker, otherWorker})
	require.NoError(t, err)
	assert.Equal(t, map[uuid.UUID]uuid.UUID{worker: newDispatcher, otherWorker: oldDispatcher}, result, "only the worker in the message moves")
	assert.Len(t, lookup.queries, 1, "the message replaces the entry without a query")

	// a payload without ids carries nothing to cache and is ignored
	empty, err := msgqueue.NewTenantMessage(tenantId, msgqueue.MsgIDWorkerDispatcherChanged, true, false, tasktypes.WorkerDispatcherChangedPayload{})
	require.NoError(t, err)
	require.NoError(t, s.handleTask(ctx, empty))

	result, _, err = s.dispatcherIds.get(ctx, tenantId, []uuid.UUID{worker})
	require.NoError(t, err)
	assert.Equal(t, map[uuid.UUID]uuid.UUID{worker: newDispatcher}, result)
}
