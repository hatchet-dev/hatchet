package task

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

func (tc *TasksControllerImpl) processTaskRetryQueueItems(ctx context.Context, tenantId string) (bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "process-retry-queue-items")
	defer span.End()

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "tenant.id", Value: tenantId})
	tenantIdUUID := uuid.MustParse(tenantId)

	retryQueueItems, shouldContinue, err := tc.repov1.Tasks().ProcessTaskRetryQueueItems(ctx, tenantIdUUID)

	if err != nil {
		return false, fmt.Errorf("could not list step runs to reassign for tenant %s: %w", tenantId, err)
	}

	if num := len(retryQueueItems); num > 0 {
		tc.l.Info().Ctx(ctx).Msgf("reassigning %d step runs", num)
	}

	return shouldContinue, nil
}
