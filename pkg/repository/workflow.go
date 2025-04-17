package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

var ErrDagParentNotFound = errors.New("dag parent not found")

type CreateWorkflowVersionOpts struct {
	// (required) the workflow name
	Name string `validate:"required,hatchetName"`

	Tags []CreateWorkflowTagOpts `validate:"dive"`

	// (optional) the workflow description
	Description *string `json:"description,omitempty"`

	// (optional) the workflow version
	Version *string `json:"version,omitempty"`

	// (optional) event triggers for the workflow
	EventTriggers []string

	// (optional) cron triggers for the workflow
	CronTriggers []string `validate:"dive,cron"`

	// (optional) the input bytes for the cron triggers
	CronInput []byte

	// (optional) scheduled triggers for the workflow
	ScheduledTriggers []time.Time

	// (required) the workflow jobs
	Jobs []CreateWorkflowJobOpts `validate:"required,min=1,dive"`

	OnFailureJob *CreateWorkflowJobOpts `json:"onFailureJob,omitempty" validate:"omitempty"`

	// (optional) the workflow concurrency groups
	Concurrency *CreateWorkflowConcurrencyOpts `json:"concurrency,omitempty" validator:"omitnil"`

	// (optional) the amount of time for step runs to wait to be scheduled before timing out
	ScheduleTimeout *string `validate:"omitempty,duration"`

	// (optional) sticky strategy
	Sticky *string `validate:"omitempty,oneof=SOFT HARD"`

	// (optional) the workflow kind
	Kind *string `validate:"omitempty,oneof=FUNCTION DURABLE DAG"`

	// (optional) the default priority for steps in the workflow (1-3)
	DefaultPriority *int32 `validate:"omitempty,min=1,max=3"`
}

type CreateCronWorkflowTriggerOpts struct {
	// (required) the workflow id
	WorkflowId string `validate:"required,uuid"`

	// (required) the workflow name
	Name string `validate:"required"`

	Cron string `validate:"required,cron"`

	Input              map[string]interface{}
	AdditionalMetadata map[string]interface{}

	Priority *int32 `validate:"omitempty,min=1,max=3"`
}

type CreateWorkflowConcurrencyOpts struct {
	// (optional) the action id for getting the concurrency group
	Action *string `validate:"omitempty,actionId"`

	// (optional) the maximum number of concurrent workflow runs, default 1
	MaxRuns *int32

	// (optional) the strategy to use when the concurrency limit is reached, default CANCEL_IN_PROGRESS
	LimitStrategy *string `validate:"omitnil,oneof=CANCEL_IN_PROGRESS GROUP_ROUND_ROBIN CANCEL_NEWEST"`

	// (optional) a concurrency expression for evaluating the concurrency key
	Expression *string `validate:"omitempty,celworkflowrunstr"`
}

type CreateWorkflowSchedulesOpts struct {
	ScheduledTriggers []time.Time

	Input              []byte
	AdditionalMetadata []byte

	Priority *int32 `validate:"omitnil,min=1,max=3"`
}

type CreateScheduledWorkflowRunForWorkflowOpts struct {
	WorkflowId string `validate:"required,uuid"`

	ScheduledTrigger time.Time

	Input              map[string]interface{}
	AdditionalMetadata map[string]interface{}

	Priority *int32 `validate:"omitempty,min=1,max=3"`
}

type CreateWorkflowTagOpts struct {
	// (required) the tag name
	Name string `validate:"required"`

	// (optional) the tag color
	Color *string
}

type CreateWorkflowJobOpts struct {
	// (required) the job name
	Name string `validate:"required,hatchetName"`

	// (optional) the job description
	Description *string

	// (required) the job steps
	Steps []CreateWorkflowStepOpts `validate:"required,min=1,dive"`

	Kind string `validate:"required,oneof=DEFAULT ON_FAILURE"`
}

type CreateWorkflowStepOpts struct {
	// (required) the step name
	ReadableId string `validate:"hatchetName"`

	// (required) the step action id
	Action string `validate:"required,actionId"`

	// (optional) the step timeout
	Timeout *string `validate:"omitnil,duration"`

	// (optional) the parents that this step depends on
	Parents []string `validate:"dive,hatchetName"`

	// (optional) the custom user data for the step, serialized as a json string
	UserData *string `validate:"omitnil,json"`

	// (optional) the step retry max
	Retries *int `validate:"omitempty,min=0"`

	// (optional) rate limits for this step
	RateLimits []CreateWorkflowStepRateLimitOpts `validate:"dive"`

	// (optional) desired worker affinity state for this step
	DesiredWorkerLabels map[string]DesiredWorkerLabelOpts `validate:"omitempty"`

	// (optional) the step retry backoff factor
	RetryBackoffFactor *float64 `validate:"omitnil,min=1,max=1000"`

	// (optional) the step retry backoff max seconds (can't be greater than 86400)
	RetryBackoffMaxSeconds *int `validate:"omitnil,min=1,max=86400"`
}

