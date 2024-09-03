package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
)

type WorkflowRunEventEngineRepository interface {

	// CreateWorkflowRunEvent creates a new workflow run event
	CreateSucceededWorkflowRunEvent(ctx context.Context, tenantId string, workflowRunId string) error
	CreateQueuedWorkflowRunEvent(ctx context.Context, tenantId string, workflowRunId string) error
	CreateFailedWorkflowRunEvent(ctx context.Context, tenantId string, workflowRunId string) error
	CreatePendingWorkflowRunEvent(ctx context.Context, tenantId string, workflowRunId string) error
	CreateRunningWorkflowRunEvent(ctx context.Context, tenantId string, workflowRunId string) error
	GetWorkflowRunEventMetrics(ctx context.Context, tenantId string, startTimestamp *time.Time, endTimestamp *time.Time) ([]*dbsqlc.WorkflowRunEventsMetricsRow, error)
}

type CreateSucceededWorkflowRunEventOpts struct {
	TenantId      pgtype.UUID
	WorkflowRunId pgtype.UUID
}
