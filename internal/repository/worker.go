package repository

import (
	"time"

	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
)

type CreateWorkerOpts struct {
	// The id of the dispatcher
	DispatcherId string `validate:"required,uuid"`

	// The name of the worker
	Name string `validate:"required,hatchetName"`

	// The name of the service
	Services []string `validate:"dive,hatchetName"`

	// A list of actions this worker can run
	Actions []string `validate:"dive,actionId"`
}

type UpdateWorkerOpts struct {
	// The status of the worker
	Status *db.WorkerStatus

	// When the last worker heartbeat was
	LastHeartbeatAt *time.Time

	// A list of actions this worker can run
	Actions []string `validate:"dive,actionId"`
}

type WorkerWithStepCount struct {
	Worker       *db.WorkerModel
	StepRunCount int
}

type ListWorkersOpts struct {
	Action *string `validate:"omitempty,actionId"`
}

type WorkerRepository interface {
	// ListWorkers lists workers for the tenant
	ListWorkers(tenantId string, opts *ListWorkersOpts) ([]WorkerWithStepCount, error)

	// ListRecentWorkerStepRuns lists recent step runs for a given worker
	ListRecentWorkerStepRuns(tenantId, workerId string) ([]db.StepRunModel, error)

	// CreateNewWorker creates a new worker for a given tenant.
	CreateNewWorker(tenantId string, opts *CreateWorkerOpts) (*db.WorkerModel, error)

	// UpdateWorker updates a worker for a given tenant.
	UpdateWorker(tenantId, workerId string, opts *UpdateWorkerOpts) (*db.WorkerModel, error)

	// DeleteWorker removes the worker from the database
	DeleteWorker(tenantId, workerId string) error

	// GetWorkerById returns a worker by its id.
	GetWorkerById(workerId string) (*db.WorkerModel, error)

	// AddStepRun assigns a step run to a worker.
	AddStepRun(tenantId, workerId, stepRunId string) error
}
