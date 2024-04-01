package repository

import (
	"time"

	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
)

type ListGetGroupKeyRunsOpts struct {
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

type GetGroupKeyRunEngineRepository interface {
	// ListStepRunsToRequeue returns a list of step runs which are in a requeueable state.
	ListGetGroupKeyRunsToRequeue(tenantId string) ([]*dbsqlc.GetGroupKeyRun, error)

	ListGetGroupKeyRunsToReassign(tenantId string) ([]*dbsqlc.GetGroupKeyRun, error)

	AssignGetGroupKeyRunToWorker(tenantId, getGroupKeyRunId string) (workerId string, dispatcherId string, err error)
	AssignGetGroupKeyRunToTicker(tenantId, getGroupKeyRunId string) (tickerId string, err error)

	UpdateGetGroupKeyRun(tenantId, getGroupKeyRunId string, opts *UpdateGetGroupKeyRunOpts) (*dbsqlc.GetGroupKeyRunForEngineRow, error)

	GetGroupKeyRunForEngine(tenantId, getGroupKeyRunId string) (*dbsqlc.GetGroupKeyRunForEngineRow, error)
}
