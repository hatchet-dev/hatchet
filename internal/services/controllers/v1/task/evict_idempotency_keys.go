package task

import (
	"context"
	"fmt"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (tc *TasksControllerImpl) evictExpiredIdempotencyKeys(ctx context.Context, tenantId string) (bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "evict-expired-idempotency-keys")
	defer span.End()

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "tenant_id", Value: tenantId})

	err := tc.repov1.Idempotency().EvictExpiredIdempotencyKeys(ctx, sqlchelpers.UUIDFromStr(tenantId))

	if err != nil {
		return false, fmt.Errorf("failed to evict expired idempotency keys for tenant %s: %w", tenantId, err)
	}

	// hard-coded false here since the EvictExpiredIdempotencyKeys method deletes everything in one shot
	return false, nil
}
