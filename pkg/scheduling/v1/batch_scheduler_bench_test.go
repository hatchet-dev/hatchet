package v1

import (
	"fmt"
	"io"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func benchBatchedQueueItem(id int64, tenantId, stepId uuid.UUID, batchKey string) *sqlcv1.V1BatchedQueueItem {
	now := pgtype.Timestamptz{Valid: true, Time: time.Now()}

	return &sqlcv1.V1BatchedQueueItem{
		ID:             id,
		TenantID:       tenantId,
		Queue:          "default",
		TaskID:         id,
		TaskInsertedAt: now,
		ExternalID:     uuid.New(),
		ActionID:       "some-reasonably-long-action-name",
		StepID:         stepId,
		WorkflowID:     uuid.New(),
		WorkflowRunID:  uuid.New(),
		BatchKey:       batchKey,
		InsertedAt:     now,
		PayloadSize:    512,
	}
}

// BenchmarkBatchGroupBufferAppend measures the marginal memory cost of buffering one
// V1BatchedQueueItem in a batchGroup. This is the actual driver of BatchScheduler memory usage:
// there is currently no cap on how large group.buffer can grow, so if flushes stop succeeding
// (e.g. no available workers) items accumulate here indefinitely instead of draining. -benchmem's
// B/op is the number to project against an expected worst-case backlog size.
//
//	go test ./pkg/scheduling/v1/... -run '^$' -bench BenchmarkBatchGroupBufferAppend -benchmem
func BenchmarkBatchGroupBufferAppend(b *testing.B) {
	tenantId := uuid.New()
	stepId := uuid.New()

	group := &batchGroup{batchKey: "", l: new(zerolog.New(io.Discard))}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		group.buffer = append(group.buffer, benchBatchedQueueItem(int64(i), tenantId, stepId, ""))
	}
}

// BenchmarkBatchSchedulerWorstCaseMemory populates a single BatchScheduler with a worst-case
// backlog: numGroups batch keys under one step, each holding maxBufferedPerGroup items, as if
// every flush attempt has been failing (no available workers, maxRuns exhausted, etc.) so
// fetchNewItems keeps appending to the buffers tick after tick with nothing draining them. All DB
// calls are mocked out (fakeBatchRepo is never actually invoked here); this only measures the
// resident memory of the buffered items themselves. Reports the total and per-item resident bytes
// via runtime.MemStats as custom metrics -- run with a fixed iteration count since this isn't a
// timing benchmark:
//
//	go test ./pkg/scheduling/v1/... -run '^$' -bench BenchmarkBatchSchedulerWorstCaseMemory -benchtime=1x
func BenchmarkBatchSchedulerWorstCaseMemory(b *testing.B) {
	const (
		numGroups           = 100  // e.g. 10 steps x 10 batch keys, one BatchScheduler per step
		maxBufferedPerGroup = 2000 // items piled up in a single group while flush is stuck
	)

	tenantId := uuid.New()
	stepId := uuid.New()

	resource := &sqlcv1.ListDistinctBatchResourcesRow{
		StepID:       stepId,
		BatchKey:     "key-0",
		BatchMaxSize: 50,
	}

	for n := 0; n < b.N; n++ {
		runtime.GC()

		var before runtime.MemStats
		runtime.ReadMemStats(&before)

		scheduler := newBatchScheduler(
			newTestSharedConfig(&fakeBatchRepo{}),
			tenantId,
			resource,
			nil,
			nil,
			func(*QueueResults) {},
		)

		for g := 0; g < numGroups; g++ {
			batchKey := fmt.Sprintf("key-%d", g)
			group := &batchGroup{batchKey: batchKey, l: new(zerolog.New(io.Discard))}

			for i := 0; i < maxBufferedPerGroup; i++ {
				group.buffer = append(group.buffer, benchBatchedQueueItem(int64(g*maxBufferedPerGroup+i), tenantId, stepId, batchKey))
			}

			scheduler.groups[batchKey] = group
		}

		runtime.GC()

		var after runtime.MemStats
		runtime.ReadMemStats(&after)

		totalItems := numGroups * maxBufferedPerGroup
		totalBytes := after.HeapAlloc - before.HeapAlloc

		b.ReportMetric(float64(totalBytes), "worst_case_bytes")
		b.ReportMetric(float64(totalBytes)/1e6, "worst_case_MB")
		b.ReportMetric(float64(totalBytes)/float64(totalItems), "bytes/item")

		runtime.KeepAlive(scheduler)
	}
}
