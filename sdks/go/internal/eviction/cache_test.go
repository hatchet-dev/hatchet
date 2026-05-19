package eviction

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var baseTime = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func dt(seconds int) time.Time {
	return baseTime.Add(time.Duration(seconds) * time.Second)
}

func TestTTLEvictionPrefersOldestWaitingAndPriority(t *testing.T) {
	cache := NewDurableEvictionCache()

	key1 := "run-1/0"
	key2 := "run-2/0"

	highPrio := &EvictionPolicy{TTL: 10 * time.Second, Priority: 10}
	lowPrio := &EvictionPolicy{TTL: 10 * time.Second, Priority: 0}

	cache.RegisterRun(key1, "run-1", 1, dt(0), highPrio)
	cache.RegisterRun(key2, "run-2", 1, dt(0), lowPrio)

	cache.MarkWaiting(key1, dt(0), "workflow_run_result", "wf1")
	cache.MarkWaiting(key2, dt(5), "workflow_run_result", "wf2")

	chosen := cache.SelectEvictionCandidate(dt(20), 100, 0, 0)
	assert.Equal(t, key2, chosen)
}

func TestNoneEvictionParamsNeverSelected(t *testing.T) {
	cache := NewDurableEvictionCache()

	keyNo := "run-no/0"
	keyYes := "run-yes/0"

	cache.RegisterRun(keyNo, "run-no", 1, dt(0), nil)
	cache.RegisterRun(keyYes, "run-yes", 1, dt(0), &EvictionPolicy{TTL: 1 * time.Second})

	cache.MarkWaiting(keyNo, dt(0), "durable_event", "x")
	cache.MarkWaiting(keyYes, dt(0), "durable_event", "y")

	chosen := cache.SelectEvictionCandidate(dt(10), 100, 0, 0)
	assert.Equal(t, keyYes, chosen)
}

func TestCapacityEvictionRespectsAllowCapacityAndMinWait(t *testing.T) {
	cache := NewDurableEvictionCache()

	keyBlocked := "run-blocked/0"
	keyOK := "run-ok/0"

	cache.RegisterRun(keyBlocked, "run-blocked", 1, dt(0), &EvictionPolicy{
		TTL:                   1 * time.Hour,
		AllowCapacityEviction: false,
		Priority:              0,
	})
	cache.RegisterRun(keyOK, "run-ok", 1, dt(0), &EvictionPolicy{
		TTL:                   1 * time.Hour,
		AllowCapacityEviction: true,
		Priority:              0,
	})

	cache.MarkWaiting(keyBlocked, dt(0), "durable_event", "x")
	cache.MarkWaiting(keyOK, dt(0), "durable_event", "y")

	// Capacity pressure (waiting_count==durable_slots==2), but min_wait not met
	chosenTooSoon := cache.SelectEvictionCandidate(dt(5), 2, 0, 10*time.Second)
	assert.Equal(t, "", chosenTooSoon)

	// Now past min wait: only keyOK is eligible
	chosen := cache.SelectEvictionCandidate(dt(15), 2, 0, 10*time.Second)
	assert.Equal(t, keyOK, chosen)
}

func TestConcurrentWaitsKeepWaitingUntilAllResolved(t *testing.T) {
	cache := NewDurableEvictionCache()
	key := "run-bulk/0"
	policy := &EvictionPolicy{TTL: 5 * time.Second, Priority: 0}

	cache.RegisterRun(key, "run-bulk", 1, dt(0), policy)

	cache.MarkWaiting(key, dt(1), "spawn_child", "child0")
	cache.MarkWaiting(key, dt(1), "spawn_child", "child1")
	cache.MarkWaiting(key, dt(1), "spawn_child", "child2")

	rec := cache.Get(key)
	require.NotNil(t, rec)
	assert.True(t, rec.IsWaiting())
	assert.Equal(t, 3, rec.WaitCount())

	// child0 completes -- run should still be waiting
	cache.MarkActive(key, dt(2))
	assert.True(t, rec.IsWaiting())
	assert.Equal(t, 2, rec.WaitCount())
	assert.NotNil(t, rec.GetWaitingSince())
	assert.Equal(t, dt(1), *rec.GetWaitingSince())

	// TTL still fires while 2 children pending
	chosen := cache.SelectEvictionCandidate(dt(10), 100, 0, 0)
	assert.Equal(t, key, chosen)

	// child1 completes
	cache.MarkActive(key, dt(11))
	assert.True(t, rec.IsWaiting())
	assert.Equal(t, 1, rec.WaitCount())

	// child2 completes -- now truly active
	cache.MarkActive(key, dt(12))
	assert.False(t, rec.IsWaiting())
	assert.Equal(t, 0, rec.WaitCount())
	assert.Nil(t, rec.GetWaitingSince())
}

func TestFindKeyByStepRunIDReturnsMatchingKey(t *testing.T) {
	cache := NewDurableEvictionCache()
	cache.RegisterRun("run-a/0", "ext-a", 1, dt(0), nil)
	cache.RegisterRun("run-b/0", "ext-b", 1, dt(0), nil)

	key := cache.FindKeyByStepRunID("ext-a")
	require.NotNil(t, key)
	assert.Equal(t, "run-a/0", *key)

	key = cache.FindKeyByStepRunID("ext-b")
	require.NotNil(t, key)
	assert.Equal(t, "run-b/0", *key)
}

func TestFindKeyByStepRunIDReturnsNilForUnknown(t *testing.T) {
	cache := NewDurableEvictionCache()
	cache.RegisterRun("run-a/0", "ext-a", 1, dt(0), nil)

	key := cache.FindKeyByStepRunID("no-such-id")
	assert.Nil(t, key)
}

func TestFindKeyByStepRunIDReturnsNilAfterUnregister(t *testing.T) {
	cache := NewDurableEvictionCache()
	cache.RegisterRun("run-a/0", "ext-a", 1, dt(0), nil)

	key := cache.FindKeyByStepRunID("ext-a")
	require.NotNil(t, key)
	assert.Equal(t, "run-a/0", *key)

	cache.UnregisterRun("run-a/0")
	key = cache.FindKeyByStepRunID("ext-a")
	assert.Nil(t, key)
}

func TestMarkActiveFloorsAtZero(t *testing.T) {
	cache := NewDurableEvictionCache()
	key := "run-extra/0"
	policy := &EvictionPolicy{TTL: 5 * time.Second, Priority: 0}

	cache.RegisterRun(key, "run-extra", 1, dt(0), policy)
	cache.MarkWaiting(key, dt(0), "sleep", "s")

	cache.MarkActive(key, dt(1))
	cache.MarkActive(key, dt(2)) // extra call

	rec := cache.Get(key)
	require.NotNil(t, rec)
	assert.Equal(t, 0, rec.WaitCount())
	assert.False(t, rec.IsWaiting())
}
