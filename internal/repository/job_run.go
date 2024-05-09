package repository

import (
	"context"

	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
)

type UpdateJobRunLookupDataOpts struct {
	FieldPath []string
	Data      []byte
}

type ListAllJobRunsOpts struct {
	TickerId *string

	NoTickerId *bool

	Status *db.JobRunStatus
}

func JobRunStatusPtr(status db.JobRunStatus) *db.JobRunStatus {
	return &status
}

type JobRunAPIRepository interface {
	// SetJobRunStatusRunning resets the status of a job run to a RUNNING status. This is useful if a step
	// run is being manually replayed, but shouldn't be used by most callers.
	SetJobRunStatusRunning(tenantId, jobRunId string) error
}

type JobRunEngineRepository interface {
	// SetJobRunStatusRunning resets the status of a job run to a RUNNING status. This is useful if a step
	// run is being manually replayed, but shouldn't be used by most callers.
	SetJobRunStatusRunning(ctx context.Context, tenantId, jobRunId string) error

	ListJobRunsForWorkflowRun(ctx context.Context, tenantId, workflowRunId string) ([]*dbsqlc.ListJobRunsForWorkflowRunRow, error)

	GetJobRunByWorkflowRunIdAndJobId(ctx context.Context, tenantId, workflowRunId, jobId string) (*dbsqlc.GetJobRunByWorkflowRunIdAndJobIdRow, error)
}
