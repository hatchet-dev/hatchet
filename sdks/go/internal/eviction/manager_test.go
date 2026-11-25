package eviction

import (
	"context"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
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

func TestEvictAllWaitingSnapshotsWaitState(t *testing.T) {
	var supplied *DurableRunRecord
	var liveKind, liveResource string
	mgr := makeTestManager(nil)
	mgr.requestEvict = func(ctx context.Context, key string, rec *DurableRunRecord) error {
		mgr.MarkActive(key)
		if live := mgr.cache.Get(key); live != nil {
			liveKind = live.WaitKind
			liveResource = live.WaitResourceID
		}
		supplied = rec
		return nil
	}

	key := "run-1/0"
	mgr.RegisterRun(key, "ext-1", 1, &EvictionPolicy{TTL: 30 * time.Second})
	mgr.MarkWaiting(key, "sleep", "resource-1")

	n := mgr.EvictAllWaiting(context.Background())
	require.Equal(t, 1, n)
	require.NotNil(t, supplied)
	assert.Equal(t, "sleep", supplied.WaitKind)
	assert.Equal(t, "resource-1", supplied.WaitResourceID)
	assert.Equal(t, "", liveKind)
	assert.Equal(t, "", liveResource)
	assert.Nil(t, mgr.cache.Get(key))
}

func TestTickSnapshotsWaitState(t *testing.T) {
	var supplied *DurableRunRecord
	var liveKind, liveResource string
	mgr := makeTestManager(nil)
	mgr.requestEvict = func(ctx context.Context, key string, rec *DurableRunRecord) error {
		mgr.MarkActive(key)
		if live := mgr.cache.Get(key); live != nil {
			liveKind = live.WaitKind
			liveResource = live.WaitResourceID
		}
		supplied = rec
		return nil
	}

	key := "run-1/0"
	mgr.RegisterRun(key, "ext-1", 1, &EvictionPolicy{TTL: 30 * time.Second})
	mgr.cache.MarkWaiting(key, time.Now().UTC().Add(-time.Minute), "sleep", "resource-1")

	require.NoError(t, mgr.tick(context.Background()))
	require.NotNil(t, supplied)
	assert.Equal(t, "sleep", supplied.WaitKind)
	assert.Equal(t, "resource-1", supplied.WaitResourceID)
	assert.Equal(t, "", liveKind)
	assert.Equal(t, "", liveResource)
	assert.Nil(t, mgr.cache.Get(key))
}

func TestEvictAllWaitingConcurrentWaitReturn(t *testing.T) {
	logger := zerolog.New(io.Discard).Level(zerolog.DebugLevel)
	mgr := makeTestManager(nil)
	mgr.l = &logger
	mgr.requestEvict = func(ctx context.Context, key string, rec *DurableRunRecord) error {
		_ = rec.WaitKind
		_ = rec.WaitResourceID
		_ = rec.StepRunID
		return nil
	}

	const workers = 32
	const rounds = 20

	var ready sync.WaitGroup
	ready.Add(1)

	var wg sync.WaitGroup
	wg.Add(workers + 1)
	for i := 0; i < workers; i++ {
		go func(i int) {
			defer wg.Done()
			ready.Wait()
			key := fmt.Sprintf("run-%d/0", i)
			for n := 0; n < rounds; n++ {
				mgr.RegisterRun(key, fmt.Sprintf("ext-%d", i), n, &EvictionPolicy{TTL: time.Second})
				mgr.MarkWaiting(key, "sleep", fmt.Sprintf("res-%d", i))
				mgr.MarkActive(key)
			}
		}(i)
	}
	go func() {
		defer wg.Done()
		ready.Wait()
		for n := 0; n < rounds; n++ {
			mgr.EvictAllWaiting(context.Background())
		}
	}()

	ready.Done()
	wg.Wait()
}
