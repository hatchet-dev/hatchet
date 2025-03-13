package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/pkg/random"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

type CreateWorkflowRunOpts struct {
	// (optional) the workflow run display name
	DisplayName *string

	TenantId string `validate:"required,uuid"`

	// (required) the workflow version id
	WorkflowVersionId string `validate:"required,uuid"`

	ManualTriggerInput *string `validate:"omitnil,required_without=TriggeringEventId,required_without=Cron,required_without=ScheduledWorkflowId,excluded_with=TriggeringEventId,excluded_with=Cron,excluded_with=ScheduledWorkflowId"`

	// (optional) the event id that triggered the workflow run
	TriggeringEventId *string `validate:"omitnil,uuid,required_without=ManualTriggerInput,required_without=Cron,required_without=ScheduledWorkflowId,excluded_with=ManualTriggerInput,excluded_with=Cron,excluded_with=ScheduledWorkflowId"`

	// (optional) the cron schedule that triggered the workflow run
	Cron         *string `validate:"omitnil,cron,required_without=ManualTriggerInput,required_without=TriggeringEventId,required_without=ScheduledWorkflowId,excluded_with=ManualTriggerInput,excluded_with=TriggeringEventId,excluded_with=ScheduledWorkflowId"`
	CronParentId *string `validate:"omitnil,uuid,required_without=ManualTriggerInput,required_without=TriggeringEventId,required_without=ScheduledWorkflowId,excluded_with=ManualTriggerInput,excluded_with=TriggeringEventId,excluded_with=ScheduledWorkflowId"`
	CronName     *string `validate:"omitnil"`

	// (optional) the scheduled trigger
	ScheduledWorkflowId *string `validate:"omitnil,uuid,required_without=ManualTriggerInput,required_without=TriggeringEventId,required_without=Cron,excluded_with=ManualTriggerInput,excluded_with=TriggeringEventId,excluded_with=Cron"`

	InputData []byte

	TriggeredBy string

	GetGroupKeyRun *CreateGroupKeyRunOpts `validate:"omitempty"`

	// (optional) the parent workflow run which this workflow run was triggered from
	ParentId *string `validate:"omitempty,uuid"`

	// (optional) the parent step run id which this workflow run was triggered from
	ParentStepRunId *string `validate:"omitempty,uuid"`

	// (optional) the child key of the workflow run, if this is a child run of a different workflow
	ChildKey *string

	// (optional) the child index of the workflow run, if this is a child run of a different workflow
	// python sdk uses -1 as default value
	ChildIndex *int `validate:"omitempty,min=-1"`

	// (optional) additional metadata for the workflow run
	AdditionalMetadata map[string]interface{} `validate:"omitempty"`

	// (optional) the desired worker id for sticky state
	DesiredWorkerId *string `validate:"omitempty,uuid"`

	// (optional) the deduplication value for the workflow run
	DedupeValue *string `validate:"omitempty"`

	// (optional) the priority of the workflow run
	Priority *int32 `validate:"omitempty,min=1,max=3"`
}

type CreateGroupKeyRunOpts struct {
	// (optional) the input data
	Input []byte
}

type CreateWorkflowRunOpt func(*CreateWorkflowRunOpts)

func WithParent(
	parentId, parentStepRunId string,
	childIndex int,
	childKey *string,
	additionalMetadata map[string]interface{},
	parentAdditionalMetadata map[string]interface{},
) CreateWorkflowRunOpt {
	return func(opts *CreateWorkflowRunOpts) {
		opts.ParentId = &parentId
		opts.ParentStepRunId = &parentStepRunId
		opts.ChildIndex = &childIndex
		opts.ChildKey = childKey

		opts.AdditionalMetadata = parentAdditionalMetadata

		if opts.AdditionalMetadata == nil {
			opts.AdditionalMetadata = make(map[string]interface{})
		}

		for k, v := range additionalMetadata {
			opts.AdditionalMetadata[k] = v
		}

	}
}

