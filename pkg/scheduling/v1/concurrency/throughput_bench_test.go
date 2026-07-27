package concurrency

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// BenchmarkConcurrencyThroughput measures how quickly the in-memory strategy drains a large backlog
// of queued concurrency slots when each flush of concurrency updates to the database costs ~1ms (the
// flushLatency seam on mockConcurrencyRepo); everything else is in-memory. Two shapes are covered:
//
//   - wal/batch=N: the steady-state outbox path. Slots arrive as INSERT WAL messages in batches of N;
//     each batch is one processWALMessages call = one 1ms flush. Throughput is bound by the number of
//     flushes, so larger batches amortize the per-flush latency.
//   - initial-drain: the post-build pass. buildIndex hydrates the whole backlog, then a single
//     queueAllSubQueues call decides + flushes it all in one flush. Throughput is CPU-bound (the
//     in-memory decide work), not latency-bound.
//
// Run with:
//
//	go test -run x -bench BenchmarkConcurrencyThroughput -benchmem ./pkg/scheduling/v1/concurrency/
func BenchmarkConcurrencyThroughput(b *testing.B) {
	const (
		totalSlots     = 200_000
		numKeys        = 200
		maxConcurrency = 100
		flushLatency   = time.Millisecond
	)

	for _, batchSize := range []int{100, 1_000, 10_000} {
		batches := buildWALBatches(totalSlots, numKeys, batchSize)

		b.Run(fmt.Sprintf("wal/batch=%d", batchSize), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				b.StopTimer()
				repo := &mockConcurrencyRepo{flushLatency: flushLatency}
				c := newGroupRoundRobinStrategy(repo, maxConcurrency)
				b.StartTimer()

				for _, batch := range batches {
					if _, err := c.processWALMessages(context.Background(), nil, batch); err != nil {
						b.Fatalf("processWALMessages: %v", err)
					}
					// mirror Run's success path: commit the undo scopes and prune emptied sub-queues.
					c.pruneEmpty(c.commitScopes())
				}
			}

			b.StopTimer()
			b.ReportMetric(float64(totalSlots)*float64(b.N)/b.Elapsed().Seconds(), "slots/sec")
		})
	}

	b.Run("initial-drain", func(b *testing.B) {
		rows := buildIndexRows(totalSlots, numKeys)

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			b.StopTimer()
			repo := &mockConcurrencyRepo{flushLatency: flushLatency, indexRows: rows}
			c := newGroupRoundRobinStrategy(repo, maxConcurrency)
			if err := c.buildIndex(context.Background()); err != nil {
				b.Fatalf("buildIndex: %v", err)
			}
			b.StartTimer()

			if _, err := c.queueAllSubQueues(context.Background()); err != nil {
				b.Fatalf("queueAllSubQueues: %v", err)
			}
		}

		b.StopTimer()
		b.ReportMetric(float64(totalSlots)*float64(b.N)/b.Elapsed().Seconds(), "slots/sec")
	})
}

// buildWALBatches produces total INSERT messages spread round-robin across numKeys sub-queues, split
// into batches of batchSize (the last batch may be short). Task ids are unique so no slot supersedes
// another.
func buildWALBatches(total, numKeys, batchSize int) [][]walMessage {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	batches := make([][]walMessage, 0, (total+batchSize-1)/batchSize)
	cur := make([]walMessage, 0, batchSize)

	for i := 0; i < total; i++ {
		key := fmt.Sprintf("key-%d", i%numKeys)
		// vary priority so the heap ordering is actually exercised rather than insertion-ordered.
		cur = append(cur, walInsert(key, int64(i), int32(i%10), now, future))

		if len(cur) == batchSize {
			batches = append(batches, cur)
			cur = make([]walMessage, 0, batchSize)
		}
	}

	if len(cur) > 0 {
		batches = append(batches, cur)
	}

	return batches
}

// buildIndexRows produces total queued (is_filled = false) index rows spread round-robin across
// numKeys sub-queues, for the initial-drain benchmark to hydrate via buildIndex.
func buildIndexRows(total, numKeys int) []*sqlcv1.ListConcurrencySlotsForIndexingRow {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	rows := make([]*sqlcv1.ListConcurrencySlotsForIndexingRow, 0, total)

	for i := 0; i < total; i++ {
		key := fmt.Sprintf("key-%d", i%numKeys)
		rows = append(rows, indexRow(key, int64(i), int32(i%10), 0, now, future, false))
	}

	return rows
}
