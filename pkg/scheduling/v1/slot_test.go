//go:build !e2e && !load && !rampup && !integration

package v1

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func newSlot(worker *worker, slotType string) *slot {
	return &slot{worker: worker, slotType: slotType}
}

var stableWorkerId1 = uuid.New()
var stableWorkerId2 = uuid.New()

func TestSlotPool_TakeRelease(t *testing.T) {
	workerId := uuid.New()
	w := &worker{ListActiveWorkersResult: &v1.ListActiveWorkersResult{ID: workerId}}

	pool := &slotPool{worker: w, slotType: "cpu"}
	pool.reset([]*slot{newSlot(w, "cpu"), newSlot(w, "cpu")}, time.Now().Add(defaultSlotExpiry))

	now := time.Now()
	require.Equal(t, 2, pool.freeCountAt(now))

	first := pool.take()
	require.NotNil(t, first)
	require.True(t, first.used)
	require.Equal(t, 1, pool.freeCountAt(now))

	second := pool.take()
	require.NotNil(t, second)
	require.Equal(t, 0, pool.freeCountAt(now))
	require.Nil(t, pool.take(), "empty freelist returns nil")

	pool.release(first)
	require.False(t, first.used)
	require.Equal(t, 1, pool.freeCountAt(now))

	// double release must not duplicate the freelist entry
	pool.release(first)
	require.Equal(t, 1, pool.freeCountAt(now))

	require.Same(t, first, pool.take())
}

func TestSlotPool_ResetRetainsUsedSlots(t *testing.T) {
	workerId := uuid.New()
	w := &worker{ListActiveWorkersResult: &v1.ListActiveWorkersResult{ID: workerId}}

	oldPool := &slotPool{worker: w, slotType: "cpu"}
	unacked := newSlot(w, "cpu")
	oldPool.reset([]*slot{unacked}, time.Now().Add(defaultSlotExpiry))
	require.Same(t, unacked, oldPool.take())

	// replenish carries the unacked slot into a rebuilt pool
	newPool := &slotPool{worker: w, slotType: "cpu"}
	fresh := newSlot(w, "cpu")
	newPool.reset([]*slot{fresh, unacked}, time.Now().Add(defaultSlotExpiry))

	now := time.Now()
	require.Equal(t, 1, newPool.freeCountAt(now), "used slot must stay off the freelist")
	require.Same(t, newPool, unacked.pool, "carried slot points at the new pool")

	// a nack after the rebuild releases into the new pool
	newPool.release(unacked)
	require.Equal(t, 2, newPool.freeCountAt(now))
}

func TestSlotPool_Staleness(t *testing.T) {
	workerId := uuid.New()
	w := &worker{ListActiveWorkersResult: &v1.ListActiveWorkersResult{ID: workerId}}

	pool := &slotPool{worker: w, slotType: "cpu"}
	pool.reset([]*slot{newSlot(w, "cpu")}, time.Now().Add(-time.Millisecond))

	now := time.Now()
	require.True(t, pool.staleAt(now))
	require.Equal(t, 0, pool.freeCountAt(now), "stale pools have no assignable capacity")

	var nilPool *slotPool
	require.True(t, nilPool.staleAt(now))
	require.Equal(t, 0, nilPool.freeCountAt(now))

	// zero-value expiry (never replenished) is stale
	empty := &slotPool{worker: w, slotType: "cpu"}
	require.True(t, empty.staleAt(now))
}

