package eviction

import (
	"context"
	"testing"
	"time"
)

func newTestRecord(key string, evictionPolicy *Policy) (context.Context, context.CancelCauseFunc) {
	return context.WithCancelCause(context.Background())
}

func TestInMemoryCache_RegisterAndGet(t *testing.T) {
	cache := NewInMemoryDurableEvictionCache()
	ctx, cancel := newTestRecord("run-1", nil)

	cache.RegisterRun("run-1", "step-1", ctx, cancel, time.Now().UTC(), nil)

	rec := cache.Get("run-1")
	if rec == nil {
		t.Fatal("expected record, got nil")
	}
	if rec.StepRunId != "step-1" {
		t.Fatalf("expected step-1, got %s", rec.StepRunId)
	}
	if rec.IsWaiting() {
		t.Fatal("expected not waiting")
	}
	if rec.IsCancelled() {
		t.Fatal("expected not cancelled")
	}
}

func TestInMemoryCache_UnregisterRun(t *testing.T) {
	cache := NewInMemoryDurableEvictionCache()
	ctx, cancel := newTestRecord("run-1", nil)
	cache.RegisterRun("run-1", "step-1", ctx, cancel, time.Now().UTC(), nil)

	cache.UnregisterRun("run-1")

	if rec := cache.Get("run-1"); rec != nil {
		t.Fatal("expected nil after unregister")
	}
}

func TestInMemoryCache_UnregisterNonexistent(t *testing.T) {
	cache := NewInMemoryDurableEvictionCache()
	// Should not panic
	cache.UnregisterRun("nonexistent")
}

func TestInMemoryCache_MarkWaitingAndActive(t *testing.T) {
	cache := NewInMemoryDurableEvictionCache()
	ctx, cancel := newTestRecord("run-1", nil)
	cache.RegisterRun("run-1", "step-1", ctx, cancel, time.Now().UTC(), nil)

	now := time.Now().UTC()
	cache.MarkWaiting("run-1", now, "sleep", "signal-1")

	rec := cache.Get("run-1")
	if !rec.IsWaiting() {
		t.Fatal("expected waiting")
	}
	if rec.WaitKind != "sleep" {
		t.Fatalf("expected wait kind 'sleep', got '%s'", rec.WaitKind)
	}
	if rec.WaitResourceID != "signal-1" {
		t.Fatalf("expected resource id 'signal-1', got '%s'", rec.WaitResourceID)
	}

	cache.MarkActive("run-1")

	rec = cache.Get("run-1")
	if rec.IsWaiting() {
		t.Fatal("expected not waiting after mark active")
	}
	if rec.WaitKind != "" {
		t.Fatalf("expected empty wait kind, got '%s'", rec.WaitKind)
	}
}

func TestInMemoryCache_MarkWaitingSkipsCancelled(t *testing.T) {
	cache := NewInMemoryDurableEvictionCache()
	ctx, cancel := newTestRecord("run-1", nil)
	cache.RegisterRun("run-1", "step-1", ctx, cancel, time.Now().UTC(), nil)

	// Cancel the context
	cancel(nil)

	cache.MarkWaiting("run-1", time.Now().UTC(), "sleep", "signal-1")

	rec := cache.Get("run-1")
	if rec.IsWaiting() {
		t.Fatal("expected not waiting after context cancelled")
	}
}

func TestInMemoryCache_MarkWaitingNonexistent(t *testing.T) {
	cache := NewInMemoryDurableEvictionCache()
	// Should not panic
	cache.MarkWaiting("nonexistent", time.Now().UTC(), "sleep", "signal-1")
}

func TestInMemoryCache_MarkActiveNonexistent(t *testing.T) {
	cache := NewInMemoryDurableEvictionCache()
	// Should not panic
	cache.MarkActive("nonexistent")
}

func TestInMemoryCache_SelectEvictionCandidate_NoWaiting(t *testing.T) {
	cache := NewInMemoryDurableEvictionCache()
	key := cache.SelectEvictionCandidate(time.Now().UTC(), 1000, 0, 10*time.Second)
	if key != "" {
		t.Fatalf("expected empty key, got %s", key)
	}
}

