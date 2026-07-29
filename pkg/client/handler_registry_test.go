package client

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlerRegistryStaleRemoveIsNoOp(t *testing.T) {
	reg := newHandlerRegistry[string, int]()

	remove := reg.store("key", "session-a", func(int) error { return nil }, nil)
	reg.removeSession("key", "session-a")

	called := false
	reg.store("key", "session-b", func(int) error {
		called = true
		return nil
	}, nil)

	remove()
	regs := reg.snapshot("key")
	require.Len(t, regs, 1)
	require.NoError(t, regs[0].handle(1))
	assert.True(t, called)
}

func TestHandlerRegistryStaleRemoveSameSessionIsNoOp(t *testing.T) {
	reg := newHandlerRegistry[string, int]()

	remove := reg.store("key", "session", func(int) error { return nil }, nil)
	reg.removeSession("key", "session")

	called := false
	reg.store("key", "session", func(int) error {
		called = true
		return nil
	}, nil)

	remove()
	regs := reg.snapshot("key")
	require.Len(t, regs, 1)
	require.NoError(t, regs[0].handle(1))
	assert.True(t, called)
}

func TestHandlerRegistryFailAllNotifiesAndEmptiesRegistry(t *testing.T) {
	reg := newHandlerRegistry[string, int]()

	var notified atomic.Int32
	reg.store("k1", "s1", func(int) error { return nil }, func(error) { notified.Add(1) })
	reg.store("k2", "s2", func(int) error { return nil }, func(error) { notified.Add(1) })
	reg.store("k3", "s3", func(int) error { return nil }, nil)

	err := errors.New("boom")
	count := reg.failAll(err)

	assert.Equal(t, 3, count)
	assert.Equal(t, int32(2), notified.Load())
	assert.False(t, reg.hasAny())
}

func TestHandlerRegistryRemoveRegistrationsSparesConcurrentStores(t *testing.T) {
	reg := newHandlerRegistry[string, int]()

	regs := reg.snapshot("key")
	require.Empty(t, regs)

	removeFirst := reg.store("key", "first", func(int) error { return nil }, nil)
	firstSnapshot := reg.snapshot("key")
	require.Len(t, firstSnapshot, 1)

	reg.store("key", "second", func(int) error { return nil }, nil)

	reg.removeRegistrations("key", firstSnapshot)
	removeFirst()

	second := reg.snapshot("key")
	require.Len(t, second, 1)
}

func TestHandlerRegistryAutoSessionsAreUnique(t *testing.T) {
	reg := newHandlerRegistry[string, int]()

	reg.store("key", "", func(int) error { return nil }, nil)
	reg.store("key", "", func(int) error { return nil }, nil)

	regs := reg.snapshot("key")
	assert.Len(t, regs, 2)
}

func TestHandlerRegistryConcurrentStoreAndSnapshotRace(t *testing.T) {
	reg := newHandlerRegistry[string, int]()

	const iterations = 500
	var wg sync.WaitGroup

	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := "key"
			remove := reg.store(key, "", func(int) error { return nil }, nil)
			snap := reg.snapshot(key)
			reg.removeRegistrations(key, snap)
			remove()
		}(i)
	}

	wg.Wait()
}
