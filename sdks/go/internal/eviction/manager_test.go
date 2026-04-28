package eviction

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeTestManager(cancel *mockCancelLocal) *DurableEvictionManager {
	if cancel == nil {
		cancel = &mockCancelLocal{}
	}

	return &DurableEvictionManager{
		durableSlots: 10,
		cancelLocal:  cancel.call,
		requestEvict: func(ctx context.Context, key string, rec *DurableRunRecord) error {
			return nil
		},
		config: DurableEvictionConfig{CheckInterval: 1 * time.Hour},
		cache:  NewDurableEvictionCache(),
	}
}

type mockCancelLocal struct {
	calls []string
}

func (m *mockCancelLocal) call(key string) {
	m.calls = append(m.calls, key)
}

func TestHandleServerEvictionCancelsAndUnregisters(t *testing.T) {
	cancel := &mockCancelLocal{}
	mgr := makeTestManager(cancel)

	key := "run-1/0"
	mgr.RegisterRun(key, "ext-1", 2, &EvictionPolicy{TTL: 30 * time.Second})
	mgr.MarkWaiting(key, "sleep", "s1")

	mgr.HandleServerEviction("ext-1", 2)

	require.Len(t, cancel.calls, 1)
	assert.Equal(t, key, cancel.calls[0])
	assert.Nil(t, mgr.cache.Get(key))
}

func TestHandleServerEvictionUnknownIDIsNoop(t *testing.T) {
	cancel := &mockCancelLocal{}
	mgr := makeTestManager(cancel)

	mgr.RegisterRun("run-1/0", "ext-1", 1, nil)

	mgr.HandleServerEviction("no-such-id", 1)

	assert.Empty(t, cancel.calls)
	assert.NotNil(t, mgr.cache.Get("run-1/0"))
}

func TestHandleServerEvictionOnlyEvictsMatchingRun(t *testing.T) {
	cancel := &mockCancelLocal{}
	mgr := makeTestManager(cancel)

	mgr.RegisterRun("run-1/0", "ext-1", 1, &EvictionPolicy{TTL: 30 * time.Second})
	mgr.RegisterRun("run-2/0", "ext-2", 1, &EvictionPolicy{TTL: 30 * time.Second})
	mgr.MarkWaiting("run-1/0", "sleep", "s1")
	mgr.MarkWaiting("run-2/0", "sleep", "s2")

	mgr.HandleServerEviction("ext-1", 1)

	require.Len(t, cancel.calls, 1)
	assert.Equal(t, "run-1/0", cancel.calls[0])
	assert.Nil(t, mgr.cache.Get("run-1/0"))
	assert.NotNil(t, mgr.cache.Get("run-2/0"))
}

func TestHandleServerEvictionSkipsNewerInvocation(t *testing.T) {
	cancel := &mockCancelLocal{}
	mgr := makeTestManager(cancel)

	mgr.RegisterRun("run-1/0", "ext-1", 3, &EvictionPolicy{TTL: 30 * time.Second})
	mgr.MarkWaiting("run-1/0", "sleep", "s1")

	mgr.HandleServerEviction("ext-1", 2)

	assert.Empty(t, cancel.calls)
	assert.NotNil(t, mgr.cache.Get("run-1/0"))
}

func TestHandleServerEvictionEvictsExactInvocationMatch(t *testing.T) {
	cancel := &mockCancelLocal{}
	mgr := makeTestManager(cancel)

	mgr.RegisterRun("run-1/0", "ext-1", 5, &EvictionPolicy{TTL: 30 * time.Second})
	mgr.MarkWaiting("run-1/0", "sleep", "s1")

	mgr.HandleServerEviction("ext-1", 5)

	require.Len(t, cancel.calls, 1)
	assert.Equal(t, "run-1/0", cancel.calls[0])
	assert.Nil(t, mgr.cache.Get("run-1/0"))
}
