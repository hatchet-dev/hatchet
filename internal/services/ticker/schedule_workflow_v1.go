package ticker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
)

func (t *TickerImpl) RunScheduledWorkflowV1(ctx context.Context, tenantId uuid.UUID, opts v1.RunScheduledWorkflowV1Opts) error {
	_, err := RunScheduledWorkflow(ctx, t.l, t.mqv1, t.repov1, tenantId, opts)
	return err
}

func RunScheduledWorkflow(ctx context.Context, l *zerolog.Logger, mq msgqueue.MessageQueue, repo v1.Repository, tenantId uuid.UUID, opts v1.RunScheduledWorkflowV1Opts) (uuid.UUID, error) {
	expiresAt := opts.TriggerAt.Add(time.Second * 30)
	err := repo.Idempotency().CreateIdempotencyKey(ctx, tenantId, opts.Id.String(), sqlchelpers.TimestamptzFromTime(expiresAt))

	var pgErr *pgconn.PgError
	if err != nil && errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
		l.Info().Ctx(ctx).Msgf("idempotency key for scheduled workflow %s already exists, skipping", opts.Id.String())
		return uuid.Nil, nil
	} else if err != nil {
		return uuid.Nil, fmt.Errorf("could not create idempotency key: %w", err)
	}

	key := v1.IdempotencyKey(opts.Id.String())
	externalId := uuid.New()

	msg, err := tasktypes.TriggerTaskMessage(
		tenantId,
		&v1.WorkflowNameTriggerOpts{
			TriggerTaskData: &v1.TriggerTaskData{
				WorkflowName:       opts.WorkflowName,
				Data:               opts.Input,
				AdditionalMetadata: opts.AdditionalMetadata,
				Priority:           opts.Priority,
			},
			IdempotencyKey: &key,
			ExternalId:     externalId,
			ShouldSkip:     false,
		},
	)

	if err != nil {
		return uuid.Nil, fmt.Errorf("could not create trigger task message: %w", err)
	}

	if err := mq.SendMessage(ctx, msgqueue.TASK_PROCESSING_QUEUE, msg); err != nil {
		return uuid.Nil, fmt.Errorf("could not send message to task queue: %w", err)
	}

	return externalId, repo.WorkflowSchedules().DeleteScheduledWorkflow(ctx, tenantId, opts.Id)
}