func TestRankWorkerIds(t *testing.T) {
	otherWorkerId := uuid.New()

	tests := []struct {
		name       string
		qi         *sqlcv1.V1QueueItem
		labels     []*sqlcv1.GetDesiredLabelsRow
		workers    []*v1.ListActiveWorkersResult
		candidates []uuid.UUID
		expected   []uuid.UUID
	}{
		{
			name: "HARD sticky strategy with desired worker available",
			qi: &sqlcv1.V1QueueItem{
				Sticky:          sqlcv1.V1StickyStrategyHARD,
				DesiredWorkerID: &stableWorkerId1,
			},
			candidates: []uuid.UUID{stableWorkerId1, otherWorkerId},
			expected:   []uuid.UUID{stableWorkerId1},
		},
		{
			name: "HARD sticky strategy without desired worker available",
			qi: &sqlcv1.V1QueueItem{
				Sticky:          sqlcv1.V1StickyStrategyHARD,
				DesiredWorkerID: &stableWorkerId1,
			},
			candidates: []uuid.UUID{stableWorkerId2, otherWorkerId},
			expected:   []uuid.UUID{},
		},
		{
			name: "SOFT sticky strategy prefers the desired worker",
			qi: &sqlcv1.V1QueueItem{
				Sticky:          sqlcv1.V1StickyStrategySOFT,
				DesiredWorkerID: &stableWorkerId1,
			},
			candidates: []uuid.UUID{stableWorkerId2, stableWorkerId1},
			expected:   []uuid.UUID{stableWorkerId1, stableWorkerId2},
		},
		{
			name: "affinity labels rank workers by weight",
			qi:   &sqlcv1.V1QueueItem{},
			labels: []*sqlcv1.GetDesiredLabelsRow{
				{
					Key:        "key1",
					Weight:     1,
					Comparator: sqlcv1.WorkerLabelComparatorGREATERTHAN,
					IntValue:   pgtype.Int4{Int32: 1, Valid: true},
				},
				{
					Key:        "key2",
					Weight:     1,
					Comparator: sqlcv1.WorkerLabelComparatorGREATERTHAN,
					IntValue:   pgtype.Int4{Int32: 1, Valid: true},
				},
			},
			workers: []*v1.ListActiveWorkersResult{
				{ID: stableWorkerId1, Labels: []*sqlcv1.ListManyWorkerLabelsRow{
					{Key: "key1", IntValue: pgtype.Int4{Int32: 2, Valid: true}},
				}},
				{ID: stableWorkerId2, Labels: []*sqlcv1.ListManyWorkerLabelsRow{
					{Key: "key1", IntValue: pgtype.Int4{Int32: 4, Valid: true}},
					{Key: "key2", IntValue: pgtype.Int4{Int32: 4, Valid: true}},
				}},
			},
			candidates: []uuid.UUID{stableWorkerId1, stableWorkerId2},
			expected:   []uuid.UUID{stableWorkerId2, stableWorkerId1},
		},
		{
			name: "required labels drop unsatisfiable workers",
			qi:   &sqlcv1.V1QueueItem{},
			labels: []*sqlcv1.GetDesiredLabelsRow{
				{
					Key:        "key1",
					Weight:     1,
					Required:   true,
					Comparator: sqlcv1.WorkerLabelComparatorEQUAL,
					IntValue:   pgtype.Int4{Int32: 1, Valid: true},
				},
			},
			workers: []*v1.ListActiveWorkersResult{
				{ID: stableWorkerId1, Labels: []*sqlcv1.ListManyWorkerLabelsRow{
					{Key: "key1", IntValue: pgtype.Int4{Int32: 1, Valid: true}},
				}},
				{ID: stableWorkerId2, Labels: []*sqlcv1.ListManyWorkerLabelsRow{
					{Key: "key1", IntValue: pgtype.Int4{Int32: 2, Valid: true}},
				}},
			},
			candidates: []uuid.UUID{stableWorkerId1, stableWorkerId2},
			expected:   []uuid.UUID{stableWorkerId1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestScheduler(t, uuid.New(), &mockAssignmentRepo{})
			s.setWorkers(tt.workers)

			var ranked []uuid.UUID
			onLoop(t, s, func() {
				ranked, _ = s.rankWorkerIds(tt.qi, tt.labels, tt.candidates)
			})

			assert.Equal(t, tt.expected, ranked)
		})
	}
}

func TestSelectSlotsForWorker(t *testing.T) {
	workerId := uuid.New()
	worker := &worker{ListActiveWorkersResult: &v1.ListActiveWorkersResult{ID: workerId}}

	expiry := time.Now().Add(defaultSlotExpiry)

	cpuPool := &slotPool{worker: worker, slotType: "cpu"}
	cpuPool.reset([]*slot{
		newSlot(worker, "cpu"),
		newSlot(worker, "cpu"),
		newSlot(worker, "cpu"),
	}, expiry)
	memPool := &slotPool{worker: worker, slotType: "mem"}
	memPool.reset([]*slot{newSlot(worker, "mem")}, expiry)
	poolsByType := map[string]*slotPool{"cpu": cpuPool, "mem": memPool}

	now := time.Now()

	selected, ok := selectSlotsFromPools(poolsByType, map[string]int32{"cpu": 2, "mem": 1}, now)
	assert.True(t, ok)
	assert.Len(t, selected, 3)

	_, ok = selectSlotsFromPools(poolsByType, map[string]int32{"cpu": 4}, now)
	assert.False(t, ok)
}
