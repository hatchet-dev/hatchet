package ticker

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

func (t *TickerImpl) RunScheduledWorkflowV1(ctx context.Context, tenantId uuid.UUID, opts v1.RunScheduledWorkflowV1Opts) error {
	_, err := RunScheduledWorkflow(ctx, t.l, t.mqv1, t.repov1, tenantId, opts)
	return err
}

func RunScheduledWorkflow(ctx context.Context, l *zerolog.Logger, mq msgqueue.MessageQueue, repo v1.Repository, tenantId uuid.UUID, opts v1.RunScheduledWorkflowV1Opts) (*uuid.UUID, error) {
	externalId := uuid.New()

	claimed, err := repo.Idempotency().ClaimKey(ctx, tenantId, fmt.Sprintf("hatchet_internal_%s", opts.ID.String()), opts.TriggerAt.Add(30*time.Second), externalId)

	if err != nil {
		return nil, fmt.Errorf("could not claim idempotency key for scheduled workflow: %w", err)
	}

	if !claimed {
		l.Info().Ctx(ctx).Msgf("idempotency key for scheduled workflow %s already claimed, skipping", opts.ID.String())
		return nil, nil
	}

	msg, err := tasktypes.TriggerTaskMessage(
		tenantId,
		&v1.WorkflowNameTriggerOpts{
			TriggerTaskData: &v1.TriggerTaskData{
				WorkflowName:       opts.WorkflowName,
				Data:               opts.Input,
				AdditionalMetadata: opts.AdditionalMetadata,
				Priority:           opts.Priority,
			},
			ExternalId: externalId,
			ShouldSkip: false,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("could not create trigger task message: %w", err)
	}

	if err := mq.SendMessage(ctx, msgqueue.TASK_PROCESSING_QUEUE, msg); err != nil {
		return nil, fmt.Errorf("could not send message to task queue: %w", err)
	}

	return &externalId, repo.WorkflowSchedules().DeleteScheduledWorkflow(ctx, tenantId, opts.ID)
}
