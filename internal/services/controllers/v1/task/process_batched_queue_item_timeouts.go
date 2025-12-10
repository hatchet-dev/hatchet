package task

import (
	"context"
	"fmt"

	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

func (tc *TasksControllerImpl) processBatchedQueueItemTimeouts(ctx context.Context, tenantId string) (bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "process-batched-queue-item-timeout")
	defer span.End()

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "tenant.id", Value: tenantId})

	count, shouldContinue, err := tc.repov1.Tasks().ProcessBatchedQueueItemTimeouts(ctx, tenantId)

	if err != nil {
		return false, fmt.Errorf("could not process batched queue item timeouts for tenant %s: %w", tenantId, err)
	}

	if count > 0 {
		tc.l.Info().Int("count", count).Str("tenant_id", tenantId).Msg("processed batched queue item schedule timeouts")
	}

	return shouldContinue, nil
}
