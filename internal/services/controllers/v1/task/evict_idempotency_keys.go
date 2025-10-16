package task

import (
	"context"
	"fmt"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

func (tc *TasksControllerImpl) evictExpiredIdempotencyKeys(ctx context.Context, tenantId string) (bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "evict-expired-idempotency-keys")
	defer span.End()

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "tenant.id", Value: tenantId})

	err := tc.repov1.Idempotency().EvictExpiredIdempotencyKeys(ctx, sqlchelpers.UUIDFromStr(tenantId))

	if err != nil {
		return false, fmt.Errorf("failed to evict expired idempotency keys for tenant %s: %w", tenantId, err)
	}

	// hard-coded false here since the EvictExpiredIdempotencyKeys method deletes everything in one shot
	return false, nil
}
