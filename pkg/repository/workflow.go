package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/digest"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
)

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
}

type CreateWorkflowConcurrencyOpts struct {
	// (required) the action id for getting the concurrency group
	Action string `validate:"required,actionId"`

	// (optional) the maximum number of concurrent workflow runs, default 1
	MaxRuns *int32

	// (optional) the strategy to use when the concurrency limit is reached, default CANCEL_IN_PROGRESS
	LimitStrategy *string `validate:"omitnil,oneof=CANCEL_IN_PROGRESS DROP_NEWEST QUEUE_NEWEST GROUP_ROUND_ROBIN"`
}

func (o *CreateWorkflowVersionOpts) Checksum() (string, error) {
	// compute a checksum for the workflow
	declaredValues, err := datautils.ToJSONMap(o)

	if err != nil {
		return "", err
	}

	workflowChecksum, err := digest.DigestValues(declaredValues)

	if err != nil {
		return "", err
	}

	return workflowChecksum.String(), nil
}

type CreateWorkflowSchedulesOpts struct {
	ScheduledTriggers []time.Time

	Input []byte
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
}

type CreateWorkflowStepRateLimitOpts struct {
	// (required) the rate limit key
	Key string `validate:"required"`

	// (required) the rate limit units to consume
	Units int
}

type ListWorkflowsOpts struct {
	// (optional) number of workflows to skip
	Offset *int

	// (optional) number of workflows to return
	Limit *int

	// (optional) the event key to filter by
	EventKey *string
}

type ListWorkflowsRow struct {
	*db.WorkflowModel

	LatestRun *db.WorkflowRunModel
}

type ListWorkflowsResult struct {
	Rows  []*ListWorkflowsRow
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

type WorkflowAPIRepository interface {
	// ListWorkflows returns all workflows for a given tenant.
	ListWorkflows(tenantId string, opts *ListWorkflowsOpts) (*ListWorkflowsResult, error)

	// GetWorkflowById returns a workflow by its name. It will return db.ErrNotFound if the workflow does not exist.
	GetWorkflowById(workflowId string) (*db.WorkflowModel, error)

	// GetWorkflowByName returns a workflow by its name. It will return db.ErrNotFound if the workflow does not exist.
	GetWorkflowByName(tenantId, workflowName string) (*db.WorkflowModel, error)

	// GetWorkflowVersionById returns a workflow version by its id. It will return db.ErrNotFound if the workflow
	// version does not exist.
	GetWorkflowVersionById(tenantId, workflowId string) (*db.WorkflowVersionModel, error)

	// DeleteWorkflow deletes a workflow for a given tenant.
	DeleteWorkflow(tenantId, workflowId string) (*db.WorkflowModel, error)

	// GetWorkflowVersionMetrics returns the metrics for a given workflow version.
	GetWorkflowMetrics(tenantId, workflowId string, opts *GetWorkflowMetricsOpts) (*WorkflowMetrics, error)
}

type WorkflowEngineRepository interface {
	// CreateNewWorkflow creates a new workflow for a given tenant. It will create the parent
	// workflow based on the version's name.
	CreateNewWorkflow(ctx context.Context, tenantId string, opts *CreateWorkflowVersionOpts) (*dbsqlc.GetWorkflowVersionForEngineRow, error)

	// CreateWorkflowVersion creates a new workflow version for a given tenant. This will fail if there is
	// not a parent workflow with the same name already in the database.
	CreateWorkflowVersion(ctx context.Context, tenantId string, opts *CreateWorkflowVersionOpts) (*dbsqlc.GetWorkflowVersionForEngineRow, error)

	// CreateSchedules creates schedules for a given workflow version.
	CreateSchedules(ctx context.Context, tenantId, workflowVersionId string, opts *CreateWorkflowSchedulesOpts) ([]*dbsqlc.WorkflowTriggerScheduledRef, error)

	// GetScheduledById returns a scheduled workflow by its id.
	// GetScheduledById(tenantId, scheduleTriggerId string) (*db.WorkflowTriggerScheduledRefModel, error)

	GetLatestWorkflowVersion(ctx context.Context, tenantId, workflowId string) (*dbsqlc.GetWorkflowVersionForEngineRow, error)

	// GetWorkflowByName returns a workflow by its name. It will return db.ErrNotFound if the workflow does not exist.
	GetWorkflowByName(ctx context.Context, tenantId, workflowName string) (*dbsqlc.Workflow, error)

	// ListWorkflowsForEvent returns the latest workflow versions for a given tenant that are triggered by the
	// given event.
	ListWorkflowsForEvent(ctx context.Context, tenantId, eventKey string) ([]*dbsqlc.GetWorkflowVersionForEngineRow, error)

	// GetWorkflowVersionById returns a workflow version by its id. It will return db.ErrNotFound if the workflow
	// version does not exist.
	GetWorkflowVersionById(ctx context.Context, tenantId, workflowVersionId string) (*dbsqlc.GetWorkflowVersionForEngineRow, error)
}
