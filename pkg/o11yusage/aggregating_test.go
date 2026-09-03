package o11yusage

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

func TestAddNilReceiver(t *testing.T) {
	t.Parallel()

	var a *Aggregator
	a.AddLogs(uuid.New(), 10)
	a.AddOtel(uuid.New(), 10)
	if err := a.Shutdown(); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}

func TestFlushKeepsKindsSeparate(t *testing.T) {
	t.Parallel()

	tenantA := uuid.New()
	tenantB := uuid.New()

	var got atomic.Value
	done := make(chan struct{})

	l := zerolog.Nop()
	a := NewAggregator(&l, time.Hour, func(tenants map[uuid.UUID]TenantBytes) error {
		got.Store(tenants)
		close(done)
		return nil
	})

	a.AddLogs(tenantA, 100)
	a.AddLogs(tenantA, 50)
	a.AddOtel(tenantA, 7)
	a.AddOtel(tenantB, 3)
	a.flush()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("flush did not run")
	}

	tenants, ok := got.Load().(map[uuid.UUID]TenantBytes)
	if !ok {
		t.Fatal("expected tenant map")
	}
	if tenants[tenantA] != (TenantBytes{Logs: 150, Otel: 7}) {
		t.Fatalf("tenant A = %+v, want logs=150 otel=7", tenants[tenantA])
	}
	if tenants[tenantB] != (TenantBytes{Otel: 3}) {
		t.Fatalf("tenant B = %+v, want otel=3", tenants[tenantB])
	}

	a.flush()
	if v, ok := a.counters.Load(counterKey{TenantID: tenantA, Kind: KindLogs}); ok && v.(*atomic.Int64).Load() != 0 {
		t.Fatalf("expected counters cleared after flush")
	}
}

func TestShutdownFlushes(t *testing.T) {
	t.Parallel()

	tenantID := uuid.New()
	var mu sync.Mutex
	var flushed map[uuid.UUID]TenantBytes

	l := zerolog.Nop()
	a := NewAggregator(&l, time.Hour, func(tenants map[uuid.UUID]TenantBytes) error {
		mu.Lock()
		defer mu.Unlock()
		flushed = tenants
		return nil
	})
	a.Start()
	a.AddLogs(tenantID, 42)

	if err := a.Shutdown(); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if flushed[tenantID] != (TenantBytes{Logs: 42}) {
		t.Fatalf("flushed = %v, want %s logs=42", flushed, tenantID)
	}
}

func TestDropOnFlushError(t *testing.T) {
	t.Parallel()

	tenantID := uuid.New()
	l := zerolog.Nop()
	a := NewAggregator(&l, time.Hour, func(tenants map[uuid.UUID]TenantBytes) error {
		return errFlush
	})
	a.AddOtel(tenantID, 9)
	a.flush()

	if v, ok := a.counters.Load(counterKey{TenantID: tenantID, Kind: KindOtel}); ok && v.(*atomic.Int64).Load() != 0 {
		t.Fatalf("expected dropped snapshot after flush error, still have %d", v.(*atomic.Int64).Load())
	}
}

var errFlush = errString("flush failed")

type errString string

func (e errString) Error() string { return string(e) }
