package analytics

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

var nopLogger = zerolog.Nop()

type flushedEvent struct {
	Resource   Resource
	Action     Action
	TenantID   uuid.UUID
	TokenID    *uuid.UUID
	Count      int64
	Properties Properties
}

type flushRecorder struct {
	mu     sync.Mutex
	events []flushedEvent
}

func (r *flushRecorder) record(resource Resource, action Action, tenantID uuid.UUID, tokenID *uuid.UUID, count int64, properties Properties) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, flushedEvent{
		Resource:   resource,
		Action:     action,
		TenantID:   tenantID,
		TokenID:    tokenID,
		Count:      count,
		Properties: properties,
	})
}

func (r *flushRecorder) getEvents() []flushedEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := make([]flushedEvent, len(r.events))
	copy(cp, r.events)
	return cp
}

func TestCount_SingleTenant(t *testing.T) {
	rec := &flushRecorder{}
	agg := NewAggregator(&nopLogger, true, 50*time.Millisecond, 0, rec.record)
	agg.Start()
	defer agg.Shutdown()

	tenantID := uuid.New()
	for i := 0; i < 100; i++ {
		agg.Count(Event, Create, tenantID, nil, 1)
	}

	time.Sleep(120 * time.Millisecond)

	events := rec.getEvents()
	if len(events) == 0 {
		t.Fatal("expected at least one flushed event")
	}

	var total int64
	for _, e := range events {
		if e.Resource != Event || e.Action != Create {
			t.Errorf("unexpected resource:action = %s:%s", e.Resource, e.Action)
		}
		if e.TenantID != tenantID {
			t.Errorf("unexpected tenant ID")
		}
		total += e.Count
	}

	if total != 100 {
		t.Errorf("expected total count 100, got %d", total)
	}
}

func TestCount_BatchSize(t *testing.T) {
	rec := &flushRecorder{}
	agg := NewAggregator(&nopLogger, true, 50*time.Millisecond, 0, rec.record)
	agg.Start()
	defer agg.Shutdown()

	tenantID := uuid.New()
	agg.Count(Event, Create, tenantID, nil, 500)
	agg.Count(Event, Create, tenantID, nil, 300)

	time.Sleep(120 * time.Millisecond)

	events := rec.getEvents()
	var total int64
	for _, e := range events {
		total += e.Count
	}

	if total != 800 {
		t.Errorf("expected total count 800, got %d", total)
	}
}

func TestCount_MultipleTenants(t *testing.T) {
	rec := &flushRecorder{}
	agg := NewAggregator(&nopLogger, true, 50*time.Millisecond, 0, rec.record)
	agg.Start()
	defer agg.Shutdown()

	tenant1 := uuid.New()
	tenant2 := uuid.New()

	agg.Count(Event, Create, tenant1, nil, 10)
	agg.Count(Event, Create, tenant2, nil, 20)
	agg.Count(WorkflowRun, Create, tenant1, nil, 5)

	time.Sleep(120 * time.Millisecond)

	events := rec.getEvents()
	if len(events) != 3 {
		t.Fatalf("expected 3 flushed events, got %d", len(events))
	}

	counts := make(map[string]int64)
	for _, e := range events {
		key := string(e.Resource) + ":" + string(e.Action) + ":" + e.TenantID.String()
		counts[key] = e.Count
	}

	if counts[string(Event)+":"+string(Create)+":"+tenant1.String()] != 10 {
		t.Error("tenant1 event:create count mismatch")
	}
	if counts[string(Event)+":"+string(Create)+":"+tenant2.String()] != 20 {
		t.Error("tenant2 event:create count mismatch")
	}
	if counts[string(WorkflowRun)+":"+string(Create)+":"+tenant1.String()] != 5 {
		t.Error("tenant1 workflow-run:create count mismatch")
	}
}

func TestFlush_EvictsIdleKeys(t *testing.T) {
	rec := &flushRecorder{}
	agg := NewAggregator(&nopLogger, true, 50*time.Millisecond, 0, rec.record)
	agg.Start()
	defer agg.Shutdown()

	tenantID := uuid.New()
	agg.Count(Event, Create, tenantID, nil, 1)

	// Wait for first flush (flushes count=1)
	time.Sleep(80 * time.Millisecond)

	// Wait for second flush (should evict the zero-count key)
	time.Sleep(80 * time.Millisecond)

	keyCount := 0
	agg.counters.Range(func(_, _ any) bool {
		keyCount++
		return true
	})

	if keyCount != 0 {
		t.Errorf("expected 0 keys after eviction, got %d", keyCount)
	}
}

func TestShutdown_FinalFlush(t *testing.T) {
	rec := &flushRecorder{}
	agg := NewAggregator(&nopLogger, true, 10*time.Second, 0, rec.record) // long interval
	agg.Start()

	tenantID := uuid.New()
	agg.Count(Event, Create, tenantID, nil, 42)

	agg.Shutdown()

	events := rec.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event after shutdown flush, got %d", len(events))
	}
	if events[0].Count != 42 {
		t.Errorf("expected count 42, got %d", events[0].Count)
	}
}

