//go:build !e2e && !load && !rampup && !integration

package v1

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	repo "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// inventoryTopology controls how actions attach to workers. This is the main
// axis that changes fan-out in the current action-centric slot inventory:
// each worker's slot objects are appended into every action that worker owns.
type inventoryTopology string

const (
	// topologyDense: every worker registers every action.
	// Unique slots = W*S; action-slot refs ≈ W*S*A.
	topologyDense inventoryTopology = "dense"

	// topologySparse: each worker registers ActionsPerWorker actions, chosen
	// round-robin from the action pool. Fan-out ≈ ActionsPerWorker.
	topologySparse inventoryTopology = "sparse"

	// topologyPartitioned: workers and actions split into Partitions disjoint
	// groups; within a group, every worker has every group action (dense local).
	topologyPartitioned inventoryTopology = "partitioned"
)

// inventoryShape is a synthetic (workers × slots × actions × topology) fixture
// used to baseline the current scheduler inventory before trying alternate shapes.
type inventoryShape struct {
	Name             string
	Workers          int
	Actions          int
	SlotsPerWorker   int
	Topology         inventoryTopology
	ActionsPerWorker int // sparse only
	Partitions       int // partitioned only
}

type inventoryFixture struct {
	shape          inventoryShape
	tenantId       uuid.UUID
	workerIds      []uuid.UUID
	activeWorkers  []*repo.ListActiveWorkersResult
	actionIds      []string
	actionRows     []*sqlcv1.ListActionsForWorkersRow
	scheduler      *Scheduler
	uniqueSlots    int
	actionSlotRefs int
	fanout         float64
}

func baselineShapes() []inventoryShape {
	return []inventoryShape{
		{
			Name:           "small_dense",
			Workers:        10,
			Actions:        20,
			SlotsPerWorker: 4,
			Topology:       topologyDense,
		},
		// High action fan-out: many shared actions, rebuild rewrites large action inventories.
		{
			Name:           "dense_high_action_fanout",
			Workers:        50,
			Actions:        1724,
			SlotsPerWorker: 20,
			Topology:       topologyDense,
		},
		// High slot capacity: fewer actions, large per-worker pools → expensive
		// activeCount scans across duplicated action slices.
		{
			Name:           "dense_high_slot_capacity",
			Workers:        40,
			Actions:        30,
			SlotsPerWorker: 1400,
			Topology:       topologyDense,
		},
		{
			Name:             "sparse_low_fanout",
			Workers:          200,
			Actions:          2000,
			SlotsPerWorker:   20,
			Topology:         topologySparse,
			ActionsPerWorker: 5,
		},
		{
			Name:           "partitioned_20",
			Workers:        200,
			Actions:        2000,
			SlotsPerWorker: 20,
			Topology:       topologyPartitioned,
			Partitions:     20,
		},
	}
}

func buildActionRows(shape inventoryShape, workerIds []uuid.UUID, actionIds []string) []*sqlcv1.ListActionsForWorkersRow {
	switch shape.Topology {
	case topologyDense:
		rows := make([]*sqlcv1.ListActionsForWorkersRow, 0, shape.Workers*shape.Actions)
		for _, wid := range workerIds {
			for _, aid := range actionIds {
				rows = append(rows, &sqlcv1.ListActionsForWorkersRow{
					WorkerId: wid,
					ActionId: pgtype.Text{String: aid, Valid: true},
				})
			}
		}
		return rows

	case topologySparse:
		per := shape.ActionsPerWorker
		if per <= 0 {
			per = 1
		}
		rows := make([]*sqlcv1.ListActionsForWorkersRow, 0, shape.Workers*per)
		for wi, wid := range workerIds {
			for j := 0; j < per; j++ {
				aid := actionIds[(wi*per+j)%len(actionIds)]
				rows = append(rows, &sqlcv1.ListActionsForWorkersRow{
					WorkerId: wid,
					ActionId: pgtype.Text{String: aid, Valid: true},
				})
			}
		}
		return rows

	case topologyPartitioned:
		parts := shape.Partitions
		if parts <= 0 {
			parts = 1
		}
		rows := make([]*sqlcv1.ListActionsForWorkersRow, 0)
		for wi, wid := range workerIds {
			part := wi % parts
			for ai, aid := range actionIds {
				if ai%parts != part {
					continue
				}
				rows = append(rows, &sqlcv1.ListActionsForWorkersRow{
					WorkerId: wid,
					ActionId: pgtype.Text{String: aid, Valid: true},
				})
			}
		}
		return rows

	default:
		panic(fmt.Sprintf("unknown topology %q", shape.Topology))
	}
}