type DesiredWorkerLabelOpts struct {
	// (required) the label key
	Key string `validate:"required"`

	// (required if StringValue is nil) the label integer value
	IntValue *int32 `validate:"omitnil,required_without=StrValue"`

	// (required if StrValue is nil) the label string value
	StrValue *string `validate:"omitnil,required_without=IntValue"`

	// (optional) if the label is required
	Required *bool `validate:"omitempty"`

	// (optional) the weight of the label for scheduling (default: 100)
	Weight *int32 `validate:"omitempty"`

	// (optional) the label comparator for scheduling (default: EQUAL)
	Comparator *string `validate:"omitempty,oneof=EQUAL NOT_EQUAL GREATER_THAN LESS_THAN GREATER_THAN_OR_EQUAL LESS_THAN_OR_EQUAL"`
}

type CreateWorkflowStepRateLimitOpts struct {
	// (required) the rate limit key
	Key string `validate:"required"`

	// (optional) a CEL expression for the rate limit key
	KeyExpr *string `validate:"omitnil,celsteprunstr,required_without=Key"`

	// (optional) the rate limit units to consume
	Units *int `validate:"omitnil,required_without=UnitsExpr"`

	// (optional) a CEL expression for the rate limit units
	UnitsExpr *string `validate:"omitnil,celsteprunstr,required_without=Units"`

	// (optional) a CEL expression for a dynamic limit value for the rate limit
	LimitExpr *string `validate:"omitnil,celsteprunstr"`

	// (optional) the rate limit duration, defaults to MINUTE
	Duration *string `validate:"omitnil,oneof=SECOND MINUTE HOUR DAY WEEK MONTH YEAR"`
}

type ListWorkflowsOpts struct {
	// (optional) number of workflows to skip
	Offset *int

	// (optional) number of workflows to return
	Limit *int

	// (optional) the workflow name to filter by
	Name *string
}

type ListWorkflowsResult struct {
	Rows  []*dbsqlc.Workflow
	Count int
}

type JobRunHasCycleError struct {
	JobName string
}

func (e *JobRunHasCycleError) Error() string {
	return fmt.Sprintf("job %s has a cycle", e.JobName)
}

type UpsertWorkflowDeploymentConfigOpts struct {
	// (required) the github app installation id
	GithubAppInstallationId string `validate:"required,uuid"`

	// (required) the github repository name
	GitRepoName string `validate:"required"`

	// (required) the github repository owner
	GitRepoOwner string `validate:"required"`

	// (required) the github repository branch
	GitRepoBranch string `validate:"required"`
}

type WorkflowMetrics struct {
	// the number of runs for a specific group key
	GroupKeyRunsCount int `json:"groupKeyRunsCount,omitempty"`

	// the total number of concurrency group keys
	GroupKeyCount int `json:"groupKeyCount,omitempty"`
}

type GetWorkflowMetricsOpts struct {
	// (optional) the group key to filter by
	GroupKey *string

	// (optional) the workflow run status to filter by
	Status *string `validate:"omitnil,oneof=PENDING QUEUED RUNNING SUCCEEDED FAILED"`
}

type UpdateWorkflowOpts struct {
	// (optional) is paused -- if true, the workflow will not be scheduled
	IsPaused *bool
}

