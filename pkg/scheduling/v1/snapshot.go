package v1

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/randomticker"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

// This file contains the utilization-snapshot path: a periodic, read-only view
// of the tenant's slot pools reported to the registered extensions (e.g. the
// Prometheus extension). It runs on the same run loop as scheduling, so
// snapshots are always internally consistent.

func (s *Scheduler) loopSnapshot(ctx context.Context) {
	ticker := randomticker.NewRandomTicker(1000*time.Millisecond, 1500*time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.snapshot(ctx)
		}
	}
}

// snapshot builds a point-in-time view of the tenant's slot utilization and
// reports it to the registered extensions.
func (s *Scheduler) snapshot(ctx context.Context) {
	ctx, span := telemetry.NewSpan(ctx, "snapshot")
	defer span.End()

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "tenant.id", Value: s.tenantId.String()},
	)

	in, ok := s.getSnapshotInput(ctx)

	if !ok {
		telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "snapshot.skipped", Value: true})
		return
	}

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "snapshot.worker_count", Value: len(in.Workers)})

	s.exts.ReportSnapshot(ctx, s.tenantId, in)
}

func (s *Scheduler) getSnapshotInput(ctx context.Context) (*SnapshotInput, bool) {
	ctx, span := telemetry.NewSpan(ctx, "get-snapshot-input")
	defer span.End()

	var res *SnapshotInput

	if ok := s.do(ctx, func() {
		res = s.buildSnapshotInput()
	}); !ok {
		return nil, false
	}

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "snapshot.worker_count", Value: len(res.Workers)},
	)

	return res, true
}

// buildSnapshotInput runs on the run loop.
func (s *Scheduler) buildSnapshotInput() *SnapshotInput {
	res := &SnapshotInput{
		Workers:                     make(map[uuid.UUID]*WorkerCp, len(s.workers)),
		WorkerSlotUtilization:       make(map[uuid.UUID]*SlotUtilization, len(s.workers)),
		WorkerSlotUtilizationByType: make(map[uuid.UUID]map[string]*SlotUtilization, len(s.workers)),
	}

	for workerId, worker := range s.workers {
		totalSlots := 0

		for _, units := range worker.TotalSlotsByType {
			totalSlots += units
		}

		res.Workers[workerId] = &WorkerCp{
			WorkerId: workerId,
			Labels:   worker.Labels,
			Name:     worker.Name,
			MaxRuns:  totalSlots,
		}
	}

	utilizationByType := make(map[uuid.UUID]map[string]*SlotUtilization, len(s.workers))

	for workerId := range s.workers {
		utilizationByType[workerId] = make(map[string]*SlotUtilization)
	}

	for key, pool := range s.pools {
		byType, active := utilizationByType[key.workerId]
		if !active {
			continue
		}

		utilization := byType[key.slotType]
		if utilization == nil {
			utilization = &SlotUtilization{}
			byType[key.slotType] = utilization
		}

		utilization.UtilizedSlots += len(pool.slots) - len(pool.free)
		utilization.NonUtilizedSlots += len(pool.free)
	}

	// prune warm state for workers which are no longer registered
	for workerId := range s.warmedSlotTypes {
		if _, ok := s.workers[workerId]; !ok {
			delete(s.warmedSlotTypes, workerId)
		}
	}

	// The in-memory pool only holds slots which have not been assigned (plus assigned slots
	// which are not yet flushed to the database), so the used counts walked above miss any
	// slot consumed by a running task. Derive the true used count per slot type from the
	// worker's slot capacity instead: everything that is not free is in use.
	for workerId, byType := range utilizationByType {
		var capacities map[string]int

		if worker, ok := s.workers[workerId]; ok {
			capacities = worker.TotalSlotsByType
		}

		warmed := s.warmedSlotTypes[workerId]

		for slotType, utilization := range byType {
			if utilization.UtilizedSlots+utilization.NonUtilizedSlots > 0 {
				if warmed == nil {
					warmed = make(map[string]struct{})
					s.warmedSlotTypes[workerId] = warmed
				}

				warmed[slotType] = struct{}{}
			}
		}

		// slot types with capacity but no walked slots still get reported: once the
		// type has warmed up, an empty in-memory pool means all of its slots are in use
		for slotType := range capacities {
			if _, ok := byType[slotType]; !ok {
				byType[slotType] = &SlotUtilization{}
			}
		}

		aggregate := &SlotUtilization{}

		for slotType, utilization := range byType {
			_, isWarmed := warmed[slotType]

			// Only derive from capacity once the slot type has had slots in the pool:
			// a never-replenished worker would otherwise report full utilization
			// between registration and its first replenish. Un-warmed types report
			// zero slots, which extensions treat as a transient state.
			if capacity := capacities[slotType]; capacity > 0 && isWarmed {
				used := capacity - utilization.NonUtilizedSlots
				if used < 0 {
					used = 0
				}

				utilization.UtilizedSlots = used
			}
			// no capacity known for this slot type; fall back to the walked counts

			aggregate.UtilizedSlots += utilization.UtilizedSlots
			aggregate.NonUtilizedSlots += utilization.NonUtilizedSlots
		}

		res.WorkerSlotUtilizationByType[workerId] = byType
		res.WorkerSlotUtilization[workerId] = aggregate
	}

	return res
}
