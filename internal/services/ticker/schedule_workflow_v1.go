package ticker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *TickerImpl) runScheduledWorkflowV1(ctx context.Context, tenantId uuid.UUID, workflowVersion *sqlcv1.GetWorkflowVersionForEngineRow, scheduledWorkflowId uuid.UUID, scheduled *sqlcv1.PollScheduledWorkflowsRow) error {
	expiresAt := scheduled.TriggerAt.Time.Add(time.Second * 30)
	err := t.repov1.Idempotency().CreateIdempotencyKey(ctx, tenantId, scheduledWorkflowId.String(), sqlchelpers.TimestamptzFromTime(expiresAt))

	var pgErr *pgconn.PgError
	// if we get a unique violation, it means we tried to create a duplicate idempotency key, which means this
	// run has already been processed, so we should just return
	if err != nil && errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
		t.l.Info().Msgf("idempotency key for scheduled workflow %s already exists, skipping", scheduledWorkflowId)
		return nil
	} else if err != nil {
		return fmt.Errorf("could not create idempotency key: %w", err)
	}

	key := v1.IdempotencyKey(scheduledWorkflowId.String())

	// send workflow run to task controller
	opt := &v1.WorkflowNameTriggerOpts{
		TriggerTaskData: &v1.TriggerTaskData{
			WorkflowName:       workflowVersion.WorkflowName,
			Data:               scheduled.Input,
			AdditionalMetadata: scheduled.AdditionalMetadata,
			Priority:           &scheduled.Priority,
		},
		IdempotencyKey: &key,
		ExternalId:     uuid.New(),
		ShouldSkip:     false,
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
