package repository

import (
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
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
	SetJobRunStatusRunning(tenantId, jobRunId string) error

	ListJobRunsForWorkflowRun(tenantId, workflowRunId string) ([]pgtype.UUID, error)
}
