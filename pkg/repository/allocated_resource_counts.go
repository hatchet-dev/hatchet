package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// errNoLiveWorkflowVersion means the workflow has no version a cron or schedule
// can attach to. Count changes treat that as nothing to count.
var errNoLiveWorkflowVersion = errors.New("workflow has no live version")

// lockLatestWorkflowVersion locks the latest live version of the workflow.
// Cron and schedule count changes take this lock before they read or write,
// in the same order as version registration, so a version change and a count
// change cannot both apply a delta for the same rows.
func lockLatestWorkflowVersion(ctx context.Context, q *sqlcv1.Queries, db sqlcv1.DBTX, workflowId uuid.UUID) (uuid.UUID, error) {
	id, err := q.LockWorkflowVersion(ctx, db, workflowId)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, errNoLiveWorkflowVersion
	}

	if err != nil {
		return uuid.Nil, err
	}

	return id, nil
}

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