func GetCreateWorkflowRunOptsFromManual(
	workflowVersion *dbsqlc.GetWorkflowVersionForEngineRow,
	input []byte,
	additionalMetadata map[string]interface{},
) (*CreateWorkflowRunOpts, error) {
	if input == nil {
		input = []byte("{}")
	}

	opts := &CreateWorkflowRunOpts{
		DisplayName:        StringPtr(getWorkflowRunDisplayName(workflowVersion.WorkflowName)),
		WorkflowVersionId:  sqlchelpers.UUIDToStr(workflowVersion.WorkflowVersion.ID),
		ManualTriggerInput: StringPtr(string(input)),
		TriggeredBy:        string(datautils.TriggeredByManual),
		InputData:          input,
		AdditionalMetadata: additionalMetadata,
	}

	if workflowVersion.ConcurrencyLimitStrategy.Valid && workflowVersion.ConcurrencyGroupId.Valid {
		opts.GetGroupKeyRun = &CreateGroupKeyRunOpts{
			Input: input,
		}
	}

	return opts, nil
}

func GetCreateWorkflowRunOptsFromParent(
	workflowVersion *dbsqlc.GetWorkflowVersionForEngineRow,
	input []byte,
	parentId, parentStepRunId string,
	childIndex int,
	childKey *string,
	additionalMetadata map[string]interface{},
	parentAdditionalMetadata map[string]interface{},
) (*CreateWorkflowRunOpts, error) {
	if input == nil {
		input = []byte("{}")
	}

	opts := &CreateWorkflowRunOpts{
		DisplayName:        StringPtr(getWorkflowRunDisplayName(workflowVersion.WorkflowName)),
		WorkflowVersionId:  sqlchelpers.UUIDToStr(workflowVersion.WorkflowVersion.ID),
		ManualTriggerInput: StringPtr(string(input)),
		TriggeredBy:        string(datautils.TriggeredByParent),
		InputData:          input,
	}

	WithParent(parentId, parentStepRunId, childIndex, childKey, additionalMetadata, parentAdditionalMetadata)(opts)

	if workflowVersion.ConcurrencyLimitStrategy.Valid && workflowVersion.ConcurrencyGroupId.Valid {
		opts.GetGroupKeyRun = &CreateGroupKeyRunOpts{
			Input: input,
		}
	}

	return opts, nil
}

func GetCreateWorkflowRunOptsFromEvent(
	eventId string,
	workflowVersion *dbsqlc.GetWorkflowVersionForEngineRow,
	input []byte,
	additionalMetadata map[string]interface{},
) (*CreateWorkflowRunOpts, error) {
	if input == nil {
		input = []byte("{}")
	}

	opts := &CreateWorkflowRunOpts{
		DisplayName:        StringPtr(getWorkflowRunDisplayName(workflowVersion.WorkflowName)),
		WorkflowVersionId:  sqlchelpers.UUIDToStr(workflowVersion.WorkflowVersion.ID),
		TriggeringEventId:  &eventId,
		TriggeredBy:        string(datautils.TriggeredByEvent),
		InputData:          input,
		AdditionalMetadata: additionalMetadata,
	}

	if workflowVersion.ConcurrencyLimitStrategy.Valid && workflowVersion.ConcurrencyGroupId.Valid {
		opts.GetGroupKeyRun = &CreateGroupKeyRunOpts{
			Input: input,
		}
	}

	return opts, nil
}

