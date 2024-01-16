package repository

import (
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/steebchen/prisma-client-go/runtime/types"
)

type ListAllStepRunsOpts struct {
	TickerId *string

	NoTickerId *bool

	Status *db.StepRunStatus
}

type ListStepRunsOpts struct {
	Requeuable *bool

	JobRunId *string

	Status *db.StepRunStatus
}

type UpdateStepRunOpts struct {
	RequeueAfter *time.Time

	ScheduleTimeoutAt *time.Time

	Status *db.StepRunStatus

	StartedAt *time.Time

	FailedAt *time.Time

	FinishedAt *time.Time

	CancelledAt *time.Time

	CancelledReason *string

	Error *string

	Input *types.JSON

	Output *types.JSON
}

func StepRunStatusPtr(status db.StepRunStatus) *db.StepRunStatus {
	return &status
}

var StepRunIsNotPendingErr error = fmt.Errorf("step run is not pending")

type StepRunRepository interface {
	// ListAllStepRuns returns a list of all step runs which match the given options.
	ListAllStepRuns(opts *ListAllStepRunsOpts) ([]db.StepRunModel, error)

	// ListStepRuns returns a list of step runs for a tenant which match the given options.
	ListStepRuns(tenantId string, opts *ListStepRunsOpts) ([]db.StepRunModel, error)

	UpdateStepRun(tenantId, stepRunId string, opts *UpdateStepRunOpts) (*db.StepRunModel, error)

	GetStepRunById(tenantId, stepRunId string) (*db.StepRunModel, error)

	// QueueStepRun is like UpdateStepRun, except that it will only update the step run if it is in
	// a pending state.
	QueueStepRun(tenantId, stepRunId string, opts *UpdateStepRunOpts) (*db.StepRunModel, error)

	CancelPendingStepRuns(tenantId, jobRunId, reason string) error

	ListStartableStepRuns(tenantId, jobRunId, parentStepRunId string) ([]*dbsqlc.StepRun, error)
}
