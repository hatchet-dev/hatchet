package ticker

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *TickerImpl) runScheduledWorkflowV1(ctx context.Context, tenantId uuid.UUID, workflowVersion *sqlcv1.GetWorkflowVersionForEngineRow, scheduledWorkflowId uuid.UUID, scheduled *sqlcv1.PollScheduledWorkflowsRow) error {
	expiresAt := pgtype.Timestamptz{Time: scheduled.TriggerAt.Time.Add(30 * time.Second), Valid: true}

	claimed, err := t.repov1.Idempotency().ClaimKey(ctx, tenantId, scheduledWorkflowId.String(), expiresAt, scheduledWorkflowId)
	if err != nil {
		return fmt.Errorf("could not claim idempotency key for scheduled workflow: %w", err)
	}

	if !claimed {
		t.l.Info().Ctx(ctx).Msgf("idempotency key for scheduled workflow %s already claimed, skipping", scheduledWorkflowId)
		return nil
	}

	opt := &v1.WorkflowNameTriggerOpts{
		TriggerTaskData: &v1.TriggerTaskData{
			WorkflowName:       workflowVersion.WorkflowName,
			Data:               scheduled.Input,
			AdditionalMetadata: scheduled.AdditionalMetadata,
			Priority:           &scheduled.Priority,
		},
		ExternalId: uuid.New(),
		ShouldSkip: false,
	}

	msg, err := tasktypes.TriggerTaskMessage(
		tenantId,
		opt,
	)

	if err != nil {
		return fmt.Errorf("could not create trigger task message: %w", err)
	}

	err = t.mqv1.SendMessage(ctx, msgqueue.TASK_PROCESSING_QUEUE, msg)

	if err != nil {
		return fmt.Errorf("could not send message to task queue: %w", err)
	}

	// delete the scheduled workflow
	return t.repov1.WorkflowSchedules().DeleteScheduledWorkflow(ctx, tenantId, scheduledWorkflowId)
}