func GetCreateWorkflowRunOptsFromCron(
	cron,
	cronParentId string,
	cronName *string,
	workflowVersion *dbsqlc.GetWorkflowVersionForEngineRow,
	input []byte,
	additionalMetadata map[string]interface{},
) (*CreateWorkflowRunOpts, error) {
	if input == nil {
		input = []byte("{}")
	}

	opts := &CreateWorkflowRunOpts{
		DisplayName:        StringPtr(getWorkflowRunDisplayName(workflowVersion.WorkflowName)),
		WorkflowVersionId:  sqlchelpers.UUIDToStr(workflowVersion.WorkflowVersion.ID),
		Cron:               &cron,
		CronParentId:       &cronParentId,
		CronName:           cronName,
		TriggeredBy:        string(datautils.TriggeredByCron),
		InputData:          input,
		AdditionalMetadata: additionalMetadata,
	}

	if workflowVersion.ConcurrencyLimitStrategy.Valid && workflowVersion.ConcurrencyGroupId.Valid {
		opts.GetGroupKeyRun = &CreateGroupKeyRunOpts{
			Input: input,
		}
	}

	return opts, nil
}

func GetCreateWorkflowRunOptsFromSchedule(
	scheduledWorkflowId string,
	workflowVersion *dbsqlc.GetWorkflowVersionForEngineRow,
	input []byte,
	additionalMetadata map[string]interface{},
	fs ...CreateWorkflowRunOpt,
) (*CreateWorkflowRunOpts, error) {
	if input == nil {
		input = []byte("{}")
	}

	opts := &CreateWorkflowRunOpts{
		DisplayName:         StringPtr(getWorkflowRunDisplayName(workflowVersion.WorkflowName)),
		WorkflowVersionId:   sqlchelpers.UUIDToStr(workflowVersion.WorkflowVersion.ID),
		ScheduledWorkflowId: &scheduledWorkflowId,
		TriggeredBy:         string(datautils.TriggeredBySchedule),
		InputData:           input,
		AdditionalMetadata:  additionalMetadata,
	}

	if workflowVersion.ConcurrencyLimitStrategy.Valid && workflowVersion.ConcurrencyGroupId.Valid {
		opts.GetGroupKeyRun = &CreateGroupKeyRunOpts{
			Input: input,
		}
	}

	for _, f := range fs {
		f(opts)
	}

	return opts, nil
}

func getWorkflowRunDisplayName(workflowName string) string {
	workflowSuffix, _ := random.Generate(6)

	return workflowName + "-" + workflowSuffix
}

type ListWorkflowRunsOpts struct {
	// (optional) the workflow id
	WorkflowId *string `validate:"omitempty,uuid"`

	// (optional) the workflow version id
	WorkflowVersionId *string `validate:"omitempty,uuid"`

	// (optional) a list of workflow run ids to filter by
	Ids []string `validate:"omitempty,dive,uuid"`

	// (optional) the parent workflow run id
	ParentId *string `validate:"omitempty,uuid"`

	// (optional) the parent step run id
	ParentStepRunId *string `validate:"omitempty,uuid"`

	// (optional) the event id that triggered the workflow run
	EventId *string `validate:"omitempty,uuid"`

	// (optional) the group key for the workflow run
	GroupKey *string

	// (optional) the status of the workflow run
	Statuses *[]dbsqlc.WorkflowRunStatus

	// (optional) a list of kinds to filter by
	Kinds *[]dbsqlc.WorkflowKind

	// (optional) number of events to skip
	Offset *int

	// (optional) number of events to return
	Limit *int

	// (optional) the order by field
	OrderBy *string `validate:"omitempty,oneof=createdAt finishedAt startedAt duration"`

	// (optional) the order direction
	OrderDirection *string `validate:"omitempty,oneof=ASC DESC"`

	// (optional) a time after which the run was created
	CreatedAfter *time.Time

	// (optional) a time before which the run was created
	CreatedBefore *time.Time

	// (optional) a time after which the run was finished
	FinishedAfter *time.Time

	// (optional) a time before which the run was finished
	FinishedBefore *time.Time

	// (optional) exact metadata to filter by
	AdditionalMetadata map[string]interface{} `validate:"omitempty"`
}