func newShapeScheduler(tenantId uuid.UUID, ar repo.AssignmentRepository) *Scheduler {
	l := zerolog.Nop()
	sr := &mockSchedulerRepo{assignment: ar}
	cf := &sharedConfig{repo: sr, l: &l}
	return newScheduler(cf, tenantId, nil, &Extensions{})
}

func newInventoryFixture(shape inventoryShape) *inventoryFixture {
	tenantId := uuid.New()

	workerIds := make([]uuid.UUID, shape.Workers)
	activeWorkers := make([]*repo.ListActiveWorkersResult, shape.Workers)
	for i := range workerIds {
		workerIds[i] = uuid.New()
		activeWorkers[i] = testWorker(workerIds[i])
	}

	actionIds := make([]string, shape.Actions)
	for i := range actionIds {
		actionIds[i] = fmt.Sprintf("action-%05d", i)
	}

	actionRows := buildActionRows(shape, workerIds, actionIds)
	slotsPerWorker := int32(shape.SlotsPerWorker)

	ar := &mockAssignmentRepo{
		listActionsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, ids []uuid.UUID) ([]*sqlcv1.ListActionsForWorkersRow, error) {
			return actionRows, nil
		},
		listAvailableSlotsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, params sqlcv1.ListAvailableSlotsForWorkersParams) ([]*sqlcv1.ListAvailableSlotsForWorkersRow, error) {
			rows := make([]*sqlcv1.ListAvailableSlotsForWorkersRow, 0, len(params.Workerids))
			for _, wid := range params.Workerids {
				rows = append(rows, &sqlcv1.ListAvailableSlotsForWorkersRow{
					ID:             wid,
					AvailableSlots: slotsPerWorker,
				})
			}
			return rows, nil
		},
	}

	s := newShapeScheduler(tenantId, ar)
	s.setWorkers(activeWorkers)

	return &inventoryFixture{
		shape:         shape,
		tenantId:      tenantId,
		workerIds:     workerIds,
		activeWorkers: activeWorkers,
		actionIds:     actionIds,
		actionRows:    actionRows,
		scheduler:     s,
	}
}

func (f *inventoryFixture) measureInventory() {
	unique := make(map[*slot]struct{})
	refs := 0
	for _, a := range f.scheduler.actions {
		refs += len(a.slots)
		for _, sl := range a.slots {
			unique[sl] = struct{}{}
		}
	}
	f.uniqueSlots = len(unique)
	f.actionSlotRefs = refs
	if f.uniqueSlots > 0 {
		f.fanout = float64(f.actionSlotRefs) / float64(f.uniqueSlots)
	}
}

func (f *inventoryFixture) scanActiveCounts() (total int) {
	for _, a := range f.scheduler.actions {
		total += a.activeCount()
	}
	return total
}

func (f *inventoryFixture) firstAssignableAction() string {
	for _, aid := range f.actionIds {
		if a := f.scheduler.actions[aid]; a != nil && len(a.slots) > 0 {
			return aid
		}
	}
	return ""
}

