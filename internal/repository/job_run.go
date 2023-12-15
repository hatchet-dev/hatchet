package repository

import (
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/steebchen/prisma-client-go/runtime/types"
)

type UpdateJobRunOpts struct {
	Status *db.JobRunStatus
}

type UpdateJobRunLookupDataOpts struct {
	LookupData *types.JSON
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

	UpdateJobRun(tenantId, jobRunId string, opts *UpdateJobRunOpts) (*db.JobRunModel, error)

	GetJobRunLookupData(tenantId, jobRunId string) (*db.JobRunLookupDataModel, error)

	UpdateJobRunLookupData(tenantId, jobRunId string, opts *UpdateJobRunLookupDataOpts) (*db.JobRunLookupDataModel, error)
}