type WorkflowRunsMetricsOpts struct {
	// (optional) the workflow id
	WorkflowId *string `validate:"omitempty,uuid"`

	// (optional) the workflow version id
	WorkflowVersionId *string `validate:"omitempty,uuid"`

	// (optional) the parent workflow run id
	ParentId *string `validate:"omitempty,uuid"`

	// (optional) the parent step run id
	ParentStepRunId *string `validate:"omitempty,uuid"`

	// (optional) the event id that triggered the workflow run
	EventId *string `validate:"omitempty,uuid"`

	// (optional) exact metadata to filter by
	AdditionalMetadata map[string]interface{} `validate:"omitempty"`

	// (optional) the time the workflow run was created before
	CreatedBefore *time.Time `validate:"omitempty"`

	// (optional) the time the workflow run was created after
	CreatedAfter *time.Time `validate:"omitempty"`
}

type ListWorkflowRunsResult struct {
	Rows  []*dbsqlc.ListWorkflowRunsRow
	Count int
}

type CreateWorkflowRunPullRequestOpts struct {
	RepositoryOwner       string
	RepositoryName        string
	PullRequestID         int
	PullRequestTitle      string
	PullRequestNumber     int
	PullRequestHeadBranch string
	PullRequestBaseBranch string
	PullRequestState      string
}

type ListPullRequestsForWorkflowRunOpts struct {
	State *string
}

type ListWorkflowRunRoundRobinsOpts struct {
	// (optional) the workflow id
	WorkflowId *string `validate:"omitempty,uuid"`

	// (optional) the workflow version id
	WorkflowVersionId *string `validate:"omitempty,uuid"`

	// (optional) the status of the workflow run
	Status *dbsqlc.WorkflowRunStatus

	// (optional) number of events to skip
	Offset *int

	// (optional) number of events to return
	Limit *int
}

type WorkflowRunMetricsCountOpts struct {
	// (optional) the workflow id
	WorkflowId *string `validate:"omitempty,uuid"`

	// (optional) the workflow version id
	WorkflowVersionId *string `validate:"omitempty,uuid"`
}

type StepRunForJobRun struct {
	*dbsqlc.GetStepRunsForJobRunsWithOutputRow
	ChildWorkflowsCount int
}

type ListScheduledWorkflowsOpts struct {
	// (optional) number of events to skip
	Offset *int

	// (optional) number of events to return
	Limit *int

	// (optional) the order by field
	OrderBy *string `validate:"omitempty,oneof=createdAt triggerAt"`

	// (optional) the order direction
	OrderDirection *string `validate:"omitempty,oneof=ASC DESC"`

	// (optional) the workflow id
	WorkflowId *string `validate:"omitempty,uuid"`

	// (optional) the parent workflow run id
	ParentWorkflowRunId *string `validate:"omitempty,uuid"`

	// (optional) the parent step run id
	ParentStepRunId *string `validate:"omitempty,uuid"`

	// (optional) statuses to filter by
	Statuses *[]dbsqlc.WorkflowRunStatus

	// (optional) include scheduled runs that are in the future
	IncludeFuture *bool

	// (optional) additional metadata for the workflow run
	AdditionalMetadata map[string]interface{} `validate:"omitempty"`
}

// TODO move this to workflow.go
type ListCronWorkflowsOpts struct {
	// (optional) number of events to skip
	Offset *int

	// (optional) number of events to return
	Limit *int

	// (optional) the order by field
	OrderBy *string `validate:"omitempty,oneof=createdAt name"`

	// (optional) the order direction
	OrderDirection *string `validate:"omitempty,oneof=ASC DESC"`

	// (optional) the workflow id
	WorkflowId *string `validate:"omitempty,uuid"`

	// (optional) additional metadata for the workflow run
	AdditionalMetadata map[string]interface{} `validate:"omitempty"`
}

