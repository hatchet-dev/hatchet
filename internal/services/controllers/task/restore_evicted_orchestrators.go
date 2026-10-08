package task

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

const (
	// stuckEvictedOrchestratorGrace is how long a durable orchestrator's runtime must have been
	// evicted, with every durable event log entry satisfied, before this pass restores it.
	// Generous so it never races a normal callback-driven restore.
	stuckEvictedOrchestratorGrace = 5 * time.Minute

	evictedTaskRuntimeWindowSize = 2000
)

// processStuckEvictedDurableOrchestrators restores DAG-orchestrator durable tasks that were
// evicted while blocked, are now ready (all durable event log entries satisfied), but never got
// the edge-triggered DurableRestoreTask -- the child callback that would have published it was
// lost (engine roll), or every entry was already satisfied at eviction time so no later callback
// arrived. Without this they sit evicted forever and their DAG stays RUNNING.
func (tc *TasksControllerImpl) processStuckEvictedDurableOrchestrators(ctx context.Context, tenantIdStr string) (bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "process-stuck-evicted-durable-orchestrators")
	defer span.End()

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "tenant.id", Value: tenantIdStr})

	tenantId, err := uuid.Parse(tenantIdStr)
	if err != nil {
		return false, fmt.Errorf("invalid tenant id %q: %w", tenantId, err)
	}

	cursor, _ := tc.evictedTaskRuntimeCursors.Load(tenantId)

	rows, err := tc.repov1.Tasks().ListEvictedTaskRuntimeWindow(ctx, tenantId, cursor, stuckEvictedOrchestratorGrace, evictedTaskRuntimeWindowSize)

	if err != nil {
		return false, fmt.Errorf("could not list evicted task runtime window for tenant %s: %w", tenantId, err)
	}

	if len(rows) < evictedTaskRuntimeWindowSize {
		tc.evictedTaskRuntimeCursors.Delete(tenantId)
	} else {
		tc.evictedTaskRuntimeCursors.Store(tenantId, v1.EvictedTaskRuntimeCursorFromRow(rows[len(rows)-1]))
	}

	for _, row := range rows {
		if !row.IsStuckDurableOrchestrator {
			continue
		}

		msg, err := tasktypes.DurableRestoreTaskMessage(
			tenantId,
			row.ExternalID,
			"periodic restore: durable orchestrator evicted with all durable events satisfied",
		)
		if err != nil {
			tc.l.Error().Ctx(ctx).Err(err).Msgf("could not build durable restore message for task %s", row.ExternalID)
			continue
		}

		if err := tc.mq.SendMessage(ctx, msgqueue.TASK_PROCESSING_QUEUE, msg); err != nil {
			tc.l.Error().Ctx(ctx).Err(err).Msgf("could not publish durable restore message for task %s", row.ExternalID)
			continue
		}

		tc.l.Warn().Ctx(ctx).Msgf("restoring stuck evicted durable orchestrator %s (evicted, all durable events satisfied)", row.ExternalID)
	}

	return false, nil
}
