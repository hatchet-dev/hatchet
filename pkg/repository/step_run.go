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

var ErrStepRunIsNotPending = fmt.Errorf("step run is not pending")
var ErrNoWorkerAvailable = fmt.Errorf("no worker available")
var ErrRateLimitExceeded = fmt.Errorf("rate limit exceeded")
var ErrStepRunIsNotAssigned = fmt.Errorf("step run is not assigned")

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
	GetStepRunById(stepRunId int64) (*GetStepRunFull, error)

	ListStepRunEvents(stepRunId int64, opts *ListStepRunEventOpts) (*ListStepRunEventResult, error)

	ListStepRunEventsByWorkflowRunId(ctx context.Context, tenantId, workflowRunId string, lastId *int32) (*ListStepRunEventResult, error)

	ListStepRunArchives(tenantId string, stepRunId int64, opts *ListStepRunArchivesOpts) (*ListStepRunArchivesResult, error)
}

type QueuedStepRun struct {
	StepRunId    int64
	WorkerId     string
	DispatcherId string
}

type QueueStepRunsResult struct {
	Queued             []QueuedStepRun
	SchedulingTimedOut []int64
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
	ListStepRunsToReassign(ctx context.Context, tenantId string) ([]int64, error)

	ListStepRunsToTimeout(ctx context.Context, tenantId string) ([]*dbsqlc.GetStepRunForEngineRow, error)

	StepRunStarted(ctx context.Context, tenantId string, stepRunId int64, startedAt time.Time) error

	StepRunSucceeded(ctx context.Context, tenantId string, stepRunId int64, finishedAt time.Time, output []byte) error

	StepRunCancelled(ctx context.Context, tenantId string, stepRunId int64, cancelledAt time.Time, cancelledReason string) error

	StepRunFailed(ctx context.Context, tenantId string, stepRunId int64, failedAt time.Time, errStr string) error

	ReplayStepRun(ctx context.Context, tenantId string, stepRunId int64, input []byte) (*dbsqlc.GetStepRunForEngineRow, error)

	// PreflightCheckReplayStepRun checks if a step run can be replayed. If it can, it will return nil.
	PreflightCheckReplayStepRun(ctx context.Context, tenantId string, stepRunId int64) error

	ReleaseStepRunSemaphore(ctx context.Context, tenantId string, stepRunId int64, isUserTriggered bool) error

	// UpdateStepRunOverridesData updates the overrides data field in the input for a step run. This returns the input
	// bytes.
	UpdateStepRunOverridesData(ctx context.Context, tenantId string, stepRunId int64, opts *UpdateStepRunOverridesDataOpts) ([]byte, error)

	UpdateStepRunInputSchema(ctx context.Context, tenantId string, stepRunId int64, schema []byte) ([]byte, error)

	GetStepRunForEngine(ctx context.Context, tenantId string, stepRunId int64) (*dbsqlc.GetStepRunForEngineRow, error)

	GetStepRunDataForEngine(ctx context.Context, tenantId string, stepRunId int64) (*dbsqlc.GetStepRunDataForEngineRow, error)

	GetStepRunMetaForEngine(ctx context.Context, tenantId string, stepRunId int64) (*dbsqlc.GetStepRunMetaRow, error)

	// QueueStepRun is like UpdateStepRun, except that it will only update the step run if it is in
	// a pending state.
	QueueStepRun(ctx context.Context, tenantId string, stepRunId int64, opts *QueueStepRunOpts) (*dbsqlc.GetStepRunForEngineRow, error)

	ProcessStepRunUpdates(ctx context.Context, qlp *zerolog.Logger, tenantId string) (ProcessStepRunUpdatesResult, error)

	QueueStepRuns(ctx context.Context, ql *zerolog.Logger, tenantId string) (QueueStepRunsResult, error)

	CleanupQueueItems(ctx context.Context, tenantId string) error

	CleanupInternalQueueItems(ctx context.Context, tenantId string) error

	ListStartableStepRuns(ctx context.Context, tenantId, jobRunId string, parentStepRunId *int64) ([]*dbsqlc.GetStepRunForEngineRow, error)

	ArchiveStepRunResult(ctx context.Context, tenantId string, stepRunId int64, err *string) error

	RefreshTimeoutBy(ctx context.Context, tenantId string, stepRunId int64, opts RefreshTimeoutBy) (*dbsqlc.StepRun, error)

	DeferredStepRunEvent(
		tenantId string, stepRunId int64,
		opts CreateStepRunEventOpts,
	)

	ClearStepRunPayloadData(ctx context.Context, tenantId string) (bool, error)
}
