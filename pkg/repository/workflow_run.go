package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

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

type WorkflowRunAPIRepository interface {
	// List ScheduledWorkflows lists workflows by scheduled trigger
	ListScheduledWorkflows(ctx context.Context, tenantId string, opts *ListScheduledWorkflowsOpts) ([]*dbsqlc.ListScheduledWorkflowsRow, int64, error)

	// DeleteScheduledWorkflow deletes a scheduled workflow run
	DeleteScheduledWorkflow(ctx context.Context, tenantId, scheduledWorkflowId string) error

	// GetScheduledWorkflow gets a scheduled workflow run
	GetScheduledWorkflow(ctx context.Context, tenantId, scheduledWorkflowId string) (*dbsqlc.ListScheduledWorkflowsRow, error)

	// UpdateScheduledWorkflow updates a scheduled workflow run
	UpdateScheduledWorkflow(ctx context.Context, tenantId, scheduledWorkflowId string, triggerAt time.Time) error

	// ScheduledWorkflowMetaByIds returns minimal metadata for scheduled workflows by id.
	// Intended for bulk operations to avoid N+1 DB calls.
	ScheduledWorkflowMetaByIds(ctx context.Context, tenantId string, scheduledWorkflowIds []string) (map[string]ScheduledWorkflowMeta, error)

	// BulkDeleteScheduledWorkflows deletes scheduled workflows in bulk and returns deleted ids.
	BulkDeleteScheduledWorkflows(ctx context.Context, tenantId string, scheduledWorkflowIds []string) ([]string, error)

	// BulkUpdateScheduledWorkflows updates scheduled workflows in bulk and returns updated ids.
	BulkUpdateScheduledWorkflows(ctx context.Context, tenantId string, updates []ScheduledWorkflowUpdate) ([]string, error)

	GetWorkflowRunShape(ctx context.Context, workflowVersionId uuid.UUID) ([]*dbsqlc.GetWorkflowRunShapeRow, error)

	GetStepsForJobs(ctx context.Context, tenantId string, jobIds []string) ([]*dbsqlc.GetStepsForJobsRow, error)
}

type ScheduledWorkflowMeta struct {
	Id              string
	Method          dbsqlc.WorkflowTriggerScheduledRefMethods
	HasTriggeredRun bool
}

type ScheduledWorkflowUpdate struct {
	Id        string
	TriggerAt time.Time
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

type ChildWorkflowRun struct {
	ParentId        string
	ParentStepRunId string
	ChildIndex      int
	Childkey        *string
}

type WorkflowRunEngineRepository interface {
	DeleteScheduledWorkflow(ctx context.Context, tenantId, scheduledWorkflowId string) error
}