type WorkflowRunAPIRepository interface {
	RegisterCreateCallback(callback TenantScopedCallback[*dbsqlc.WorkflowRun])

	// ListWorkflowRuns returns workflow runs for a given workflow version id.
	ListWorkflowRuns(ctx context.Context, tenantId string, opts *ListWorkflowRunsOpts) (*ListWorkflowRunsResult, error)

	// Counts by status
	WorkflowRunMetricsCount(ctx context.Context, tenantId string, opts *WorkflowRunsMetricsOpts) (*dbsqlc.WorkflowRunsMetricsCountRow, error)

	// List ScheduledWorkflows lists workflows by scheduled trigger
	ListScheduledWorkflows(ctx context.Context, tenantId string, opts *ListScheduledWorkflowsOpts) ([]*dbsqlc.ListScheduledWorkflowsRow, int64, error)

	// DeleteScheduledWorkflow deletes a scheduled workflow run
	DeleteScheduledWorkflow(ctx context.Context, tenantId, scheduledWorkflowId string) error

	// GetScheduledWorkflow gets a scheduled workflow run
	GetScheduledWorkflow(ctx context.Context, tenantId, scheduledWorkflowId string) (*dbsqlc.ListScheduledWorkflowsRow, error)

	// UpdateScheduledWorkflow updates a scheduled workflow run
	UpdateScheduledWorkflow(ctx context.Context, tenantId, scheduledWorkflowId string, triggerAt time.Time) error

	// CreateNewWorkflowRun creates a new workflow run for a workflow version.
	CreateNewWorkflowRun(ctx context.Context, tenantId string, opts *CreateWorkflowRunOpts) (*dbsqlc.WorkflowRun, error)

	// GetWorkflowRunById returns a workflow run by id.
	GetWorkflowRunById(ctx context.Context, tenantId, runId string) (*dbsqlc.GetWorkflowRunByIdRow, error)

	// GetWorkflowRunById returns a workflow run by id.
	GetWorkflowRunByIds(ctx context.Context, tenantId string, runIds []string) ([]*dbsqlc.GetWorkflowRunByIdsRow, error)

	GetStepsForJobs(ctx context.Context, tenantId string, jobIds []string) ([]*dbsqlc.GetStepsForJobsRow, error)

	GetStepRunsForJobRuns(ctx context.Context, tenantId string, jobRunIds []string) ([]*StepRunForJobRun, error)

	GetWorkflowRunShape(ctx context.Context, workflowVersionId uuid.UUID) ([]*dbsqlc.GetWorkflowRunShapeRow, error)
}

var (
	ErrWorkflowRunNotFound = fmt.Errorf("workflow run not found")
)

type ErrDedupeValueExists struct {
	DedupeValue string
}

func (e ErrDedupeValueExists) Error() string {
	return fmt.Sprintf("workflow run with dedupe value %s already exists", e.DedupeValue)
}

type UpdateWorkflowRunFromGroupKeyEvalOpts struct {
	GroupKey *string

	Error *string
}
type ChildWorkflowRun struct {
	ParentId        string
	ParentStepRunId string
	ChildIndex      int
	Childkey        *string
}