type WorkflowAPIRepository interface {
	// ListWorkflows returns all workflows for a given tenant.
	ListWorkflows(tenantId string, opts *ListWorkflowsOpts) (*ListWorkflowsResult, error)

	// GetWorkflowById returns a workflow by its name. It will return db.ErrNotFound if the workflow does not exist.
	GetWorkflowById(context context.Context, workflowId string) (*dbsqlc.GetWorkflowByIdRow, error)

	// GetWorkflowVersionById returns a workflow version by its id. It will return db.ErrNotFound if the workflow
	// version does not exist.
	GetWorkflowVersionById(tenantId, workflowVersionId string) (*dbsqlc.GetWorkflowVersionByIdRow,
		[]*dbsqlc.WorkflowTriggerCronRef,
		[]*dbsqlc.WorkflowTriggerEventRef,
		[]*dbsqlc.WorkflowTriggerScheduledRef,
		error)

	// DeleteWorkflow deletes a workflow for a given tenant.
	DeleteWorkflow(ctx context.Context, tenantId, workflowId string) (*dbsqlc.Workflow, error)

	// GetWorkflowVersionMetrics returns the metrics for a given workflow version.
	GetWorkflowMetrics(tenantId, workflowId string, opts *GetWorkflowMetricsOpts) (*WorkflowMetrics, error)

	// UpdateWorkflow updates a workflow for a given tenant.
	UpdateWorkflow(ctx context.Context, tenantId, workflowId string, opts *UpdateWorkflowOpts) (*dbsqlc.Workflow, error)

	// GetWorkflowWorkerCount returns the number of workers for a given workflow.
	GetWorkflowWorkerCount(tenantId, workflowId string) (int, int, error)

	// CreateCronWorkflow creates a cron trigger
	CreateCronWorkflow(ctx context.Context, tenantId string, opts *CreateCronWorkflowTriggerOpts) (*dbsqlc.ListCronWorkflowsRow, error)

	// List ScheduledWorkflows lists workflows by scheduled trigger
	ListCronWorkflows(ctx context.Context, tenantId string, opts *ListCronWorkflowsOpts) ([]*dbsqlc.ListCronWorkflowsRow, int64, error)

	// GetCronWorkflow gets a cron workflow run
	GetCronWorkflow(ctx context.Context, tenantId, cronWorkflowId string) (*dbsqlc.ListCronWorkflowsRow, error)

	// DeleteCronWorkflow deletes a cron workflow run
	DeleteCronWorkflow(ctx context.Context, tenantId, id string) error

	// CreateScheduledWorkflow creates a scheduled workflow run
	CreateScheduledWorkflow(ctx context.Context, tenantId string, opts *CreateScheduledWorkflowRunForWorkflowOpts) (*dbsqlc.ListScheduledWorkflowsRow, error)
}

type WorkflowEngineRepository interface {
	// CreateNewWorkflow creates a new workflow for a given tenant. It will create the parent
	// workflow based on the version's name.
	CreateNewWorkflow(ctx context.Context, tenantId string, opts *CreateWorkflowVersionOpts) (*dbsqlc.GetWorkflowVersionForEngineRow, error)

	// CreateWorkflowVersion creates a new workflow version for a given tenant. This will fail if there is
	// not a parent workflow with the same name already in the database.
	CreateWorkflowVersion(ctx context.Context, tenantId string, opts *CreateWorkflowVersionOpts, oldWorkflowVersion *dbsqlc.GetWorkflowVersionForEngineRow) (*dbsqlc.GetWorkflowVersionForEngineRow, error)

	// CreateSchedules creates schedules for a given workflow version.
	CreateSchedules(ctx context.Context, tenantId, workflowVersionId string, opts *CreateWorkflowSchedulesOpts) ([]*dbsqlc.WorkflowTriggerScheduledRef, error)

	// GetScheduledById returns a scheduled workflow by its id.
	// GetScheduledById(tenantId, scheduleTriggerId string) (*db.WorkflowTriggerScheduledRefModel, error)

	GetLatestWorkflowVersion(ctx context.Context, tenantId, workflowId string) (*dbsqlc.GetWorkflowVersionForEngineRow, error)

	GetLatestWorkflowVersions(ctx context.Context, tenantId string, workflowIds []string) ([]*dbsqlc.GetWorkflowVersionForEngineRow, error)

	// GetWorkflowByName returns a workflow by its name. It will return db.ErrNotFound if the workflow does not exist.
	GetWorkflowByName(ctx context.Context, tenantId, workflowName string) (*dbsqlc.Workflow, error)

	// GetWorkflowsByName returns all workflows by their name. It will return db.ErrNotFound if the workflow does not exist.
	GetWorkflowsByNames(ctx context.Context, tenantId string, workflowNames []string) ([]*dbsqlc.Workflow, error)

	// ListWorkflowsForEvent returns the latest workflow versions for a given tenant that are triggered by the
	// given event.
	ListWorkflowsForEvent(ctx context.Context, tenantId, eventKey string) ([]*dbsqlc.GetWorkflowVersionForEngineRow, error)

	// GetWorkflowVersionById returns a workflow version by its id. It will return db.ErrNotFound if the workflow
	// version does not exist.
	GetWorkflowVersionById(ctx context.Context, tenantId, workflowVersionId string) (*dbsqlc.GetWorkflowVersionForEngineRow, error)
}