func TestInMemoryCache_SelectEvictionCandidate_NoEvictionPolicy(t *testing.T) {
	cache := NewInMemoryDurableEvictionCache()
	ctx, cancel := newTestRecord("run-1", nil)
	cache.RegisterRun("run-1", "step-1", ctx, cancel, time.Now().UTC(), nil) // nil policy
	cache.MarkWaiting("run-1", time.Now().Add(-20*time.Minute), "sleep", "signal-1")

	key := cache.SelectEvictionCandidate(time.Now().UTC(), 1000, 0, 10*time.Second)
	if key != "" {
		t.Fatalf("expected empty key for nil eviction policy, got %s", key)
	}
}

func TestInMemoryCache_SelectEvictionCandidate_TTLEviction(t *testing.T) {
	cache := NewInMemoryDurableEvictionCache()
	ttl := 5 * time.Minute
	policy := &Policy{TTL: &ttl, AllowCapacityEviction: true, Priority: 0}

	ctx, cancel := newTestRecord("run-1", policy)
	cache.RegisterRun("run-1", "step-1", ctx, cancel, time.Now().UTC(), policy)

	// Mark waiting 10 minutes ago (exceeds 5-minute TTL)
	cache.MarkWaiting("run-1", time.Now().Add(-10*time.Minute), "sleep", "signal-1")

	key := cache.SelectEvictionCandidate(time.Now().UTC(), 1000, 0, 10*time.Second)
	if key != "run-1" {
		t.Fatalf("expected 'run-1', got '%s'", key)
	}
}

func TestInMemoryCache_SelectEvictionCandidate_TTLNotYetExpired(t *testing.T) {
	cache := NewInMemoryDurableEvictionCache()
	ttl := 15 * time.Minute
	policy := &Policy{TTL: &ttl, AllowCapacityEviction: false, Priority: 0}

	ctx, cancel := newTestRecord("run-1", policy)
	cache.RegisterRun("run-1", "step-1", ctx, cancel, time.Now().UTC(), policy)

	// Mark waiting 5 minutes ago (less than 15-minute TTL)
	cache.MarkWaiting("run-1", time.Now().Add(-5*time.Minute), "sleep", "signal-1")

	key := cache.SelectEvictionCandidate(time.Now().UTC(), 1000, 0, 10*time.Second)
	if key != "" {
		t.Fatalf("expected empty key (TTL not expired), got '%s'", key)
	}
}

func TestInMemoryCache_SelectEvictionCandidate_TTLPriorityOrdering(t *testing.T) {
	cache := NewInMemoryDurableEvictionCache()
	ttl := 1 * time.Minute

	// run-1: priority 5 (higher = evicted later)
	policy1 := &Policy{TTL: &ttl, Priority: 5}
	ctx1, cancel1 := newTestRecord("run-1", policy1)
	cache.RegisterRun("run-1", "step-1", ctx1, cancel1, time.Now().UTC(), policy1)
	cache.MarkWaiting("run-1", time.Now().Add(-10*time.Minute), "sleep", "s1")

	// run-2: priority 1 (lower = evicted first)
	policy2 := &Policy{TTL: &ttl, Priority: 1}
	ctx2, cancel2 := newTestRecord("run-2", policy2)
	cache.RegisterRun("run-2", "step-2", ctx2, cancel2, time.Now().UTC(), policy2)
	cache.MarkWaiting("run-2", time.Now().Add(-10*time.Minute), "sleep", "s2")

	key := cache.SelectEvictionCandidate(time.Now().UTC(), 1000, 0, 10*time.Second)
	if key != "run-2" {
		t.Fatalf("expected 'run-2' (lower priority), got '%s'", key)
	}
}

func TestInMemoryCache_SelectEvictionCandidate_CapacityEviction(t *testing.T) {
	cache := NewInMemoryDurableEvictionCache()
	// No TTL, but allow capacity eviction
	policy := &Policy{TTL: nil, AllowCapacityEviction: true, Priority: 0}

	ctx, cancel := newTestRecord("run-1", policy)
	cache.RegisterRun("run-1", "step-1", ctx, cancel, time.Now().UTC(), policy)
	cache.MarkWaiting("run-1", time.Now().Add(-20*time.Second), "sleep", "signal-1")

	// durableSlots=1, reserveSlots=0, 1 waiting >= (1-0)=1 -> capacity pressure
	key := cache.SelectEvictionCandidate(time.Now().UTC(), 1, 0, 10*time.Second)
	if key != "run-1" {
		t.Fatalf("expected 'run-1' under capacity pressure, got '%s'", key)
	}
}

