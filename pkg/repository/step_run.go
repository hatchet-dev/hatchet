package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"

	"github.com/rs/zerolog"
)

type ListStepRunsOpts struct {
	JobRunId *string `validate:"omitempty,uuid"`

	WorkflowRunIds []string `validate:"dive,uuid"`

	Status *dbsqlc.StepRunStatus
}

func IsFinalStepRunStatus(status dbsqlc.StepRunStatus) bool {
	return status != dbsqlc.StepRunStatusPENDING &&
		status != dbsqlc.StepRunStatusPENDINGASSIGNMENT &&
		status != dbsqlc.StepRunStatusASSIGNED &&
		status != dbsqlc.StepRunStatusRUNNING &&
		status != dbsqlc.StepRunStatusCANCELLING
}

func IsFinalJobRunStatus(status dbsqlc.JobRunStatus) bool {
	return status != dbsqlc.JobRunStatusPENDING && status != dbsqlc.JobRunStatusRUNNING
}

func IsFinalWorkflowRunStatus(status dbsqlc.WorkflowRunStatus) bool {
	return status != dbsqlc.WorkflowRunStatusPENDING &&
		status != dbsqlc.WorkflowRunStatusRUNNING &&
		status != dbsqlc.WorkflowRunStatusQUEUED
}

type CreateStepRunEventOpts struct {
	EventMessage *string

	EventReason *dbsqlc.StepRunEventReason

	EventSeverity *dbsqlc.StepRunEventSeverity

	Timestamp *time.Time

	EventData map[string]interface{}
}

type QueueStepRunOpts struct {
	IsRetry bool

	// IsInternalRetry is true if the step run is being retried internally by the system, for example if
	// it was sent to an invalid dispatcher. This does not count towards the retry limit but still gets
	// highest priority in the queue.
	IsInternalRetry bool

	Input []byte
}

type UpdateStepRunOverridesDataOpts struct {
	OverrideKey string
	Data        []byte
	CallerFile  *string
}

func StepRunStatusPtr(status db.StepRunStatus) *db.StepRunStatus {
	return &status
}

func StepRunEventReasonPtr(reason dbsqlc.StepRunEventReason) *dbsqlc.StepRunEventReason {
	return &reason
}

func StepRunEventSeverityPtr(severity dbsqlc.StepRunEventSeverity) *dbsqlc.StepRunEventSeverity {
	return &severity
}

var ErrNoWorkerAvailable = fmt.Errorf("no worker available")
var ErrRateLimitExceeded = fmt.Errorf("rate limit exceeded")
var ErrStepRunIsNotAssigned = fmt.Errorf("step run is not assigned")
var ErrAlreadyQueued = fmt.Errorf("step run is already queued")
var ErrAlreadyRunning = fmt.Errorf("step run is already running")

type StepRunUpdateInfo struct {
	WorkflowRunFinalState bool
	WorkflowRunId         string
	WorkflowRunStatus     string
}

type ListStepRunEventOpts struct {
	// (optional) number of events to skip
	Offset *int

	// (optional) number of events to return
	Limit *int
}

type ListStepRunEventResult struct {
	Rows  []*dbsqlc.StepRunEvent
	Count int
}
type ListStepRunArchivesOpts struct {
	// (optional) number of events to skip
	Offset *int

	// (optional) number of events to return
	Limit *int
}

type ListStepRunArchivesResult struct {
	Rows  []*dbsqlc.StepRunResultArchive
	Count int
}

type GetStepRunFull struct {
	*dbsqlc.StepRun
	ChildWorkflowRuns []string
}

type RefreshTimeoutBy struct {
	IncrementTimeoutBy string `validate:"required,duration"`
}

var ErrPreflightReplayStepRunNotInFinalState = fmt.Errorf("step run is not in a final state")
var ErrPreflightReplayChildStepRunNotInFinalState = fmt.Errorf("child step run is not in a final state")

type StepRunAPIRepository interface {
	GetStepRunById(stepRunId string) (*GetStepRunFull, error)

	ListStepRunEvents(stepRunId string, opts *ListStepRunEventOpts) (*ListStepRunEventResult, error)

	ListStepRunEventsByWorkflowRunId(ctx context.Context, tenantId, workflowRunId string, lastId *int32) (*ListStepRunEventResult, error)

	ListStepRunArchives(tenantId, stepRunId string, opts *ListStepRunArchivesOpts) (*ListStepRunArchivesResult, error)
}

