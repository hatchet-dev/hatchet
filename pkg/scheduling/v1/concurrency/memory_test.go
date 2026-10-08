package concurrency

import (
	"cmp"
	"context"
	"fmt"
	"os"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// measureHeap returns the live heap bytes retained by building the index via fn(n).
func measureHeap(t *testing.T, n int, trackTimeouts bool) int64 {
	t.Helper()

	now := time.Now().UTC()
	timeout := now.Add(time.Hour)

	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	idx := newInMemorySlotIndex(trackTimeouts)
	for i := 0; i < n; i++ {
		idx.insert(slot{
			priority:            int32(i % 100),
			taskId:              int64(i),
			taskInsertedAtNs:    now.UnixNano(),
			taskRetryCount:      0,
			scheduleTimeoutAtMs: timeout.UnixMilli(),
		})
	}

	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	runtime.KeepAlive(idx)
	return int64(after.HeapAlloc) - int64(before.HeapAlloc)
}

// measureSingleHeap builds one heap + one taskId->index map of n elements, storing either slot
// values or *slot pointers, and returns the retained live heap bytes.
func measureSingleHeap(t *testing.T, n int, usePointers bool) int64 {
	t.Helper()

	now := time.Now().UTC()
	timeout := now.Add(time.Hour)
	mk := func(i int) slot {
		return slot{priority: int32(i % 100), taskId: int64(i), taskInsertedAtNs: now.UnixNano(), scheduleTimeoutAtMs: timeout.UnixMilli()}
	}

	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	var keep any
	if usePointers {
		m := make(map[int64]int)
		set := func(s *slot, i int) {
			if i < 0 {
				delete(m, s.taskId)
				return
			}
			m[s.taskId] = i
		}
		get := func(s *slot) (int, bool) { i, ok := m[s.taskId]; return i, ok }
		cmp := func(a, b *slot) int { return cmp.Compare(b.priority, a.priority) }
		h := newHeap(cmp, set, get)
		for i := 0; i < n; i++ {
			s := mk(i)
			h.insert(&s)
		}
		keep = h
	} else {
		m := make(map[int64]int)
		set := func(s slot, i int) {
			if i < 0 {
				delete(m, s.taskId)
				return
			}
			m[s.taskId] = i
		}
		get := func(s slot) (int, bool) { i, ok := m[s.taskId]; return i, ok }
		cmp := func(a, b slot) int { return cmp.Compare(b.priority, a.priority) }
		h := newHeap(cmp, set, get)
		for i := 0; i < n; i++ {
			h.insert(mk(i))
		}
		keep = h
	}

	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	runtime.KeepAlive(keep)
	return int64(after.HeapAlloc) - int64(before.HeapAlloc)
}

func TestSlotPointerVsValue(t *testing.T) {
	if testing.Short() {
		t.Skip("memory measurement; skipped under -short")
	}
	const n = 1_000_000

	value := measureSingleHeap(t, n, false)
	ptr := measureSingleHeap(t, n, true)
	t.Logf("single heap+map, []slot:  %6.1f bytes/elem", float64(value)/float64(n))
	t.Logf("single heap+map, []*slot: %6.1f bytes/elem", float64(ptr)/float64(n))
}

func TestIndexMemoryFootprint(t *testing.T) {
	if testing.Short() {
		t.Skip("memory measurement; skipped under -short")
	}

	const n = 1_000_000

	t.Logf("unsafe.Sizeof(slot) = %d bytes", unsafe.Sizeof(slot{}))

	queued := measureHeap(t, n, true)
	t.Logf("queued index   (2 heaps + 1 map): %7.1f MiB total, %6.1f bytes/slot",
		float64(queued)/(1<<20), float64(queued)/float64(n))

	running := measureHeap(t, n, false)
	t.Logf("running index  (1 heap + 1 map):  %7.1f MiB total, %6.1f bytes/slot",
		float64(running)/(1<<20), float64(running)/float64(n))
}

// generatedRowsRepo streams index rows from gen instead of holding them, so tests can hydrate
// millions of slots, and so key strings are allocated by the index as they are in production.
type generatedRowsRepo struct {
	mockConcurrencyRepo
	gen func(emit func(*sqlcv1.ListConcurrencySlotsForIndexingRow))
}

func (r *generatedRowsRepo) ReadConcurrencySlotsForIndexing(ctx context.Context, tenantId uuid.UUID, strategyId int64, writeCh chan<- *sqlcv1.ListConcurrencySlotsForIndexingRow) error {
	r.gen(func(row *sqlcv1.ListConcurrencySlotsForIndexingRow) { writeCh <- row })
	return nil
}

// keyShape describes the slots every key holds when the index is built.
type keyShape struct {
	kind    sqlcv1.V1ConcurrencyStrategy
	maxRuns int32
	filled  int
	queued  int
}

// shapeRows emits keys keys of the given shape, with task ids and insert times increasing in
// emission order.
func shapeRows(keys int, shape keyShape) func(emit func(*sqlcv1.ListConcurrencySlotsForIndexingRow)) {
	return func(emit func(*sqlcv1.ListConcurrencySlotsForIndexingRow)) {
		start := time.Now().UTC()
		timeout := start.Add(time.Hour)
		id := int64(0)

		for k := 0; k < keys; k++ {
			key := fmt.Sprintf("key-%010d", k)

			for i := 0; i < shape.filled+shape.queued; i++ {
				id++
				emit(indexRow(key, id, 0, 0, start.Add(time.Duration(id)), timeout, i < shape.filled))
			}
		}
	}
}

func liveHeap() int64 {
	runtime.GC()
	runtime.GC()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return int64(m.HeapAlloc)
}

// BenchmarkKeyMemory reports the live heap retained per concurrency key after the index is built
// and the post-build queueing pass has run.
//
//	go test -run x -bench BenchmarkKeyMemory ./pkg/scheduling/v1/concurrency/
func BenchmarkKeyMemory(b *testing.B) {
	const keys = 100_000

	for _, tc := range []struct {
		name  string
		shape keyShape
	}{
		{"filled=1", keyShape{kind: sqlcv1.V1ConcurrencyStrategyCANCELQUEUEDEXCEPTNEWEST, maxRuns: 1, filled: 1}},
		{"filled=4", keyShape{kind: sqlcv1.V1ConcurrencyStrategyGROUPROUNDROBIN, maxRuns: 4, filled: 4}},
		{"filled=1,queued=1", keyShape{kind: sqlcv1.V1ConcurrencyStrategyCANCELQUEUEDEXCEPTNEWEST, maxRuns: 1, filled: 1, queued: 1}},
		// the pass fills one slot, cancels the oldest 6 and keeps the newest queued
		{"queued=8", keyShape{kind: sqlcv1.V1ConcurrencyStrategyCANCELQUEUEDEXCEPTNEWEST, maxRuns: 1, queued: 8}},
	} {
		b.Run(tc.name, func(b *testing.B) {
			var perKey float64

			for b.Loop() {
				repo := &generatedRowsRepo{gen: shapeRows(keys, tc.shape)}

				before := liveHeap()

				c := newTestStrategyKind(repo, tc.shape.maxRuns, tc.shape.kind)
				if err := c.buildIndex(context.Background()); err != nil {
					b.Fatalf("buildIndex: %v", err)
				}
				if _, err := c.queueAllSubQueues(context.Background()); err != nil {
					b.Fatalf("queueAllSubQueues: %v", err)
				}
				repo.lastFilled, repo.lastCancelled = nil, nil

				perKey = float64(liveHeap()-before) / keys
				runtime.KeepAlive(c)
			}

			b.ReportMetric(perKey, "B/key")
		})
	}
}

// sampleHeapPeak records the largest HeapAlloc seen until the returned stop func is called.
func sampleHeapPeak() (stop func() int64) {
	var peak atomic.Int64
	done := make(chan struct{})
	finished := make(chan struct{})

	go func() {
		defer close(finished)

		ticker := time.NewTicker(5 * time.Millisecond)
		defer ticker.Stop()

		for {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			if int64(m.HeapAlloc) > peak.Load() {
				peak.Store(int64(m.HeapAlloc))
			}

			select {
			case <-done:
				return
			case <-ticker.C:
			}
		}
	}()

	return func() int64 {
		close(done)
		<-finished
		return peak.Load()
	}
}

// TestIndexMemoryAtScale hydrates two strategies with a backlog of roughly 15M slots and checks
// that the index stays well under a 12 GiB scheduler while making the expected decisions:
//
//   - CANCEL_QUEUED_EXCEPT_NEWEST, max_concurrency 1, 5M keys with one filled slot each; every 10th
//     key also has two queued slots.
//   - GROUP_ROUND_ROBIN, max_concurrency 5, 200 keys with 50k queued slots each.
//
// It needs several GiB, so it only runs when HATCHET_CONCURRENCY_SCALE_TEST is set:
//
//	HATCHET_CONCURRENCY_SCALE_TEST=1 go test -run TestIndexMemoryAtScale -v -timeout 30m ./pkg/scheduling/v1/concurrency/
func TestIndexMemoryAtScale(t *testing.T) {
	if os.Getenv("HATCHET_CONCURRENCY_SCALE_TEST") == "" {
		t.Skip("set HATCHET_CONCURRENCY_SCALE_TEST to run")
	}

	const (
		manyKeys      = 5_000_000
		backlogEvery  = 10
		fewKeys       = 200
		queuedPerKey  = 50_000
		groupMaxRuns  = 5
		completedKeys = 100_000
	)

	ctx := context.Background()
	start := time.Now().UTC()
	timeout := start.Add(24 * time.Hour)

	before := liveHeap()
	stopPeak := sampleHeapPeak()

	// key k holds running slot 3k+1 and, on backlog keys, queued slots 3k+2 and 3k+3.
	manyRepo := &generatedRowsRepo{gen: func(emit func(*sqlcv1.ListConcurrencySlotsForIndexingRow)) {
		for k := int64(0); k < manyKeys; k++ {
			key := fmt.Sprintf("key-%010d", k)
			emit(indexRow(key, 3*k+1, 0, 0, start.Add(time.Duration(3*k+1)), timeout, true))

			if k%backlogEvery == 0 {
				emit(indexRow(key, 3*k+2, 0, 0, start.Add(time.Duration(3*k+2)), timeout, false))
				emit(indexRow(key, 3*k+3, 0, 0, start.Add(time.Duration(3*k+3)), timeout, false))
			}
		}
	}}
	many := newTestStrategyKind(manyRepo, 1, sqlcv1.V1ConcurrencyStrategyCANCELQUEUEDEXCEPTNEWEST)

	if err := many.buildIndex(ctx); err != nil {
		t.Fatalf("buildIndex: %v", err)
	}
	if _, err := many.queueAllSubQueues(ctx); err != nil {
		t.Fatalf("queueAllSubQueues: %v", err)
	}

	// the pass cancels the older queued slot on every backlog key and fills nothing
	if len(manyRepo.lastFilled) != 0 {
		t.Fatalf("filled %d slots, want 0", len(manyRepo.lastFilled))
	}
	if len(manyRepo.lastCancelled) != manyKeys/backlogEvery {
		t.Fatalf("cancelled %d slots, want %d", len(manyRepo.lastCancelled), manyKeys/backlogEvery)
	}
	for _, c := range manyRepo.lastCancelled {
		if c.Id%3 != 2 || (c.Id/3)%backlogEvery != 0 {
			t.Fatalf("cancelled task %d, want only the older queued slot of a backlog key", c.Id)
		}
	}
	manyRepo.lastCancelled = nil

	// complete the running slot of the first completedKeys keys: backlog keys promote their newest
	// queued slot, the rest are left empty and pruned
	completions := make([]walMessage, 0, completedKeys)
	for k := int64(0); k < completedKeys; k++ {
		completions = append(completions, walMessage{Operation: "DELETE", Key: fmt.Sprintf("key-%010d", k), TaskId: 3*k + 1})
	}
	if _, err := many.processWALMessages(ctx, nil, completions); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}
	many.pruneEmpty(many.commitScopes())
	completions = nil

	if len(manyRepo.lastFilled) != completedKeys/backlogEvery {
		t.Fatalf("filled %d slots, want %d", len(manyRepo.lastFilled), completedKeys/backlogEvery)
	}
	for _, f := range manyRepo.lastFilled {
		if f.Id%3 != 0 {
			t.Fatalf("filled task %d, want only the newest queued slot of a backlog key", f.Id)
		}
	}
	if len(manyRepo.lastCancelled) != 0 {
		t.Fatalf("cancelled %d slots on completion, want 0", len(manyRepo.lastCancelled))
	}
	manyRepo.lastFilled = nil

	if got, want := len(many.subQueues), manyKeys-completedKeys+completedKeys/backlogEvery; got != want {
		t.Fatalf("resident keys = %d, want %d", got, want)
	}

	// key g holds queued slots g*queuedPerKey+1 .. (g+1)*queuedPerKey, oldest first.
	fewRepo := &generatedRowsRepo{gen: func(emit func(*sqlcv1.ListConcurrencySlotsForIndexingRow)) {
		for g := int64(0); g < fewKeys; g++ {
			key := fmt.Sprintf("group-%03d", g)

			for i := int64(1); i <= queuedPerKey; i++ {
				id := g*queuedPerKey + i
				emit(indexRow(key, id, 0, 0, start.Add(time.Duration(id)), timeout, false))
			}
		}
	}}
	few := newTestStrategyKind(fewRepo, groupMaxRuns, sqlcv1.V1ConcurrencyStrategyGROUPROUNDROBIN)

	if err := few.buildIndex(ctx); err != nil {
		t.Fatalf("buildIndex: %v", err)
	}
	if _, err := few.queueAllSubQueues(ctx); err != nil {
		t.Fatalf("queueAllSubQueues: %v", err)
	}

	// the pass fills the oldest groupMaxRuns slots of every key
	if len(fewRepo.lastFilled) != fewKeys*groupMaxRuns {
		t.Fatalf("filled %d slots, want %d", len(fewRepo.lastFilled), fewKeys*groupMaxRuns)
	}
	for _, f := range fewRepo.lastFilled {
		if (f.Id-1)%queuedPerKey >= groupMaxRuns {
			t.Fatalf("filled task %d, want only the oldest %d slots of each key", f.Id, groupMaxRuns)
		}
	}
	if len(fewRepo.lastCancelled) != 0 {
		t.Fatalf("cancelled %d slots, want 0", len(fewRepo.lastCancelled))
	}
	fewRepo.lastFilled = nil

	peak := stopPeak() - before
	live := liveHeap() - before
	estimate := estimatedBytes(int64(len(many.subQueues)+len(few.subQueues)), many.resident.load().plus(few.resident.load()))

	t.Logf("live heap %.2f GiB, peak heap %.2f GiB, estimated %.2f GiB",
		float64(live)/(1<<30), float64(peak)/(1<<30), float64(estimate)/(1<<30))

	const budget = 12 << 30
	if live > budget/4 || peak > budget/2 {
		t.Fatalf("live heap %d B, peak heap %d B; want at most %d and %d", live, peak, budget/4, budget/2)
	}
	if estimate < live*3/4 || estimate > live*5/4 {
		t.Fatalf("estimated %d B for a live heap of %d B, want within 25%%", estimate, live)
	}

	runtime.KeepAlive(many)
	runtime.KeepAlive(few)
}
