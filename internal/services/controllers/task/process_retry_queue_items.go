package task

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"

	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
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

	queuedPayloads := make([]tasktypes.CreateMonitoringEventPayload, 0, len(retryQueueItems))

	for _, task := range retryQueueItems {
		queuedPayloads = append(queuedPayloads, tasktypes.CreateMonitoringEventPayload{
			TaskId:         task.TaskID,
			RetryCount:     task.TaskRetryCount,
			EventType:      sqlcv1.V1EventTypeOlapQUEUED,
			EventTimestamp: time.Now(),
		})
	}

	if innerErr := tc.repov1.OLAPOutbox().MonitoringEvents(ctx, tenantIdUUID, queuedPayloads...); innerErr != nil {
		err = multierror.Append(err, fmt.Errorf("could not publish monitoring event message: %w", innerErr))
	}

	if err != nil {
		return false, fmt.Errorf("could not list retry queue items for tenant %s: %w", tenantId, err)
	}

	return shouldContinue, nil
}
