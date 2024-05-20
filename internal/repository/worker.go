package repository

import (
	"context"
	"time"

	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
)

type CreateWorkerOpts struct {
	// The id of the dispatcher
	DispatcherId string `validate:"required,uuid"`

	// The maximum number of runs this worker can run at a time
	MaxRuns *int `validate:"omitempty,gte=1"`

	// The name of the worker
	Name string `validate:"required,hatchetName"`

	// The name of the service
	Services []string `validate:"dive,hatchetName"`

	// A list of actions this worker can run
	Actions []string `validate:"dive,actionId"`
}

type UpdateWorkerOpts struct {
	// The id of the dispatcher
	DispatcherId *string `validate:"omitempty,uuid"`

	// When the last worker heartbeat was
	LastHeartbeatAt *time.Time

	// If the worker is active and accepting new runs
	IsActive *bool

	// A list of actions this worker can run
	Actions []string `validate:"dive,actionId"`
}

type WorkerWithStepCount struct {
	Worker       *db.WorkerModel
	StepRunCount int
}

type ListWorkersOpts struct {
	Action *string `validate:"omitempty,actionId"`

	LastHeartbeatAfter *time.Time

	Assignable *bool
}

type WorkerAPIRepository interface {
	// ListWorkers lists workers for the tenant
	ListWorkers(tenantId string, opts *ListWorkersOpts) ([]*dbsqlc.ListWorkersWithStepCountRow, error)

	// ListRecentWorkerStepRuns lists recent step runs for a given worker
	ListRecentWorkerStepRuns(tenantId, workerId string) ([]db.StepRunModel, error)

	// GetWorkerById returns a worker by its id.
	GetWorkerById(workerId string) (*db.WorkerModel, error)
}

type WorkerEngineRepository interface {
	// CreateNewWorker creates a new worker for a given tenant.
	CreateNewWorker(ctx context.Context, tenantId string, opts *CreateWorkerOpts) (*dbsqlc.Worker, error)

	// UpdateWorker updates a worker for a given tenant.
	UpdateWorker(ctx context.Context, tenantId, workerId string, opts *UpdateWorkerOpts) (*dbsqlc.Worker, error)

	// DeleteWorker removes the worker from the database
	DeleteWorker(ctx context.Context, tenantId, workerId string) error

	GetWorkerForEngine(ctx context.Context, tenantId, workerId string) (*dbsqlc.GetWorkerForEngineRow, error)

	ResolveWorkerSemaphoreSlots(ctx context.Context) (int64, error)
}
