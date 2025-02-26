package repository

import (
	"context"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

type ListGetGroupKeyRunsOpts struct {
	Status *dbsqlc.StepRunStatus
}

type UpdateGetGroupKeyRunOpts struct {
	RequeueAfter *time.Time

	ScheduleTimeoutAt *time.Time

	Status *dbsqlc.StepRunStatus

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
	ListGetGroupKeyRunsToRequeue(ctx context.Context, tenantId string) ([]*dbsqlc.GetGroupKeyRun, error)

	ListGetGroupKeyRunsToReassign(ctx context.Context, tenantId string) ([]*dbsqlc.GetGroupKeyRun, error)

	AssignGetGroupKeyRunToWorker(ctx context.Context, tenantId, getGroupKeyRunId string) (workerId string, dispatcherId string, err error)
	AssignGetGroupKeyRunToTicker(ctx context.Context, tenantId, getGroupKeyRunId string) (tickerId string, err error)

	UpdateGetGroupKeyRun(ctx context.Context, tenantId, getGroupKeyRunId string, opts *UpdateGetGroupKeyRunOpts) (*dbsqlc.GetGroupKeyRunForEngineRow, error)

	GetGroupKeyRunForEngine(ctx context.Context, tenantId, getGroupKeyRunId string) (*dbsqlc.GetGroupKeyRunForEngineRow, error)
}
