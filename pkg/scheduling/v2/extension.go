package v2

import (
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
)

type PostScheduleInput struct {
	Workers               map[string]*WorkerCp
	HasUnassignedStepRuns bool
	WorkerSlotUtilization map[string]*SlotUtilization
}

type SlotUtilization struct {
	UtilizedSlots    int
	NonUtilizedSlots int
}

type WorkerCp struct {
	WorkerId string
	MaxRuns  int
	Labels   []*dbsqlc.ListManyWorkerLabelsRow
}

type SlotCp struct {
	WorkerId string
	Used     bool
}

type SchedulerExtension interface {
	SetTenants(tenants []*dbsqlc.Tenant)
	PostSchedule(tenantId string, input *PostScheduleInput)
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

func (e *Extensions) PostSchedule(tenantId string, input *PostScheduleInput) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, ext := range e.exts {
		f := ext.PostSchedule
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

func (e *Extensions) SetTenants(tenants []*dbsqlc.Tenant) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, ext := range e.exts {
		f := ext.SetTenants
		go f(tenants)
	}
}
