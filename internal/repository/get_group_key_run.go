package repository

import (
	"time"

	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
)

type ListGetGroupKeyRunsOpts struct {
	Requeuable *bool

	Status *db.StepRunStatus
}

type UpdateGetGroupKeyRunOpts struct {
	RequeueAfter *time.Time

	ScheduleTimeoutAt *time.Time

	Status *db.StepRunStatus

	StartedAt *time.Time

	FailedAt *time.Time

	FinishedAt *time.Time

	CancelledAt *time.Time

	CancelledReason *string

	Error *string

	Output *string
}

type GetGroupKeyRunRepository interface {
	// ListGetGroupKeyRuns returns a list of get group key runs for a tenant which match the given options.
	ListGetGroupKeyRuns(tenantId string, opts *ListGetGroupKeyRunsOpts) ([]db.GetGroupKeyRunModel, error)

	UpdateGetGroupKeyRun(tenantId, getGroupKeyRunId string, opts *UpdateGetGroupKeyRunOpts) (*db.GetGroupKeyRunModel, error)

	GetGroupKeyRunById(tenantId, getGroupKeyRunId string) (*db.GetGroupKeyRunModel, error)
}
