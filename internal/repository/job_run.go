package repository

import (
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

type JobRunRepository interface {
	ListAllJobRuns(opts *ListAllJobRunsOpts) ([]db.JobRunModel, error)

	GetJobRunById(tenantId, jobRunId string) (*db.JobRunModel, error)

	// SetJobRunStatusRunning resets the status of a job run to a RUNNING status. This is useful if a step
	// run is being manually replayed, but shouldn't be used by most callers.
	SetJobRunStatusRunning(tenantId, jobRunId string) error

	GetJobRunLookupData(tenantId, jobRunId string) (*db.JobRunLookupDataModel, error)

	UpdateJobRunLookupData(tenantId, jobRunId string, opts *UpdateJobRunLookupDataOpts) error
}
