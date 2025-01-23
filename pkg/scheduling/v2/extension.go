package v2

import (
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/v2/sqlcv2"
)

type PostScheduleInput struct {
	Workers map[string]*WorkerCp

	Slots []*SlotCp

	Unassigned []*sqlcv2.V2QueueItem

	ActionsToSlots map[string][]*SlotCp
}

type WorkerCp struct {
	WorkerId string
	MaxRuns  int
	Labels   []*sqlcv2.ListManyWorkerLabelsRow
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
