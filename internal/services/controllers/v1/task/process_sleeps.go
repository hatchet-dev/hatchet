package task

import (
	"context"
	"fmt"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
)

func (tc *TasksControllerImpl) processSleeps(ctx context.Context, tenantId string) (bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "process-sleep")
	defer span.End()

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "tenant_id", Value: tenantId})

	matchResult, shouldContinue, err := tc.repov1.Tasks().ProcessDurableSleeps(ctx, tenantId)

	if err != nil {
		return false, fmt.Errorf("could not list process durable sleeps for tenant %s: %w", tenantId, err)
	}

	if len(matchResult.CreatedTasks) > 0 {
		err = tc.signalTasksCreated(ctx, tenantId, matchResult.CreatedTasks)

		if err != nil {
			return false, fmt.Errorf("could not signal created tasks: %w", err)
		}
	}

	return shouldContinue, nil
}
