package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// allocatedResourceCountDelta is the change to apply for one tenant. An omitted
// field is zero and leaves that count unchanged.
type allocatedResourceCountDelta struct {
	Cron         int64
	ScheduledRun int64
	Webhook      int64
}

func adjustAllocatedResourceCounts(ctx context.Context, q *sqlcv1.Queries, db sqlcv1.DBTX, tenantId uuid.UUID, delta allocatedResourceCountDelta) error {
	if delta.Cron == 0 && delta.ScheduledRun == 0 && delta.Webhook == 0 {
		return nil
	}

	return q.AdjustAllocatedResourceCounts(ctx, db, sqlcv1.AdjustAllocatedResourceCountsParams{
		Tenantid:      tenantId,
		Crondelta:     delta.Cron,
		Scheduledelta: delta.ScheduledRun,
		Webhookdelta:  delta.Webhook,
	})
}
