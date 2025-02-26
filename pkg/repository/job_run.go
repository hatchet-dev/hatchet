package repository

import (
	"context"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"

	"github.com/jackc/pgx/v5/pgtype"
)

type UpdateJobRunLookupDataOpts struct {
	FieldPath []string
	Data      []byte
}

type ListAllJobRunsOpts struct {
	TickerId *string

	NoTickerId *bool

	Status *dbsqlc.JobRunStatus
}

func JobRunStatusPtr(status dbsqlc.JobRunStatus) *dbsqlc.JobRunStatus {
	return &status
}

type JobRunAPIRepository interface {
	RegisterWorkflowRunRunningCallback(callback TenantScopedCallback[pgtype.UUID])

	// SetJobRunStatusRunning resets the status of a job run to a RUNNING status. This is useful if a step
	// run is being manually replayed, but shouldn't be used by most callers.
	SetJobRunStatusRunning(tenantId, jobRunId string) error

	ListJobRunByWorkflowRunId(ctx context.Context, tenantId, WorkflowRunId string) ([]*dbsqlc.ListJobRunsForWorkflowRunFullRow, error)
}

type JobRunEngineRepository interface {
	RegisterWorkflowRunRunningCallback(callback TenantScopedCallback[pgtype.UUID])

	// SetJobRunStatusRunning resets the status of a job run to a RUNNING status. This is useful if a step
	// run is being manually replayed, but shouldn't be used by most callers.
	SetJobRunStatusRunning(ctx context.Context, tenantId, jobRunId string) error

	ListJobRunsForWorkflowRun(ctx context.Context, tenantId, workflowRunId string) ([]*dbsqlc.ListJobRunsForWorkflowRunRow, error)

	GetJobRunByWorkflowRunIdAndJobId(ctx context.Context, tenantId, workflowRunId, jobId string) (*dbsqlc.GetJobRunByWorkflowRunIdAndJobIdRow, error)

	GetJobRunsByWorkflowRunId(ctx context.Context, tenantId, workflowRunId string) ([]*dbsqlc.GetJobRunsByWorkflowRunIdRow, error)

	ClearJobRunPayloadData(ctx context.Context, tenantId string) (bool, error)

	StartJobRun(ctx context.Context, tenantId, jobId string) ([]*dbsqlc.GetStepRunForEngineRow, error)
}
