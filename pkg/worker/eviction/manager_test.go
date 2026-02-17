package eviction

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func testLogger() *zerolog.Logger {
	l := zerolog.Nop()
	return &l
}

func TestManager_StartStop(t *testing.T) {
	m := NewManager(1000, DefaultManagerConfig(), testLogger())
	m.Start()
	// Starting again should be a no-op
	m.Start()
	m.Stop()
	// Stopping again should be safe
	m.Stop()
}

func TestManager_RegisterUnregister(t *testing.T) {
	m := NewManager(1000, DefaultManagerConfig(), testLogger())
	ctx, cancel := context.WithCancelCause(context.Background())

	m.RegisterRun("run-1", "step-1", ctx, cancel, nil)

	rec := m.Cache().Get("run-1")
	if rec == nil {
		t.Fatal("expected record")
	}
	if rec.StepRunId != "step-1" {
		t.Fatalf("expected step-1, got %s", rec.StepRunId)
	}

	m.UnregisterRun("run-1")
	if m.Cache().Get("run-1") != nil {
		t.Fatal("expected nil after unregister")
	}
}

func TestManager_MarkWaitingActive(t *testing.T) {
	m := NewManager(1000, DefaultManagerConfig(), testLogger())
	ctx, cancel := context.WithCancelCause(context.Background())

	m.RegisterRun("run-1", "step-1", ctx, cancel, nil)

	m.MarkWaiting("run-1", "durable_event", "step-1:signal-1")
	rec := m.Cache().Get("run-1")
	if !rec.IsWaiting() {
		t.Fatal("expected waiting")
	}

	m.MarkActive("run-1")
	rec = m.Cache().Get("run-1")
	if rec.IsWaiting() {
		t.Fatal("expected active")
	}
}

func TestManager_EvictsTTLExpiredRun(t *testing.T) {
	config := ManagerConfig{
		CheckInterval:              50 * time.Millisecond,
		ReserveSlots:               0,
		MinWaitForCapacityEviction: 10 * time.Second,
	}

	var mu sync.Mutex
	var evictedKeys []string
	var cancelledKeys []string

	m := NewManager(1000, config, testLogger(),
		WithOnEvictionSelected(func(key string, rec *DurableRunRecord) {
			mu.Lock()
			evictedKeys = append(evictedKeys, key)
			mu.Unlock()
		}),
		WithOnEvictionCancelled(func(key string, rec *DurableRunRecord) {
			mu.Lock()
			cancelledKeys = append(cancelledKeys, key)
			mu.Unlock()
		}),
	)

	ttl := 100 * time.Millisecond
	policy := &Policy{TTL: &ttl, AllowCapacityEviction: true, Priority: 0}

	ctx, cancel := context.WithCancelCause(context.Background())
	m.RegisterRun("run-1", "step-1", ctx, cancel, policy)

	// Mark waiting so it's eligible for TTL eviction
	m.Cache().MarkWaiting("run-1", time.Now().Add(-1*time.Second), "sleep", "signal-1")

	m.Start()
	defer m.Stop()

	// Wait for the eviction to happen
	deadline := time.After(2 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for eviction")
		default:
		}

		mu.Lock()
		evicted := len(evictedKeys) > 0
		cancelled := len(cancelledKeys) > 0
		mu.Unlock()

		if evicted && cancelled {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}

	// Verify context was cancelled with ErrEvicted
	if ctx.Err() == nil {
		t.Fatal("expected context to be cancelled")
	}
	if context.Cause(ctx) != ErrEvicted {
		t.Fatalf("expected ErrEvicted cause, got %v", context.Cause(ctx))
	}

	mu.Lock()
	defer mu.Unlock()
	if len(evictedKeys) != 1 || evictedKeys[0] != "run-1" {
		t.Fatalf("expected evictedKeys=['run-1'], got %v", evictedKeys)
	}
	if len(cancelledKeys) != 1 || cancelledKeys[0] != "run-1" {
		t.Fatalf("expected cancelledKeys=['run-1'], got %v", cancelledKeys)
	}
}

func TestManager_DoesNotEvictWithoutPolicy(t *testing.T) {
	config := ManagerConfig{
		CheckInterval:              50 * time.Millisecond,
		ReserveSlots:               0,
		MinWaitForCapacityEviction: 10 * time.Second,
	}

	m := NewManager(1000, config, testLogger())

	ctx, cancel := context.WithCancelCause(context.Background())
	m.RegisterRun("run-1", "step-1", ctx, cancel, nil) // nil policy

	m.Cache().MarkWaiting("run-1", time.Now().Add(-1*time.Hour), "sleep", "signal-1")

	m.Start()
	time.Sleep(200 * time.Millisecond)
	m.Stop()

	if ctx.Err() != nil {
		t.Fatal("expected context NOT to be cancelled (no eviction policy)")
	}
}

func TestManager_EvictsUnderCapacityPressure(t *testing.T) {
	config := ManagerConfig{
		CheckInterval:              50 * time.Millisecond,
		ReserveSlots:               0,
		MinWaitForCapacityEviction: 50 * time.Millisecond,
	}

	m := NewManager(1, config, testLogger()) // Only 1 durable slot

	// No TTL, but allow capacity eviction
	policy := &Policy{TTL: nil, AllowCapacityEviction: true, Priority: 0}

	ctx, cancel := context.WithCancelCause(context.Background())
	m.RegisterRun("run-1", "step-1", ctx, cancel, policy)

	// Mark waiting 200ms ago (exceeds min wait of 50ms)
	m.Cache().MarkWaiting("run-1", time.Now().Add(-200*time.Millisecond), "sleep", "signal-1")

	m.Start()
	defer m.Stop()

	deadline := time.After(2 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for capacity eviction")
		default:
		}
		if ctx.Err() != nil {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}

	if context.Cause(ctx) != ErrEvicted {
		t.Fatalf("expected ErrEvicted cause, got %v", context.Cause(ctx))
	}
}

func TestManager_CustomCache(t *testing.T) {
	customCache := NewInMemoryDurableEvictionCache()
	m := NewManager(1000, DefaultManagerConfig(), testLogger(), WithEvictionCache(customCache))

	if m.Cache() != customCache {
		t.Fatal("expected custom cache to be used")
	}
}