func TestCount_ConcurrentAccess(t *testing.T) {
	rec := &flushRecorder{}
	agg := NewAggregator(&nopLogger, true, 200*time.Millisecond, 0, rec.record)
	agg.Start()
	defer agg.Shutdown()

	tenantID := uuid.New()
	var wg sync.WaitGroup
	numGoroutines := 100
	countsPerGoroutine := 1000

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < countsPerGoroutine; j++ {
				agg.Count(Event, Create, tenantID, nil, 1)
			}
		}()
	}
	wg.Wait()

	time.Sleep(300 * time.Millisecond)

	events := rec.getEvents()
	var total int64
	for _, e := range events {
		total += e.Count
	}

	expected := int64(numGoroutines * countsPerGoroutine)
	if total != expected {
		t.Errorf("expected total count %d, got %d", expected, total)
	}
}

func TestCount_NoLossUnderContention(t *testing.T) {
	rec := &flushRecorder{}
	agg := NewAggregator(&nopLogger, true, 20*time.Millisecond, 0, rec.record)
	agg.Start()

	tenantID := uuid.New()
	var written atomic.Int64
	var wg sync.WaitGroup

	wg.Add(50)
	for i := 0; i < 50; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				agg.Count(Event, Create, tenantID, nil, 1)
				written.Add(1)
				time.Sleep(50 * time.Microsecond)
			}
		}()
	}
	wg.Wait()
	agg.Shutdown()

	events := rec.getEvents()
	var total int64
	for _, e := range events {
		total += e.Count
	}

	if total != written.Load() {
		t.Errorf("data loss detected: wrote %d, flushed %d", written.Load(), total)
	}
}

func TestCount_WithFeatureFlags(t *testing.T) {
	rec := &flushRecorder{}
	agg := NewAggregator(&nopLogger, true, 50*time.Millisecond, 0, rec.record)
	agg.Start()
	defer agg.Shutdown()

	tenantID := uuid.New()

	flagA := Properties{"has_priority": true}
	flagAB := Properties{"has_priority": true, "has_scope": true}

	agg.Count(Event, Create, tenantID, nil, 3, flagA)
	agg.Count(Event, Create, tenantID, nil, 7, flagAB)
	agg.Count(Event, Create, tenantID, nil, 2, flagA)
	agg.Count(Event, Create, tenantID, nil, 5) // no flags

	time.Sleep(120 * time.Millisecond)

	events := rec.getEvents()
	if len(events) != 3 {
		t.Fatalf("expected 3 flushed buckets, got %d", len(events))
	}

	buckets := make(map[string]int64)
	for _, e := range events {
		_, hasP := e.Properties["has_priority"]
		_, hasS := e.Properties["has_scope"]
		label := fmt.Sprintf("p=%v,s=%v", hasP, hasS)
		buckets[label] = e.Count
	}

	if buckets["p=true,s=false"] != 5 {
		t.Errorf("expected has_priority-only count 5, got %d", buckets["p=true,s=false"])
	}
	if buckets["p=true,s=true"] != 7 {
		t.Errorf("expected has_priority+has_scope count 7, got %d", buckets["p=true,s=true"])
	}
	if buckets["p=false,s=false"] != 5 {
		t.Errorf("expected no-flags count 5, got %d", buckets["p=false,s=false"])
	}
}

func TestCount_FlagsPassedToFlush(t *testing.T) {
	rec := &flushRecorder{}
	agg := NewAggregator(&nopLogger, true, 10*time.Second, 0, rec.record)
	agg.Start()

	tenantID := uuid.New()
	props := Properties{"has_priority": true, "has_scope": true}
	agg.Count(Event, Create, tenantID, nil, 10, props)

	agg.Shutdown()

	events := rec.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	e := events[0]
	if e.Properties["has_priority"] != true {
		t.Errorf("expected has_priority=true, got %v", e.Properties["has_priority"])
	}
	if e.Properties["has_scope"] != true {
		t.Errorf("expected has_scope=true, got %v", e.Properties["has_scope"])
	}
}

func TestProps(t *testing.T) {
	got := Props(
		"worker_name", "my-worker",
		"runtime_language", "go",
		"has_labels", true,
		"has_webhook_id", false,
	)
	if got["worker_name"] != "my-worker" {
		t.Errorf("expected worker_name='my-worker', got %v", got["worker_name"])
	}
	if got["runtime_language"] != "go" {
		t.Errorf("expected runtime_language='go', got %v", got["runtime_language"])
	}
	if got["has_labels"] != true {
		t.Error("expected has_labels=true")
	}
	if got["has_webhook_id"] != false {
		t.Errorf("expected has_webhook_id=false, got %v", got["has_webhook_id"])
	}

	nilResult := Props()
	if nilResult != nil {
		t.Errorf("expected nil for empty call, got %v", nilResult)
	}

	oddResult := Props("a")
	if oddResult != nil {
		t.Errorf("expected nil for odd-length call, got %v", oddResult)
	}
}

func TestProps_NilValuesOmitted(t *testing.T) {
	got := Props(
		"present", "value",
		"absent", nil,
	)
	if got["present"] != "value" {
		t.Errorf("expected present='value', got %v", got["present"])
	}
	if _, ok := got["absent"]; ok {
		t.Error("expected nil value to be omitted")
	}
}

func TestProps_BooleanFlags(t *testing.T) {
	got := Props(
		"has_priority", true,
		"has_scope", false,
		"has_additional_meta", true,
	)
	if got["has_priority"] != true {
		t.Error("expected has_priority=true")
	}
	if got["has_scope"] != false {
		t.Errorf("expected has_scope=false, got %v", got["has_scope"])
	}
	if got["has_additional_meta"] != true {
		t.Error("expected has_additional_meta=true")
	}
}
