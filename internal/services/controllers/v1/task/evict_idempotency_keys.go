package task

import (
	"context"
	"fmt"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

func (tc *TasksControllerImpl) runTenantEvictExpiredIdempotencyKeys(ctx context.Context) func() {
	return func() {
		tc.l.Debug().Msgf("idempotency: evicting expired idempotency keys for tasks")

		// list all tenants
		tenants, err := tc.p.ListTenantsForController(ctx, dbsqlc.TenantMajorEngineVersionV1)

		if err != nil {
			tc.l.Error().Err(err).Msg("could not list tenants")
			return
		}

		tc.evictExpiredIdempotencyKeysOperations.SetTenants(tenants)

		for i := range tenants {
			tenantId := sqlchelpers.UUIDToStr(tenants[i].ID)

			tc.evictExpiredIdempotencyKeysOperations.RunOrContinue(tenantId)
		}
	}
}

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
