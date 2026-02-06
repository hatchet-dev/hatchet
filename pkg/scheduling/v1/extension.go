package v1

import (
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type PostAssignInput struct {
	HasUnassignedStepRuns bool
}

type SnapshotInput struct {
	Workers               map[uuid.UUID]*WorkerCp
	WorkerSlotUtilization map[uuid.UUID]*SlotUtilization
}

type SlotUtilization struct {
	UtilizedSlots    int
	NonUtilizedSlots int
}

type WorkerCp struct {
	Name     string
	Labels   []*sqlcv1.ListManyWorkerLabelsRow
	MaxRuns  int
	WorkerId uuid.UUID
}

type SlotCp struct {
	WorkerId uuid.UUID
	Used     bool
}

type SchedulerExtension interface {
	SetTenants(tenants []*sqlcv1.Tenant)
	ReportSnapshot(tenantId uuid.UUID, input *SnapshotInput)
	PostAssign(tenantId uuid.UUID, input *PostAssignInput)
	Cleanup() error
}

type Extensions struct {
	exts []SchedulerExtension
	mu   sync.RWMutex
}

func (e *Extensions) Add(ext SchedulerExtension) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.exts == nil {
		e.exts = make([]SchedulerExtension, 0)
	}

	e.exts = append(e.exts, ext)
}

func (e *Extensions) ReportSnapshot(tenantId uuid.UUID, input *SnapshotInput) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, ext := range e.exts {
		f := ext.ReportSnapshot
		go f(tenantId, input)
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
