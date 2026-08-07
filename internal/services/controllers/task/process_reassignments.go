package task

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

func (tc *TasksControllerImpl) processTaskReassignments(ctx context.Context, tenantId string) (bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "process-task-reassignments")
	defer span.End()

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "tenant.id", Value: tenantId})
	tenantIdUUID := uuid.MustParse(tenantId)

	res, shouldContinue, err := tc.repov1.Tasks().ProcessTaskReassignments(ctx, tenantIdUUID)

	if err != nil {
		return false, fmt.Errorf("could not list step runs to reassign for tenant %s: %w", tenantId, err)
	}

	prometheus.ReassignedTasks.Add(float64(len(res.RetriedTasks)))
	if tc.promGate.Enabled(ctx, tenantIdUUID) {
		prometheus.TenantReassignedTasks.WithLabelValues(tenantId).Add(float64(len(res.RetriedTasks)))
	}

	tc.processFailTasksResponse(ctx, tenantIdUUID, res)

	return shouldContinue, nil
}
