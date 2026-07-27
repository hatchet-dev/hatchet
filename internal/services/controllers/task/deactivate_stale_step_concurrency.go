package task

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

func (tc *TasksControllerImpl) deactivateStaleStepConcurrency(ctx context.Context, tenantId string) (bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "deactivate-stale-step-concurrency")
	defer span.End()

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "tenant.id", Value: tenantId})

	if err := tc.repov1.Scheduler().Concurrency().DeactivateStaleStepConcurrency(ctx, uuid.MustParse(tenantId)); err != nil {
		return false, fmt.Errorf("failed to deactivate stale step concurrency for tenant %s: %w", tenantId, err)
	}

	return false, nil
}