// reportShapeMetrics attaches inventory columns to the go-benchmarks / benchstat
// table. Must be called after the timed loop: ResetTimer clears custom metrics
// (Go 1.24+).
func reportShapeMetrics(b *testing.B, f *inventoryFixture) {
	b.Helper()
	b.ReportMetric(float64(f.shape.Workers), "workers")
	b.ReportMetric(float64(f.shape.Actions), "actions")
	b.ReportMetric(float64(f.shape.SlotsPerWorker), "slots_per_worker")
	b.ReportMetric(float64(len(f.actionRows)), "action_rows")
	b.ReportMetric(float64(f.uniqueSlots), "unique_slots")
	b.ReportMetric(float64(f.actionSlotRefs), "action_slot_refs")
	b.ReportMetric(f.fanout, "fanout")
}

// TestScheduler_InventoryShape_DenseFanout locks in the current action-centric
// duplication: under dense topology, each unique slot is referenced once per action.
func TestScheduler_InventoryShape_DenseFanout(t *testing.T) {
	f := newInventoryFixture(inventoryShape{
		Name:           "small_dense",
		Workers:        10,
		Actions:        20,
		SlotsPerWorker: 4,
		Topology:       topologyDense,
	})
	require.NoError(t, f.scheduler.replenish(context.Background(), true))
	f.measureInventory()

	require.Equal(t, f.shape.Workers*f.shape.SlotsPerWorker, f.uniqueSlots)
	require.Equal(t, f.uniqueSlots*f.shape.Actions, f.actionSlotRefs)
	require.InDelta(t, float64(f.shape.Actions), f.fanout, 0.01)
}

func BenchmarkScheduler_InventoryShape_ReplenishMust(b *testing.B) {
	for _, shape := range baselineShapes() {
		shape := shape
		b.Run(shape.Name, func(b *testing.B) {
			f := newInventoryFixture(shape)
			// Warm once so subsequent iterations measure rebuild, not first insert.
			if err := f.scheduler.replenish(context.Background(), true); err != nil {
				b.Fatal(err)
			}
			f.measureInventory()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := f.scheduler.replenish(context.Background(), true); err != nil {
					b.Fatal(err)
				}
			}
			b.StopTimer()
			reportShapeMetrics(b, f)
		})
	}
}

func BenchmarkScheduler_InventoryShape_ActiveCountScan(b *testing.B) {
	for _, shape := range baselineShapes() {
		shape := shape
		b.Run(shape.Name, func(b *testing.B) {
			f := newInventoryFixture(shape)
			if err := f.scheduler.replenish(context.Background(), true); err != nil {
				b.Fatal(err)
			}
			f.measureInventory()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = f.scanActiveCounts()
			}
			b.StopTimer()
			reportShapeMetrics(b, f)
		})
	}
}

func BenchmarkScheduler_InventoryShape_TryAssignBatch(b *testing.B) {
	const batchSize = 64

	for _, shape := range baselineShapes() {
		shape := shape
		b.Run(shape.Name, func(b *testing.B) {
			f := newInventoryFixture(shape)
			if err := f.scheduler.replenish(context.Background(), true); err != nil {
				b.Fatal(err)
			}
			f.measureInventory()

			assignAction := f.firstAssignableAction()
			if assignAction == "" {
				b.Fatal("no assignable action after replenish")
			}

			qis := make([]*sqlcv1.V1QueueItem, batchSize)
			stepRequests := make(map[uuid.UUID]map[string]int32, batchSize)
			for i := range qis {
				qi := testQI(f.tenantId, assignAction, int64(i+1))
				qis[i] = qi
				stepRequests[qi.StepID] = map[string]int32{repo.SlotTypeDefault: 1}
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Replenish between iterations so the batch always has capacity;
				// exclude that cost from ns/op.
				b.StopTimer()
				if err := f.scheduler.replenish(context.Background(), true); err != nil {
					b.Fatal(err)
				}
				b.StartTimer()

				if _, _, err := f.scheduler.tryAssignBatch(
					context.Background(),
					assignAction,
					qis,
					0,
					map[uuid.UUID][]*sqlcv1.GetDesiredLabelsRow{},
					stepRequests,
					nil,
					nil,
				); err != nil {
					b.Fatal(err)
				}
			}
			b.StopTimer()
			b.ReportMetric(float64(batchSize), "batch_size")
			reportShapeMetrics(b, f)
		})
	}
}
