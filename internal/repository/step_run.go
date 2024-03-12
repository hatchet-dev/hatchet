package repository

import (
	"context"
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

	RetryCount *int
}

type UpdateStepRunOverridesDataOpts struct {
	OverrideKey string
	Data        []byte
	CallerFile  *string
}

func StepRunStatusPtr(status db.StepRunStatus) *db.StepRunStatus {
	return &status
}

var ErrStepRunIsNotPending = fmt.Errorf("step run is not pending")
var ErrNoWorkerAvailable = fmt.Errorf("no worker available")

type StepRunUpdateInfo struct {
	JobRunFinalState      bool
	WorkflowRunFinalState bool
	WorkflowRunId         string
	WorkflowRunStatus     string
}

type StepRunRepository interface {
	// ListAllStepRuns returns a list of all step runs which match the given options.
	ListAllStepRuns(opts *ListAllStepRunsOpts) ([]db.StepRunModel, error)

	// ListStepRuns returns a list of step runs for a tenant which match the given options.
	ListStepRuns(tenantId string, opts *ListStepRunsOpts) ([]db.StepRunModel, error)

	// ListStepRunsToRequeue returns a list of step runs which are in a requeueable state.
	ListStepRunsToRequeue(tenantId string) ([]*dbsqlc.StepRun, error)

	// ListStepRunsToReassign returns a list of step runs which are in a reassignable state.
	ListStepRunsToReassign(tenantId string) ([]*dbsqlc.StepRun, error)

	UpdateStepRun(ctx context.Context, tenantId, stepRunId string, opts *UpdateStepRunOpts) (*dbsqlc.GetStepRunForEngineRow, *StepRunUpdateInfo, error)

	// UpdateStepRunOverridesData updates the overrides data field in the input for a step run. This returns the input
	// bytes.
	UpdateStepRunOverridesData(tenantId, stepRunId string, opts *UpdateStepRunOverridesDataOpts) ([]byte, error)

	UpdateStepRunInputSchema(tenantId, stepRunId string, schema []byte) ([]byte, error)

	AssignStepRunToWorker(tenantId, stepRunId string) (workerId string, dispatcherId string, err error)
	AssignStepRunToTicker(tenantId, stepRunId string) (tickerId string, err error)

	GetStepRunById(tenantId, stepRunId string) (*db.StepRunModel, error)
	GetStepRunForEngine(tenantId, stepRunId string) (*dbsqlc.GetStepRunForEngineRow, error)

	// QueueStepRun is like UpdateStepRun, except that it will only update the step run if it is in
	// a pending state.
	QueueStepRun(ctx context.Context, tenantId, stepRunId string, opts *UpdateStepRunOpts) (*dbsqlc.GetStepRunForEngineRow, error)

	ListStartableStepRuns(tenantId, jobRunId string, parentStepRunId *string) ([]*dbsqlc.GetStepRunForEngineRow, error)

	ArchiveStepRunResult(tenantId, stepRunId string) error

	ListArchivedStepRunResults(tenantId, stepRunId string) ([]db.StepRunResultArchiveModel, error)

	GetFirstArchivedStepRunResult(tenantId, stepRunId string) (*db.StepRunResultArchiveModel, error)
}
