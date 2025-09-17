package ticker

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	msgqueuev1 "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
)

func (t *TickerImpl) runScheduledWorkflowV1(ctx context.Context, tenantId string, workflowVersion *dbsqlc.GetWorkflowVersionForEngineRow, scheduledWorkflowId string, scheduled *dbsqlc.PollScheduledWorkflowsRow) error {
	expiresAt := scheduled.TriggerAt.Time.Add(time.Second * 30)
	err := t.repov1.Idempotency().CreateIdempotencyKey(ctx, tenantId, scheduledWorkflowId, sqlchelpers.TimestamptzFromTime(expiresAt))

	if err != nil {
		return fmt.Errorf("could not create idempotency key: %w", err)
	}

	key := v1.IdempotencyKey(scheduledWorkflowId)

	// send workflow run to task controller
	opt := &v1.WorkflowNameTriggerOpts{
		TriggerTaskData: &v1.TriggerTaskData{
			WorkflowName:       workflowVersion.WorkflowName,
			Data:               scheduled.Input,
			AdditionalMetadata: scheduled.AdditionalMetadata,
			Priority:           &scheduled.Priority,
		},
		IdempotencyKey: &key,
		ExternalId:     uuid.NewString(),
		ShouldSkip:     false,
	}

	msg, err := tasktypes.TriggerTaskMessage(
		tenantId,
		opt,
	)

	if err != nil {
		return fmt.Errorf("could not create trigger task message: %w", err)
	}

	err = t.mqv1.SendMessage(ctx, msgqueuev1.TASK_PROCESSING_QUEUE, msg)

	if err != nil {
		return fmt.Errorf("could not send message to task queue: %w", err)
	}

	// delete the scheduled workflow
	return t.repo.WorkflowRun().DeleteScheduledWorkflow(ctx, tenantId, scheduledWorkflowId)
}