type WorkflowRunEngineRepository interface {
	RegisterCreateCallback(callback TenantScopedCallback[*dbsqlc.WorkflowRun])
	RegisterQueuedCallback(callback TenantScopedCallback[pgtype.UUID])

	// ListWorkflowRuns returns workflow runs for a given workflow version id.
	ListWorkflowRuns(ctx context.Context, tenantId string, opts *ListWorkflowRunsOpts) (*ListWorkflowRunsResult, error)

	GetChildWorkflowRun(ctx context.Context, parentId, parentStepRunId string, childIndex int, childkey *string) (*dbsqlc.WorkflowRun, error)

	GetChildWorkflowRuns(ctx context.Context, childWorkflowRuns []ChildWorkflowRun) ([]*dbsqlc.WorkflowRun, error)

	GetScheduledChildWorkflowRun(ctx context.Context, parentId, parentStepRunId string, childIndex int, childkey *string) (*dbsqlc.WorkflowTriggerScheduledRef, error)

	PopWorkflowRunsCancelInProgress(ctx context.Context, tenantId, workflowVersionId string, maxRuns int) (toCancel []*dbsqlc.WorkflowRun, toStart []*dbsqlc.WorkflowRun, err error)

	PopWorkflowRunsCancelNewest(ctx context.Context, tenantId, workflowVersionId string, maxRuns int) (toCancel []*dbsqlc.WorkflowRun, toStart []*dbsqlc.WorkflowRun, err error)

	PopWorkflowRunsRoundRobin(ctx context.Context, tenantId, workflowVersionId string, maxRuns int) ([]*dbsqlc.WorkflowRun, []*dbsqlc.GetStepRunForEngineRow, error)

	// CreateNewWorkflowRun creates a new workflow run for a workflow version.
	CreateNewWorkflowRun(ctx context.Context, tenantId string, opts *CreateWorkflowRunOpts) (*dbsqlc.WorkflowRun, error)

	// CreateNewWorkflowRuns creates new workflow runs in bulk
	CreateNewWorkflowRuns(ctx context.Context, tenantId string, opts []*CreateWorkflowRunOpts) ([]*dbsqlc.WorkflowRun, error)

	CreateDeDupeKey(ctx context.Context, tenantId, workflowRunId, worrkflowVersionId, dedupeValue string) error

	GetWorkflowRunInputData(tenantId, workflowRunId string) (map[string]interface{}, error)

	ProcessWorkflowRunUpdates(ctx context.Context, tenantId string) (bool, error)

	UpdateWorkflowRunFromGroupKeyEval(ctx context.Context, tenantId, workflowRunId string, opts *UpdateWorkflowRunFromGroupKeyEvalOpts) error

	// GetWorkflowRunById returns a workflow run by id.
	GetWorkflowRunById(ctx context.Context, tenantId, runId string) (*dbsqlc.GetWorkflowRunRow, error)

	DeleteScheduledWorkflow(ctx context.Context, tenantId, scheduledWorkflowId string) error

	// TODO maybe we don't need this?
	GetWorkflowRunByIds(ctx context.Context, tenantId string, runId []string) ([]*dbsqlc.GetWorkflowRunRow, error)

	QueuePausedWorkflowRun(ctx context.Context, tenantId, workflowId, workflowRunId string) error

	QueueWorkflowRunJobs(ctx context.Context, tenant string, workflowRun string) ([]*dbsqlc.GetStepRunForEngineRow, error)

	ProcessUnpausedWorkflowRuns(ctx context.Context, tenantId string) ([]*dbsqlc.GetWorkflowRunRow, bool, error)

	GetWorkflowRunAdditionalMeta(ctx context.Context, tenantId, workflowRunId string) (*dbsqlc.GetWorkflowRunAdditionalMetaRow, error)

	ReplayWorkflowRun(ctx context.Context, tenantId, workflowRunId string) (*dbsqlc.GetWorkflowRunRow, error)

	ListActiveQueuedWorkflowVersions(ctx context.Context, tenantId string) ([]*dbsqlc.ListActiveQueuedWorkflowVersionsRow, error)

	// DeleteExpiredWorkflowRuns deletes workflow runs that were created before the given time. It returns the number of deleted runs
	// and the number of non-deleted runs that match the conditions.
	SoftDeleteExpiredWorkflowRuns(ctx context.Context, tenantId string, statuses []dbsqlc.WorkflowRunStatus, before time.Time) (bool, error)

	SoftDeleteSelectedWorkflowRuns(ctx context.Context, tenantId pgtype.UUID, ids []string) (int64, error)

	GetUpstreamErrorsForOnFailureStep(ctx context.Context, onFailureStepRunId string) ([]*dbsqlc.GetUpstreamErrorsForOnFailureStepRow, error)
}
