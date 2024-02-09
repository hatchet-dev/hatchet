package repository

import (
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
)

type ListAllStepRunsOpts struct {
	TickerId *string

	NoTickerId *bool

	Status *db.StepRunStatus
}

type ListStepRunsOpts struct {
	Requeuable *bool

	JobRunId *string

	WorkflowRunId *string

	Status *db.StepRunStatus
}

type UpdateStepRunOpts struct {
	IsRerun bool

	RequeueAfter *time.Time

	ScheduleTimeoutAt *time.Time

	Status *db.StepRunStatus

	StartedAt *time.Time

	FailedAt *time.Time

	FinishedAt *time.Time

	CancelledAt *time.Time

	CancelledReason *string

	Error *string

	Input []byte

	Output []byte
}

type UpdateStepRunOverridesDataOpts struct {
	OverrideKey string
	Data        []byte
}

func StepRunStatusPtr(status db.StepRunStatus) *db.StepRunStatus {
	return &status
}

var ErrStepRunIsNotPending = fmt.Errorf("step run is not pending")

type StepRunRepository interface {
	// ListAllStepRuns returns a list of all step runs which match the given options.
	ListAllStepRuns(opts *ListAllStepRunsOpts) ([]db.StepRunModel, error)

	// ListStepRuns returns a list of step runs for a tenant which match the given options.
	ListStepRuns(tenantId string, opts *ListStepRunsOpts) ([]db.StepRunModel, error)

	UpdateStepRun(tenantId, stepRunId string, opts *UpdateStepRunOpts) (*db.StepRunModel, error)

	// UpdateStepRunOverridesData updates the overrides data field in the input for a step run. This returns the input
	// bytes.
	UpdateStepRunOverridesData(tenantId, stepRunId string, opts *UpdateStepRunOverridesDataOpts) ([]byte, error)

	UpdateStepRunInputSchema(tenantId, stepRunId string, schema []byte) ([]byte, error)

	GetStepRunById(tenantId, stepRunId string) (*db.StepRunModel, error)

	// QueueStepRun is like UpdateStepRun, except that it will only update the step run if it is in
	// a pending state.
	QueueStepRun(tenantId, stepRunId string, opts *UpdateStepRunOpts) (*db.StepRunModel, error)

	CancelPendingStepRuns(tenantId, jobRunId, reason string) error

	ListStartableStepRuns(tenantId, jobRunId, parentStepRunId string) ([]*dbsqlc.StepRun, error)
}
