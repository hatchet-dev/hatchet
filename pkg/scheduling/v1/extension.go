package v1

import (
	"context"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

type PostAssignInput struct {
	HasUnassignedStepRuns bool
}

type SnapshotInput struct {
	Workers map[uuid.UUID]*WorkerCp

	// WorkerSlotUtilization is the per-worker slot utilization summed across slot types.
	WorkerSlotUtilization map[uuid.UUID]*SlotUtilization

	// WorkerSlotUtilizationByType breaks WorkerSlotUtilization down by slot type.
	WorkerSlotUtilizationByType map[uuid.UUID]map[string]*SlotUtilization
}

type SlotUtilization struct {
	UtilizedSlots    int
	NonUtilizedSlots int
}

type WorkerCp struct {
	WorkerId uuid.UUID
	MaxRuns  int
	Labels   []*sqlcv1.ListManyWorkerLabelsRow
	Name     string
}

type SlotCp struct {
	WorkerId uuid.UUID
	Used     bool
}

type SchedulerExtension interface {
	SetTenants(tenants []*sqlcv1.Tenant)
	ReportSnapshot(ctx context.Context, tenantId uuid.UUID, input *SnapshotInput)
	PostAssign(tenantId uuid.UUID, input *PostAssignInput)
	CleanupTenant(tenantId uuid.UUID) error
	Cleanup() error
}

type Extensions struct {
	mu   sync.RWMutex
	exts []SchedulerExtension
}

func (e *Extensions) Add(ext SchedulerExtension) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.exts == nil {
		e.exts = make([]SchedulerExtension, 0)
	}

	e.exts = append(e.exts, ext)
}

func (e *Extensions) ReportSnapshot(ctx context.Context, tenantId uuid.UUID, input *SnapshotInput) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Extensions run in their own goroutines. `ctx` is the scheduler-lifetime
	// context (not a per-iteration one), so it's safe to hand to a fire-and-
	// forget goroutine; each gets its own child span under the snapshot trace.
	for _, ext := range e.exts {
		f := ext.ReportSnapshot
		go func() {
			spanCtx, span := telemetry.NewSpan(ctx, "report-snapshot")
			defer span.End()

			f(spanCtx, tenantId, input)
		}()
	}
}

func (e *Extensions) PostAssign(tenantId uuid.UUID, input *PostAssignInput) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, ext := range e.exts {
		f := ext.PostAssign
		go f(tenantId, input)
	}
}

func (e *Extensions) CleanupTenant(tenantId uuid.UUID) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	eg := errgroup.Group{}

	for _, ext := range e.exts {
		f := ext.CleanupTenant
		eg.Go(func() error { return f(tenantId) })
	}

	return eg.Wait()
}

func (e *Extensions) Cleanup() error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	eg := errgroup.Group{}

	for _, ext := range e.exts {
		f := ext.Cleanup
		eg.Go(f)
	}

	return eg.Wait()
}

func (e *Extensions) SetTenants(tenants []*sqlcv1.Tenant) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, ext := range e.exts {
		f := ext.SetTenants
		go f(tenants)
	}
}
