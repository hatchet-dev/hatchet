package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"

	"github.com/jackc/pgx/v5/pgtype"

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
	StepRunId string `validate:"required,uuid"`

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

	ExpressionEvals []CreateExpressionEvalOpt
}

type CreateExpressionEvalOpt struct {
	Key      string
	ValueStr *string
	ValueInt *int
	Kind     dbsqlc.StepExpressionKind
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

type ProcessStepRunUpdatesResultV2 struct {
	SucceededStepRuns     []*dbsqlc.GetStepRunForEngineRow
	CompletedWorkflowRuns []*dbsqlc.ResolveWorkflowRunStatusRow
	Continue              bool
}

type StepRunEngineRepository interface {
	RegisterWorkflowRunCompletedCallback(callback TenantScopedCallback[*dbsqlc.ResolveWorkflowRunStatusRow])

	ListStepRuns(ctx context.Context, tenantId string, opts *ListStepRunsOpts) ([]*dbsqlc.GetStepRunForEngineRow, error)

	ListStepRunsToCancel(ctx context.Context, tenantId, jobRunId string) ([]*dbsqlc.GetStepRunForEngineRow, error)

	// ListStepRunsToReassign returns a list of step runs which are in a reassignable state.
	ListStepRunsToReassign(ctx context.Context, tenantId string) (reassignedStepRunIds []string, failedStepRuns []*dbsqlc.GetStepRunForEngineRow, err error)

	InternalRetryStepRuns(ctx context.Context, tenantId string, srIdsIn []string) (reassignedStepRunIds []string, failedStepRuns []*dbsqlc.GetStepRunForEngineRow, err error)

	ListStepRunsToTimeout(ctx context.Context, tenantId string) (bool, []*dbsqlc.GetStepRunForEngineRow, error)

	StepRunAcked(ctx context.Context, tenantId, workflowRunId, stepRunId string, ackedAt time.Time) error

	StepRunStarted(ctx context.Context, tenantId, workflowRunId, stepRunId string, startedAt time.Time) error

	StepRunSucceeded(ctx context.Context, tenantId, workflowRunId, stepRunId string, finishedAt time.Time, output []byte) error

	StepRunCancelled(ctx context.Context, tenantId, workflowRunId, stepRunId string, cancelledAt time.Time, cancelledReason string, propagate bool) error

	StepRunFailed(ctx context.Context, tenantId, workflowRunId, stepRunId string, failedAt time.Time, errStr string, retryCount int) error

	StepRunRetryBackoff(ctx context.Context, tenantId, workflowRunId, stepRunId string, retryAfter time.Time, retryCount int) error

	RetryStepRuns(ctx context.Context, tenantId string) (bool, error)

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

	GetStepRunBulkDataForEngine(ctx context.Context, tenantId string, stepRunIds []string) ([]*dbsqlc.GetStepRunBulkDataForEngineRow, error)

	GetStepRunMetaForEngine(ctx context.Context, tenantId, stepRunId string) (*dbsqlc.GetStepRunMetaRow, error)

	// QueueStepRun is like UpdateStepRun, except that it will only update the step run if it is in
	// a pending state.
	QueueStepRun(ctx context.Context, tenantId, stepRunId string, opts *QueueStepRunOpts) (*dbsqlc.GetStepRunForEngineRow, error)

	GetQueueCounts(ctx context.Context, tenantId string) (map[string]int, error)

	ProcessStepRunUpdatesV2(ctx context.Context, qlp *zerolog.Logger, tenantId string) (ProcessStepRunUpdatesResultV2, error)

	CleanupQueueItems(ctx context.Context, tenantId string) error

	CleanupInternalQueueItems(ctx context.Context, tenantId string) error

	CleanupRetryQueueItems(ctx context.Context, tenantId string) error

	ListInitialStepRunsForJobRun(ctx context.Context, tenantId, jobRunId string) ([]*dbsqlc.GetStepRunForEngineRow, error)

	// ListStartableStepRuns returns a list of step runs that are in a startable state, assuming that the parentStepRunId has succeeded.
	// The singleParent flag is used to determine if we should reject listing step runs with many parents. This is important to avoid
	// race conditions where a step run is started by multiple parents completing at the same time. As a result, singleParent=false should
	// be called from a serializable process after processing step run status updates.
	ListStartableStepRuns(ctx context.Context, tenantId, parentStepRunId string, singleParent bool) ([]*dbsqlc.GetStepRunForEngineRow, error)

	ArchiveStepRunResult(ctx context.Context, tenantId, stepRunId string, err *string) error

	RefreshTimeoutBy(ctx context.Context, tenantId, stepRunId string, opts RefreshTimeoutBy) (pgtype.Timestamp, error)

	DeferredStepRunEvent(
		tenantId string,
		opts CreateStepRunEventOpts,
	)

	ClearStepRunPayloadData(ctx context.Context, tenantId string) (bool, error)
}
