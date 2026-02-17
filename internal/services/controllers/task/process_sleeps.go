package task

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

func (tc *TasksControllerImpl) processSleeps(ctx context.Context, tenantId string) (bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "process-sleep")
	defer span.End()

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "tenant.id", Value: tenantId})
	tenantIdUUID, err := uuid.Parse(tenantId)

	if err != nil {
		return false, fmt.Errorf("could not parse tenant id %s: %w", tenantId, err)
	}

	matchResult, shouldContinue, err := tc.repov1.Tasks().ProcessDurableSleeps(ctx, tenantIdUUID)

	if err != nil {
		return false, fmt.Errorf("could not list process durable sleeps for tenant %s: %w", tenantId, err)
	}

	if len(matchResult.CreatedTasks) > 0 {
		err = tc.signaler.SignalTasksCreated(ctx, tenantIdUUID, matchResult.CreatedTasks)

		if err != nil {
			return false, fmt.Errorf("could not signal created tasks: %w", err)
		}
	}

	if len(matchResult.SatisfiedCallbacks) > 0 {
		if err := tc.processSatisfiedCallbacks(ctx, tenantIdUUID, matchResult.SatisfiedCallbacks); err != nil {
			tc.l.Error().Err(err).Msg("could not process satisfied callbacks from sleep")
		}
	}

	return shouldContinue, nil
}
