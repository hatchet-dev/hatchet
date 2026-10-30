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

func TestFlushEvictionRaceKeepsEveryByte(t *testing.T) {
	t.Parallel()

	tenantID := uuid.New()
	var total atomic.Int64

	l := zerolog.Nop()
	// A long interval keeps the ticker out of the way. The test drives flush
	// directly so that it laps the writers constantly and the zero-count
	// eviction branch runs while Add calls are in flight.
	a := NewAggregator(&l, time.Hour, func(tenants map[uuid.UUID]TenantBytes) error {
		for _, tb := range tenants {
			total.Add(tb.Logs)
		}
		return nil
	})

	const goroutines = 8
	const addsPerGoroutine = 2000

	stop := make(chan struct{})
	var flusher sync.WaitGroup
	flusher.Add(1)
	go func() {
		defer flusher.Done()
		for {
			select {
			case <-stop:
				return
			default:
				a.flush()
			}
		}
	}()

	var writers sync.WaitGroup
	writers.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer writers.Done()
			for j := 0; j < addsPerGoroutine; j++ {
				a.AddLogs(tenantID, 1)
			}
		}()
	}
	writers.Wait()
	close(stop)
	flusher.Wait()
	a.flush()

	if got := total.Load(); got != goroutines*addsPerGoroutine {
		t.Fatalf("flushed %d bytes, want %d", got, goroutines*addsPerGoroutine)
	}
}

func TestAddEvictedWriterRetriesIntoFreshCounter(t *testing.T) {
	t.Parallel()

	tenantID := uuid.New()
	var mu sync.Mutex
	var flushes []map[uuid.UUID]TenantBytes

	l := zerolog.Nop()
	a := NewAggregator(&l, time.Hour, func(tenants map[uuid.UUID]TenantBytes) error {
		mu.Lock()
		defer mu.Unlock()
		flushes = append(flushes, tenants)
		return nil
	})

	// Leave an idle counter behind so the next snapshot evicts it.
	a.AddLogs(tenantID, 1)
	a.flush()

	// Park the writer between its lookup of the idle counter and its
	// increment, evict and seal the counter underneath it, then let the
	// increment land. It must be retried into a fresh counter. The waits
	// are bounded so that a regression fails instead of hanging, and the
	// release is idempotent and runs at cleanup so a failed assertion never
	// leaves the writer blocked.
	const barrierTimeout = 10 * time.Second
	loaded := make(chan struct{})
	resume := make(chan struct{})
	done := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(resume) }) }
	t.Cleanup(func() {
		release()
		select {
		case <-done:
		case <-time.After(barrierTimeout):
			t.Error("parked writer did not finish after release")
		}
	})

	var calls atomic.Int64
	a.afterLoad = func() {
		if calls.Add(1) == 1 {
			close(loaded)
			<-resume
		}
	}
	go func() {
		defer close(done)
		a.AddLogs(tenantID, 5)
	}()
	select {
	case <-loaded:
	case <-time.After(barrierTimeout):
		t.Fatal("writer never reached afterLoad between Load and Add")
	}

	a.flush()
	release()
	select {
	case <-done:
	case <-time.After(barrierTimeout):
		t.Fatal("writer did not finish its Add after being released")
	}
	a.flush()

	mu.Lock()
	defer mu.Unlock()
	// The eviction flush had nothing to report, so only two flushes ran the
	// callback: the setup one and the one carrying the retried increment.
	if len(flushes) != 2 {
		t.Fatalf("got %d flushes, want 2: %v", len(flushes), flushes)
	}
	if flushes[1][tenantID] != (TenantBytes{Logs: 5}) {
		t.Fatalf("retried increment flushed as %+v, want logs=5", flushes[1][tenantID])
	}
	if _, ok := a.counters.Load(counterKey{TenantID: tenantID, Kind: KindLogs}); !ok {
		t.Fatal("expected the fresh counter to still be in the map after its flush")
	}
}

var errFlush = errString("flush failed")

type errString string

func (e errString) Error() string { return string(e) }
