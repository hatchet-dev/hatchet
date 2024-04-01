package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

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

type StepRunAPIRepository interface {
	GetStepRunById(tenantId, stepRunId string) (*db.StepRunModel, error)

	GetFirstArchivedStepRunResult(tenantId, stepRunId string) (*db.StepRunResultArchiveModel, error)
}

type StepRunEngineRepository interface {
	// ListRunningStepRunsForTicker returns a list of step runs which are currently running for a ticker.
	ListRunningStepRunsForTicker(tickerId string) ([]*dbsqlc.GetStepRunForEngineRow, error)

	// ListRunningStepRunsForWorkflowRun returns a list of step runs which are currently running for a workflow run.
	ListRunningStepRunsForWorkflowRun(tenantId, workflowRunId string) ([]*dbsqlc.GetStepRunForEngineRow, error)

	// ListStepRunsToRequeue returns a list of step runs which are in a requeueable state.
	ListStepRunsToRequeue(tenantId string) ([]*dbsqlc.GetStepRunForEngineRow, error)

	// ListStepRunsToReassign returns a list of step runs which are in a reassignable state.
	ListStepRunsToReassign(tenantId string) ([]*dbsqlc.GetStepRunForEngineRow, error)

	UpdateStepRun(ctx context.Context, tenantId, stepRunId string, opts *UpdateStepRunOpts) (*dbsqlc.GetStepRunForEngineRow, *StepRunUpdateInfo, error)

	// UpdateStepRunOverridesData updates the overrides data field in the input for a step run. This returns the input
	// bytes.
	UpdateStepRunOverridesData(tenantId, stepRunId string, opts *UpdateStepRunOverridesDataOpts) ([]byte, error)

	UpdateStepRunInputSchema(tenantId, stepRunId string, schema []byte) ([]byte, error)

	AssignStepRunToWorker(ctx context.Context, tenantId, stepRunId, actionId string, stepTimeout pgtype.Text) (workerId string, dispatcherId string, err error)

	GetStepRunForEngine(tenantId, stepRunId string) (*dbsqlc.GetStepRunForEngineRow, error)

	// QueueStepRun is like UpdateStepRun, except that it will only update the step run if it is in
	// a pending state.
	QueueStepRun(ctx context.Context, tenantId, stepRunId string, opts *UpdateStepRunOpts) (*dbsqlc.GetStepRunForEngineRow, error)

	ListStartableStepRuns(tenantId, jobRunId string, parentStepRunId *string) ([]*dbsqlc.GetStepRunForEngineRow, error)

	ArchiveStepRunResult(tenantId, stepRunId string) error
}