func TestInMemoryCache_SelectEvictionCandidate_CapacityNoPresure(t *testing.T) {
	cache := NewInMemoryDurableEvictionCache()
	policy := &Policy{TTL: nil, AllowCapacityEviction: true, Priority: 0}

	ctx, cancel := newTestRecord("run-1", policy)
	cache.RegisterRun("run-1", "step-1", ctx, cancel, time.Now().UTC(), policy)
	cache.MarkWaiting("run-1", time.Now().Add(-20*time.Second), "sleep", "signal-1")

	// durableSlots=1000, reserveSlots=0, 1 waiting < (1000-0)=1000 -> no pressure
	key := cache.SelectEvictionCandidate(time.Now().UTC(), 1000, 0, 10*time.Second)
	if key != "" {
		t.Fatalf("expected empty key (no capacity pressure), got '%s'", key)
	}
}

func TestInMemoryCache_SelectEvictionCandidate_CapacityMinWait(t *testing.T) {
	cache := NewInMemoryDurableEvictionCache()
	policy := &Policy{TTL: nil, AllowCapacityEviction: true, Priority: 0}

	ctx, cancel := newTestRecord("run-1", policy)
	cache.RegisterRun("run-1", "step-1", ctx, cancel, time.Now().UTC(), policy)
	// Only been waiting 2 seconds, min is 10 seconds
	cache.MarkWaiting("run-1", time.Now().Add(-2*time.Second), "sleep", "signal-1")

	// Under capacity pressure but hasn't waited long enough
	key := cache.SelectEvictionCandidate(time.Now().UTC(), 1, 0, 10*time.Second)
	if key != "" {
		t.Fatalf("expected empty key (min wait not met), got '%s'", key)
	}
}

func TestInMemoryCache_SelectEvictionCandidate_CapacityNotAllowed(t *testing.T) {
	cache := NewInMemoryDurableEvictionCache()
	policy := &Policy{TTL: nil, AllowCapacityEviction: false, Priority: 0}

	ctx, cancel := newTestRecord("run-1", policy)
	cache.RegisterRun("run-1", "step-1", ctx, cancel, time.Now().UTC(), policy)
	cache.MarkWaiting("run-1", time.Now().Add(-20*time.Second), "sleep", "signal-1")

	// Under pressure but AllowCapacityEviction is false
	key := cache.SelectEvictionCandidate(time.Now().UTC(), 1, 0, 10*time.Second)
	if key != "" {
		t.Fatalf("expected empty key (capacity eviction not allowed), got '%s'", key)
	}
}

func TestInMemoryCache_SelectEvictionCandidate_SkipsCancelled(t *testing.T) {
	cache := NewInMemoryDurableEvictionCache()
	ttl := 1 * time.Minute
	policy := &Policy{TTL: &ttl, AllowCapacityEviction: true, Priority: 0}

	ctx, cancel := newTestRecord("run-1", policy)
	cache.RegisterRun("run-1", "step-1", ctx, cancel, time.Now().UTC(), policy)
	cache.MarkWaiting("run-1", time.Now().Add(-10*time.Minute), "sleep", "signal-1")

	// Cancel the context before selection
	cancel(nil)

	key := cache.SelectEvictionCandidate(time.Now().UTC(), 1000, 0, 10*time.Second)
	if key != "" {
		t.Fatalf("expected empty key (cancelled run), got '%s'", key)
	}
}

func TestDurableRunRecord_IsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	rec := &DurableRunRecord{Ctx: ctx, Cancel: cancel}

	if rec.IsCancelled() {
		t.Fatal("expected not cancelled")
	}

	cancel(ErrEvicted)

	if !rec.IsCancelled() {
		t.Fatal("expected cancelled")
	}
}
