package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/pkg/random"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

type CreateWorkflowRunOpts struct {
	// (optional) the workflow run display name
	DisplayName *string

	// (required) the workflow version id
	WorkflowVersionId string `validate:"required,uuid"`

	ManualTriggerInput *string `validate:"omitnil,required_without=TriggeringEventId,required_without=Cron,required_without=ScheduledWorkflowId,excluded_with=TriggeringEventId,excluded_with=Cron,excluded_with=ScheduledWorkflowId"`

	// (optional) the event id that triggered the workflow run
	TriggeringEventId *string `validate:"omitnil,uuid,required_without=ManualTriggerInput,required_without=Cron,required_without=ScheduledWorkflowId,excluded_with=ManualTriggerInput,excluded_with=Cron,excluded_with=ScheduledWorkflowId"`

	// (optional) the cron schedule that triggered the workflow run
	Cron         *string `validate:"omitnil,cron,required_without=ManualTriggerInput,required_without=TriggeringEventId,required_without=ScheduledWorkflowId,excluded_with=ManualTriggerInput,excluded_with=TriggeringEventId,excluded_with=ScheduledWorkflowId"`
	CronParentId *string `validate:"omitnil,uuid,required_without=ManualTriggerInput,required_without=TriggeringEventId,required_without=ScheduledWorkflowId,excluded_with=ManualTriggerInput,excluded_with=TriggeringEventId,excluded_with=ScheduledWorkflowId"`

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
	ChildIndex *int

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

	if workflowVersion.ConcurrencyLimitStrategy.Valid {
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

	if workflowVersion.ConcurrencyLimitStrategy.Valid {
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

	if workflowVersion.ConcurrencyLimitStrategy.Valid {
		opts.GetGroupKeyRun = &CreateGroupKeyRunOpts{
			Input: input,
		}
	}

	return opts, nil
}

func GetCreateWorkflowRunOptsFromCron(
	cron,
	cronParentId string,
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
		TriggeredBy:        string(datautils.TriggeredByCron),
		InputData:          input,
		AdditionalMetadata: additionalMetadata,
	}

	if workflowVersion.ConcurrencyLimitStrategy.Valid {
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

	if workflowVersion.ConcurrencyLimitStrategy.Valid {
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
	Statuses *[]db.WorkflowRunStatus

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

	// (optional) a time before which the run was finished
	FinishedAfter *time.Time

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
	Status *db.WorkflowRunStatus

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

type WorkflowRunAPIRepository interface {
	RegisterCreateCallback(callback Callback[*db.WorkflowRunModel])

	// ListWorkflowRuns returns workflow runs for a given workflow version id.
	ListWorkflowRuns(ctx context.Context, tenantId string, opts *ListWorkflowRunsOpts) (*ListWorkflowRunsResult, error)

	// Counts by status
	WorkflowRunMetricsCount(ctx context.Context, tenantId string, opts *WorkflowRunsMetricsOpts) (*dbsqlc.WorkflowRunsMetricsCountRow, error)

	GetWorkflowRunInputData(tenantId, workflowRunId string) (map[string]interface{}, error)

	// CreateNewWorkflowRun creates a new workflow run for a workflow version.
	CreateNewWorkflowRun(ctx context.Context, tenantId string, opts *CreateWorkflowRunOpts) (*db.WorkflowRunModel, error)

	// GetWorkflowRunById returns a workflow run by id.
	GetWorkflowRunById(tenantId, runId string) (*db.WorkflowRunModel, error)
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

type WorkflowRunEngineRepository interface {
	RegisterCreateCallback(callback Callback[*dbsqlc.WorkflowRun])

	// ListWorkflowRuns returns workflow runs for a given workflow version id.
	ListWorkflowRuns(ctx context.Context, tenantId string, opts *ListWorkflowRunsOpts) (*ListWorkflowRunsResult, error)

	GetChildWorkflowRun(ctx context.Context, parentId, parentStepRunId string, childIndex int, childkey *string) (*dbsqlc.WorkflowRun, error)

	GetScheduledChildWorkflowRun(ctx context.Context, parentId, parentStepRunId string, childIndex int, childkey *string) (*dbsqlc.WorkflowTriggerScheduledRef, error)

	PopWorkflowRunsRoundRobin(ctx context.Context, tenantId, workflowId string, maxRuns int) ([]*dbsqlc.WorkflowRun, error)

	// CreateNewWorkflowRun creates a new workflow run for a workflow version.
	CreateNewWorkflowRun(ctx context.Context, tenantId string, opts *CreateWorkflowRunOpts) (string, error)

	// GetWorkflowRunById returns a workflow run by id.
	GetWorkflowRunById(ctx context.Context, tenantId, runId string) (*dbsqlc.GetWorkflowRunRow, error)

	GetWorkflowRunAdditionalMeta(ctx context.Context, tenantId, workflowRunId string) (*dbsqlc.GetWorkflowRunAdditionalMetaRow, error)

	ReplayWorkflowRun(ctx context.Context, tenantId, workflowRunId string) (*dbsqlc.GetWorkflowRunRow, error)

	ListActiveQueuedWorkflowVersions(ctx context.Context) ([]*dbsqlc.ListActiveQueuedWorkflowVersionsRow, error)

	// DeleteExpiredWorkflowRuns deletes workflow runs that were created before the given time. It returns the number of deleted runs
	// and the number of non-deleted runs that match the conditions.
	SoftDeleteExpiredWorkflowRuns(ctx context.Context, tenantId string, statuses []dbsqlc.WorkflowRunStatus, before time.Time) (bool, error)
}