type QueuedStepRun struct {
	StepRunId    string
	WorkerId     string
	DispatcherId string
}

type QueueStepRunsResult struct {
	Queued             []QueuedStepRun
	SchedulingTimedOut []string
	Continue           bool
}

type ProcessStepRunUpdatesResult struct {
	SucceededStepRuns     []*dbsqlc.GetStepRunForEngineRow
	CompletedWorkflowRuns []*dbsqlc.ResolveWorkflowRunStatusRow
	Continue              bool
}

type StepRunEngineRepository interface {
	// ListStepRunsForWorkflowRun returns a list of step runs for a workflow run.
	ListStepRuns(ctx context.Context, tenantId string, opts *ListStepRunsOpts) ([]*dbsqlc.GetStepRunForEngineRow, error)

	// ListStepRunsToReassign returns a list of step runs which are in a reassignable state.
	ListStepRunsToReassign(ctx context.Context, tenantId string) ([]string, error)

	ListStepRunsToTimeout(ctx context.Context, tenantId string) ([]*dbsqlc.GetStepRunForEngineRow, error)

	StepRunStarted(ctx context.Context, tenantId, stepRunId string, startedAt time.Time) error

	StepRunSucceeded(ctx context.Context, tenantId, stepRunId string, finishedAt time.Time, output []byte) error

	StepRunCancelled(ctx context.Context, tenantId, stepRunId string, cancelledAt time.Time, cancelledReason string) error

	StepRunFailed(ctx context.Context, tenantId, stepRunId string, failedAt time.Time, errStr string) error

	ReplayStepRun(ctx context.Context, tenantId, stepRunId string, input []byte) (*dbsqlc.GetStepRunForEngineRow, error)

	// PreflightCheckReplayStepRun checks if a step run can be replayed. If it can, it will return nil.
	PreflightCheckReplayStepRun(ctx context.Context, tenantId, stepRunId string) error

	ReleaseStepRunSemaphore(ctx context.Context, tenantId, stepRunId string, isUserTriggered bool) error

	// UpdateStepRunOverridesData updates the overrides data field in the input for a step run. This returns the input
	// bytes.
	UpdateStepRunOverridesData(ctx context.Context, tenantId, stepRunId string, opts *UpdateStepRunOverridesDataOpts) ([]byte, error)

	UpdateStepRunInputSchema(ctx context.Context, tenantId, stepRunId string, schema []byte) ([]byte, error)

	GetStepRunForEngine(ctx context.Context, tenantId, stepRunId string) (*dbsqlc.GetStepRunForEngineRow, error)

	GetStepRunDataForEngine(ctx context.Context, tenantId, stepRunId string) (*dbsqlc.GetStepRunDataForEngineRow, error)

	GetStepRunMetaForEngine(ctx context.Context, tenantId, stepRunId string) (*dbsqlc.GetStepRunMetaRow, error)

	// QueueStepRun is like UpdateStepRun, except that it will only update the step run if it is in
	// a pending state.
	QueueStepRun(ctx context.Context, tenantId, stepRunId string, opts *QueueStepRunOpts) (*dbsqlc.GetStepRunForEngineRow, error)

	ProcessStepRunUpdates(ctx context.Context, qlp *zerolog.Logger, tenantId string) (ProcessStepRunUpdatesResult, error)

	QueueStepRuns(ctx context.Context, ql *zerolog.Logger, tenantId string) (QueueStepRunsResult, error)

	CleanupQueueItems(ctx context.Context, tenantId string) error

	CleanupInternalQueueItems(ctx context.Context, tenantId string) error

	ListStartableStepRuns(ctx context.Context, tenantId, jobRunId string, parentStepRunId *string) ([]*dbsqlc.GetStepRunForEngineRow, error)

	ArchiveStepRunResult(ctx context.Context, tenantId, stepRunId string, err *string) error

	RefreshTimeoutBy(ctx context.Context, tenantId, stepRunId string, opts RefreshTimeoutBy) (*dbsqlc.StepRun, error)

	DeferredStepRunEvent(
		tenantId, stepRunId string,
		opts CreateStepRunEventOpts,
	)

	ClearStepRunPayloadData(ctx context.Context, tenantId string) (bool, error)
}
