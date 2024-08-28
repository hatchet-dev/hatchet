package repository

import (
	"context"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"

	"github.com/jackc/pgx/v5/pgtype"
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

	// (optional) Webhook Id associated with the worker (if any)
	WebhookId *string `validate:"omitempty,uuid"`
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

type UpsertWorkerLabelOpts struct {
	Key      string
	IntValue *int32
	StrValue *string
}

type ApiUpdateWorkerOpts struct {
	IsPaused *bool
}

type WorkerAPIRepository interface {
	// ListWorkers lists workers for the tenant
	ListWorkers(tenantId string, opts *ListWorkersOpts) ([]*dbsqlc.ListWorkersWithStepCountRow, error)

	// ListRecentWorkerStepRuns lists recent step runs for a given worker
	ListWorkerState(tenantId, workerId string, failed bool) ([]*dbsqlc.ListSemaphoreSlotsWithStateForWorkerRow, []*dbsqlc.ListRecentStepRunsForWorkerRow, error)

	// GetWorkerActionsByWorkerId returns a list of actions for a worker
	GetWorkerActionsByWorkerId(tenantid, workerId string) ([]pgtype.Text, error)

	// GetWorkerById returns a worker by its id.
	GetWorkerById(workerId string) (*dbsqlc.GetWorkerByIdRow, error)

	// ListWorkerLabels returns a list of labels config for a worker
	ListWorkerLabels(tenantId, workerId string) ([]*dbsqlc.ListWorkerLabelsRow, error)

	// UpdateWorker updates a worker for a given tenant.
	UpdateWorker(tenantId string, workerId string, opts ApiUpdateWorkerOpts) (*dbsqlc.Worker, error)
}

type WorkerEngineRepository interface {
	// CreateNewWorker creates a new worker for a given tenant.
	CreateNewWorker(ctx context.Context, tenantId string, opts *CreateWorkerOpts) (*dbsqlc.Worker, error)

	// UpdateWorker updates a worker for a given tenant.
	UpdateWorker(ctx context.Context, tenantId, workerId string, opts *UpdateWorkerOpts) (*dbsqlc.Worker, error)

	// UpdateWorker updates a worker in the repository.
	// It will only update the worker if there is no lock on the worker, else it will skip.
	UpdateWorkerHeartbeat(ctx context.Context, tenantId, workerId string, lastHeartbeatAt time.Time) error

	// DeleteWorker removes the worker from the database
	DeleteWorker(ctx context.Context, tenantId, workerId string) error

	// UpdateWorkersByWebhookId removes the worker from the database
	UpdateWorkersByWebhookId(ctx context.Context, opts dbsqlc.UpdateWorkersByWebhookIdParams) error

	GetWorkerForEngine(ctx context.Context, tenantId, workerId string) (*dbsqlc.GetWorkerForEngineRow, error)

	ResolveWorkerSemaphoreSlots(ctx context.Context, tenantId pgtype.UUID) (*dbsqlc.ResolveWorkerSemaphoreSlotsRow, error)

	UpdateWorkerActiveStatus(ctx context.Context, tenantId, workerId string, isActive bool, timestamp time.Time) (*dbsqlc.Worker, error)

	UpsertWorkerLabels(ctx context.Context, workerId pgtype.UUID, opts []UpsertWorkerLabelOpts) ([]*dbsqlc.WorkerLabel, error)
}
