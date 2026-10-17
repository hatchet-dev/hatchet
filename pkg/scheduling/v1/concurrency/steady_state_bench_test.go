package concurrency

import (
	"context"
	"fmt"
	"math/rand/v2"
	"runtime"
	"runtime/metrics"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// steadyShape describes a backlog to hydrate and the WAL traffic to run against it.
type steadyShape struct {
	name    string
	kind    sqlcv1.V1ConcurrencyStrategy
	maxRuns int32
	// keys are hydrated by the build; arrivals pick from keySpace >= keys, so keys beyond keys start
	// empty and are created (and later pruned) by WAL traffic.
	keys     int
	keySpace int
	// running filled slots on every hydrated key, plus queued slots on every backlogEvery-th key
	running      int
	queued       int
	backlogEvery int
	batch        int
}

var steadyShapes = []steadyShape{
	// many keys that each run one task, a few with a backlog; new keys keep appearing and emptying
	{
		name: "many-keys", kind: sqlcv1.V1ConcurrencyStrategyCANCELQUEUEDEXCEPTNEWEST, maxRuns: 1,
		keys: 500_000, keySpace: 1_000_000, running: 1, queued: 2, backlogEvery: 10, batch: 1_000,
	},
	// keys with spare capacity: most arrivals are filled in the batch they arrive in, so their
	// queues empty every batch
	{
		name: "drained-queues", kind: sqlcv1.V1ConcurrencyStrategyGROUPROUNDROBIN, maxRuns: 4,
		keys: 50_000, keySpace: 50_000, running: 2, batch: 1_000,
	},
	// keys at their limit, where most arrivals preempt a runner
	{
		name: "cancel-in-progress", kind: sqlcv1.V1ConcurrencyStrategyCANCELINPROGRESS, maxRuns: 2,
		keys: 50_000, keySpace: 50_000, running: 2, batch: 1_000,
	},
	// few keys with deep queues that never empty
	{
		name: "deep-backlog", kind: sqlcv1.V1ConcurrencyStrategyGROUPROUNDROBIN, maxRuns: 5,
		keys: 200, keySpace: 200, running: 5, queued: 5_000, backlogEvery: 1, batch: 1_000,
	},
	// keys running more than maxSmallRunning slots, with a backlog
	{
		name: "large-running", kind: sqlcv1.V1ConcurrencyStrategyGROUPROUNDROBIN, maxRuns: 100,
		keys: 100, keySpace: 100, running: 100, queued: 200, backlogEvery: 1, batch: 1_000,
	},
}

const steadyKeyLen = len("key-0000000000")

// steadyDriver plays the database's role: it hands the index its backlog and feeds it WAL batches.
// Each batch carries a DELETE for every slot the previous batch cancelled (the slot row's delete
// trigger echoes it back), completes the oldest runners so the number of running tasks stays
// constant, and fills up with INSERTs on uniformly random keys.
//
// Task ids encode their key (id = seq*keySpace + key) and grow with insert time, so the driver needs
// no map from task to key. Fills and cancels are sorted before they feed the next batch, so the
// WAL stream depends only on the index's decisions, not on goroutine scheduling.
type steadyDriver struct {
	shape steadyShape
	// keyBuf holds every key name back to back; key k is a substring, so keys cost the GC nothing
	keyBuf  string
	rng     *rand.Rand
	start   int64
	timeout int64
	seq     int64
	// running is a FIFO of filled task ids, oldest first, from runHead on
	running []int64
	runHead int
	// live and target count running tasks; completions bring live back to target
	live   int
	target int
	// runners is the set of running tasks, kept only for CANCEL_IN_PROGRESS, which cancels runners:
	// the FIFO skips tasks that are no longer in it
	runners map[int64]struct{}
	echoes  []int64
	msgs    []walMessage

	filled, cancels, fillSum int64
}

func newSteadyDriver(shape steadyShape) *steadyDriver {
	var keys strings.Builder
	keys.Grow(shape.keySpace * steadyKeyLen)

	for k := 0; k < shape.keySpace; k++ {
		fmt.Fprintf(&keys, "key-%010d", k)
	}

	start := time.Now().UTC()
	runners := shape.keys * shape.running

	d := &steadyDriver{
		shape:   shape,
		keyBuf:  keys.String(),
		rng:     rand.New(rand.NewPCG(1, 2)),
		start:   start.UnixNano(),
		timeout: start.Add(1000 * time.Hour).UnixMilli(),
		// the FIFO is compacted once half of it is consumed, so it peaks at about twice its length
		running: make([]int64, 0, 2*runners+2*shape.batch),
		echoes:  make([]int64, 0, 2*shape.batch),
		msgs:    make([]walMessage, 0, 3*shape.batch),
	}

	if shape.kind == sqlcv1.V1ConcurrencyStrategyCANCELINPROGRESS {
		d.runners = make(map[int64]struct{}, runners+shape.batch)
	}

	return d
}

func (d *steadyDriver) key(k int) string {
	return d.keyBuf[k*steadyKeyLen : (k+1)*steadyKeyLen]
}

func (d *steadyDriver) nextTask(key int) int64 {
	d.seq++
	return d.seq*int64(d.shape.keySpace) + int64(key)
}

func (d *steadyDriver) keyOf(taskId int64) string {
	return d.key(int(taskId % int64(d.shape.keySpace)))
}

// rows returns the hydrated backlog: per key, running slots first, then queued ones.
func (d *steadyDriver) rows() []*sqlcv1.ListConcurrencySlotsForIndexingRow {
	s := d.shape
	rows := make([]*sqlcv1.ListConcurrencySlotsForIndexingRow, 0, s.keys*s.running+s.keys/max(s.backlogEvery, 1)*s.queued)
	timeout := time.UnixMilli(d.timeout)

	for k := 0; k < s.keys; k++ {
		queued := 0
		if s.backlogEvery > 0 && k%s.backlogEvery == 0 {
			queued = s.queued
		}

		for i := 0; i < s.running+queued; i++ {
			id := d.nextTask(k)
			filled := i < s.running
			rows = append(rows, indexRow(d.key(k), id, int32(id%3), 0, time.Unix(0, d.start+id), timeout, filled))

			if filled {
				d.track(id)
			}
		}
	}

	return rows
}

func (d *steadyDriver) track(id int64) {
	d.running = append(d.running, id)
	d.live++

	if d.runners != nil {
		d.runners[id] = struct{}{}
	}
}

// record takes the fills and cancels of the last flush.
func (d *steadyDriver) record(repo *mockConcurrencyRepo) {
	n := len(d.running)

	for _, f := range repo.lastFilled {
		d.track(f.Id)
		d.fillSum += f.Id
	}

	slices.Sort(d.running[n:])

	for _, c := range repo.lastCancelled {
		d.echoes = append(d.echoes, c.Id)

		if _, ok := d.runners[c.Id]; ok {
			delete(d.runners, c.Id)
			d.live--
		}
	}

	slices.Sort(d.echoes)

	d.filled += int64(len(repo.lastFilled))
	d.cancels += int64(len(repo.lastCancelled))
	repo.lastFilled, repo.lastCancelled = nil, nil
}

func (d *steadyDriver) nextBatch() []walMessage {
	msgs := d.msgs[:0]

	for _, id := range d.echoes {
		msgs = append(msgs, walMessage{Operation: "DELETE", Key: d.keyOf(id), TaskId: id})
	}

	d.echoes = d.echoes[:0]

	for d.live > d.target && len(msgs) < d.shape.batch && d.runHead < len(d.running) {
		id := d.running[d.runHead]
		d.runHead++

		if d.runners != nil {
			if _, ok := d.runners[id]; !ok {
				// preempted: its cancel already took it out of live
				continue
			}

			delete(d.runners, id)
		}

		msgs = append(msgs, walMessage{Operation: "DELETE", Key: d.keyOf(id), TaskId: id})
		d.live--
	}

	if d.runHead > len(d.running)/2 {
		n := copy(d.running, d.running[d.runHead:])
		d.running = d.running[:n]
		d.runHead = 0
	}

	for len(msgs) < d.shape.batch {
		k := d.rng.IntN(d.shape.keySpace)
		id := d.nextTask(k)
		msgs = append(msgs, walMessage{
			Operation:           "INSERT",
			Key:                 d.key(k),
			TaskId:              id,
			Priority:            int32(id % 3),
			TaskInsertedAt:      time.Unix(0, d.start+id),
			ScheduleTimeoutAtMs: d.timeout,
		})
	}

	d.msgs = msgs

	return msgs
}

func newSteadyStrategy(d *steadyDriver, rows []*sqlcv1.ListConcurrencySlotsForIndexingRow) (*ConcurrencyStrategy, *mockConcurrencyRepo) {
	// an empty result keeps the mock from allocating one per flush
	repo := &mockConcurrencyRepo{indexRows: rows, updateResult: &repository.RunConcurrencyResult{}}

	return newTestStrategyKind(repo, d.shape.maxRuns, d.shape.kind), repo
}

// BenchmarkIndexBuild reports what building an index and running the post-build pass allocates.
//
//	go test -run x -bench BenchmarkIndexBuild -benchtime 5x -count 10 ./pkg/scheduling/v1/concurrency/
func BenchmarkIndexBuild(b *testing.B) {
	for _, shape := range steadyShapes {
		b.Run(shape.name, func(b *testing.B) {
			d := newSteadyDriver(shape)
			rows := d.rows()

			b.ReportAllocs()

			for b.Loop() {
				c, _ := newSteadyStrategy(d, rows)

				if err := c.buildIndex(context.Background()); err != nil {
					b.Fatalf("buildIndex: %v", err)
				}
				if _, err := c.queueAllSubQueues(context.Background()); err != nil {
					b.Fatalf("queueAllSubQueues: %v", err)
				}
			}
		})
	}
}

// BenchmarkSteadyState hydrates an index, then runs WAL batches against it the way Run does: process
// a batch, commit its undo scopes and prune emptied keys. One op is one batch of 1000 messages, and
// all ops run against the same index, so results are only comparable at the same -benchtime Nx.
// Besides time and allocations per batch it reports:
//
//   - cpu-ns/op: process CPU time, including GC work on other threads
//   - gc-cpu-ns/op: the runtime's estimate of GC CPU time, without GC work done on idle cores
//   - gcs/kop: GC cycles per 1000 batches; this falls as the live heap grows, so compare GC CPU
//   - live-MiB: the live heap held by the index after the last batch
//   - fills/op, cancels/op: the index's decisions, equal on any two versions that decide the same
//
// The GC metrics only move when a cycle ends, so run enough batches for a few dozen cycles:
//
//	go test -run x -bench BenchmarkSteadyState -benchtime 15000x -count 10 ./pkg/scheduling/v1/concurrency/
func BenchmarkSteadyState(b *testing.B) {
	for _, shape := range steadyShapes {
		b.Run(shape.name, func(b *testing.B) {
			d := newSteadyDriver(shape)

			// the index is unreachable once runSteadyState returns
			held := runSteadyState(b, d)
			b.ReportMetric(float64(held-liveHeap())/(1<<20), "live-MiB")
		})
	}
}

// runSteadyState builds the index, runs the batches and returns the live heap with the index held.
func runSteadyState(b *testing.B, d *steadyDriver) (held int64) {
	ctx := context.Background()
	c, repo := newSteadyStrategy(d, d.rows())

	if err := c.buildIndex(ctx); err != nil {
		b.Fatalf("buildIndex: %v", err)
	}
	if _, err := c.queueAllSubQueues(ctx); err != nil {
		b.Fatalf("queueAllSubQueues: %v", err)
	}

	repo.indexRows = nil
	d.record(repo)
	d.target = d.live - d.shape.batch/2
	d.filled, d.cancels, d.fillSum = 0, 0, 0

	// start from a collected heap, so the GC pacer sizes its goal by the index rather than by the
	// garbage hydration left behind
	runtime.GC()

	b.ReportAllocs()
	before := readGCStats()
	n := 0

	for b.Loop() {
		if _, err := c.processWALMessages(ctx, nil, d.nextBatch()); err != nil {
			b.Fatalf("processWALMessages: %v", err)
		}

		c.pruneEmpty(c.commitScopes())
		d.record(repo)
		n++
	}

	after := readGCStats()
	ops := float64(n)

	b.ReportMetric(float64(after.cpu-before.cpu)/ops, "cpu-ns/op")
	b.ReportMetric((after.gcCPU-before.gcCPU)*1e9/ops, "gc-cpu-ns/op")
	b.ReportMetric(float64(after.cycles-before.cycles)*1000/ops, "gcs/kop")
	b.ReportMetric(float64(d.filled)/ops, "fills/op")
	b.ReportMetric(float64(d.cancels)/ops, "cancels/op")
	b.Logf("%d batches: %d keys, %d running, fill id sum %d", n, len(c.subQueues), d.live, d.fillSum)

	held = liveHeap()
	runtime.KeepAlive(c)

	return held
}

type gcStats struct {
	cpu    time.Duration
	gcCPU  float64
	cycles uint64
}

func readGCStats() gcStats {
	samples := []metrics.Sample{
		{Name: "/cpu/classes/gc/total:cpu-seconds"},
		{Name: "/cpu/classes/gc/mark/idle:cpu-seconds"},
		{Name: "/gc/cycles/total:gc-cycles"},
	}
	metrics.Read(samples)

	var ru syscall.Rusage
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &ru)

	return gcStats{
		cpu:    time.Duration(ru.Utime.Nano() + ru.Stime.Nano()),
		gcCPU:  samples[0].Value.Float64() - samples[1].Value.Float64(),
		cycles: samples[2].Value.Uint64(),
	}
}
