package concurrency

import (
	"runtime"
	"testing"
	"time"
	"unsafe"
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
		cmp := func(a, b *slot) int { return cmp32(b.priority, a.priority) }
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
		cmp := func(a, b slot) int { return cmp32(b.priority, a.priority) }
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
