//go:build !e2e && !load && !rampup && !integration

package v1

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"

	repo "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// BenchmarkScheduler_ConcurrentAssignReplenishSnapshot measures assignment
// throughput while replenish and snapshot churn in the background — the
// contention profile of a busy tenant. Each iteration assigns one 64-item batch
// via tryAssign, the same entrypoint the queuers use, with each parallel worker
// driving its own action.
func BenchmarkScheduler_ConcurrentAssignReplenishSnapshot(b *testing.B) {
	const batchSize = 64

	shape := inventoryShape{
		Name:           "concurrent_dense",
		Workers:        50,
		Actions:        200,
		SlotsPerWorker: 20,
		Topology:       topologyDense,
	}

	f := newInventoryFixture(shape)
	b.Cleanup(f.stop)

	if err := f.scheduler.replenish(context.Background(), true); err != nil {
		b.Fatal(err)
	}
	f.measureInventory()

	// background churn: replenish and snapshot loops run as fast as they can
	churnCtx, stopChurn := context.WithCancel(context.Background())
	var churnWg sync.WaitGroup

	churnWg.Add(2)
	go func() {
		defer churnWg.Done()
		for churnCtx.Err() == nil {
			_ = f.scheduler.replenish(churnCtx, true)
		}
	}()
	go func() {
		defer churnWg.Done()
		for churnCtx.Err() == nil {
			_, _ = f.scheduler.getSnapshotInput(churnCtx)
		}
	}()

	var actionCounter atomic.Int64

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		// each parallel worker drives its own action, like per-action goroutines
		// fanned out by the queuers
		actionId := f.actionIds[int(actionCounter.Add(1))%len(f.actionIds)]

		qis := make([]*sqlcv1.V1QueueItem, batchSize)
		stepRequests := make(map[uuid.UUID]map[string]int32, batchSize)
		for i := range qis {
			qi := testQI(f.tenantId, actionId, int64(i+1))
			qis[i] = qi
			stepRequests[qi.StepID] = map[string]int32{repo.SlotTypeDefault: 1}
		}

		ackIds := make([]int, 0, batchSize)

		for pb.Next() {
			ch := f.scheduler.tryAssign(
				context.Background(),
				qis,
				map[uuid.UUID][]*sqlcv1.GetDesiredLabelsRow{},
				stepRequests,
				nil,
				nil,
				nil,
			)

			// ack assignments the way a queuer flush would, so unacked slots
			// don't accumulate across iterations and skew replenish churn
			ackIds = ackIds[:0]
			for r := range ch {
				for _, a := range r.assigned {
					ackIds = append(ackIds, a.AckId)
				}
			}
			f.scheduler.ack(ackIds)
		}
	})
	b.StopTimer()

	stopChurn()
	churnWg.Wait()

	b.ReportMetric(float64(batchSize), "batch_size")
	reportShapeMetrics(b, f)
}
