package task

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

func (tc *TasksControllerImpl) evictExpiredIdempotencyKeys(ctx context.Context, tenantId uuid.UUID) (bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "evict-expired-idempotency-keys")
	defer span.End()

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "tenant.id", Value: tenantId})

	err := tc.repov1.Idempotency().EvictExpiredIdempotencyKeys(ctx, tenantId)

	if err != nil {
		return false, fmt.Errorf("failed to evict expired idempotency keys for tenant %s: %w", tenantId, err)
	}

	// hard-coded false here since the EvictExpiredIdempotencyKeys method deletes everything in one shot
	return false, nil
}
