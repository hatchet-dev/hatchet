package repository

import (
	"context"
	"time"

	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
)

type CreateWorkflowVersionOpts struct {
	// (required) the workflow name
	Name string `validate:"required,hatchetName"`

	Tags []CreateWorkflowTagOpts `validate:"dive"`

	// (optional) the workflow description
	Description *string

	// (required) the workflow version
	Version string `validate:"required,semver"`

	// (optional) event triggers for the workflow
	EventTriggers []string

	// (optional) cron triggers for the workflow
	CronTriggers []string `validate:"dive,cron"`

	// (optional) scheduled triggers for the workflow
	ScheduledTriggers []time.Time

	// (required) the workflow jobs
	Jobs []CreateWorkflowJobOpts `validate:"required,min=1,dive"`
}

type CreateWorkflowSchedulesOpts struct {
	ScheduledTriggers []time.Time
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

	// (optional) the job timeout
	Timeout *string

	// (required) the job steps
	Steps []CreateWorkflowStepOpts `validate:"required,min=1,dive"`
}

type CreateWorkflowStepOpts struct {
	// (required) the step name
	ReadableId string `validate:"hatchetName"`

	// (required) the step action id
	Action string `validate:"required,actionId"`

	// (optional) the step timeout
	Timeout *string

	// (optional) the step inputs
	Inputs *db.JSON
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

type WorkflowRepository interface {
	// ListWorkflows returns all workflows for a given tenant.
	ListWorkflows(tenantId string, opts *ListWorkflowsOpts) (*ListWorkflowsResult, error)

	// CreateNewWorkflow creates a new workflow for a given tenant. It will create the parent
	// workflow based on the version's name.
	CreateNewWorkflow(tenantId string, opts *CreateWorkflowVersionOpts) (*db.WorkflowVersionModel, error)

	// CreateWorkflowVersion creates a new workflow version for a given tenant. This will fail if there is
	// not a parent workflow with the same name already in the database.
	CreateWorkflowVersion(tenantId string, opts *CreateWorkflowVersionOpts) (*db.WorkflowVersionModel, error)

	// CreateSchedules creates schedules for a given workflow version.
	CreateSchedules(tenantId, workflowVersionId string, opts *CreateWorkflowSchedulesOpts) ([]*db.WorkflowTriggerScheduledRefModel, error)

	// GetWorkflowById returns a workflow by its name. It will return db.ErrNotFound if the workflow does not exist.
	GetWorkflowById(workflowId string) (*db.WorkflowModel, error)

	// GetWorkflowByName returns a workflow by its name. It will return db.ErrNotFound if the workflow does not exist.
	GetWorkflowByName(tenantId, workflowName string) (*db.WorkflowModel, error)

	// ListWorkflowsForEvent returns the latest workflow versions for a given tenant that are triggered by the
	// given event.
	ListWorkflowsForEvent(ctx context.Context, tenantId, eventKey string) ([]db.WorkflowVersionModel, error)

	// GetWorkflowVersionById returns a workflow version by its id. It will return db.ErrNotFound if the workflow
	// version does not exist.
	GetWorkflowVersionById(tenantId, workflowId string) (*db.WorkflowVersionModel, error)

	// DeleteWorkflow deletes a workflow for a given tenant.
	DeleteWorkflow(tenantId, workflowId string) (*db.WorkflowModel, error)
}
