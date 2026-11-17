package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// errNoLiveWorkflowVersion means the workflow has no version a cron or schedule
// can attach to. Count changes treat that as nothing to count.
var errNoLiveWorkflowVersion = errors.New("workflow has no live version")

// allocatedResourceCountCompactionBatchSize is the number of deltas one
// compaction statement folds.
const allocatedResourceCountCompactionBatchSize = 10000

// allocatedResourceCountCorrectionLockKey serializes count corrections. Two
// that ran at once would each append the same correcting delta.
var allocatedResourceCountCorrectionLockKey = sqlchelpers.AdvisoryLockKey("allocated-resource-count-correction")

// lockLatestWorkflowVersion locks the latest live version of the workflow.
// Cron count changes and workflow deletes take this lock before they read or
// write, in the same order as version registration, so a version change and a
// count change cannot both apply a delta for the same rows.
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

// shareLatestWorkflowVersion takes a shared lock on the latest live version of
// the workflow. Schedule count changes take it before they read or write. It
// waits for a version registration or workflow delete holding
// lockLatestWorkflowVersion, so the two cannot both apply a delta for the same
// schedule, and schedule changes on the same workflow do not wait on each
// other.
func shareLatestWorkflowVersion(ctx context.Context, q *sqlcv1.Queries, db sqlcv1.DBTX, workflowId uuid.UUID) error {
	_, err := q.ShareLockWorkflowVersion(ctx, db, workflowId)
	if errors.Is(err, pgx.ErrNoRows) {
		return errNoLiveWorkflowVersion
	}

	return err
}

// allocatedResourceCountDelta is the change to apply for one tenant. An omitted
// field is zero and leaves that count unchanged.
type allocatedResourceCountDelta struct {
	Cron         int64
	ScheduledRun int64
	Webhook      int64
}

// adjustAllocatedResourceCounts appends the change as a delta row. Writers for
// the same tenant do not wait on each other, because none of them updates the
// tenant's count row.
func adjustAllocatedResourceCounts(ctx context.Context, q *sqlcv1.Queries, db sqlcv1.DBTX, tenantId uuid.UUID, delta allocatedResourceCountDelta) error {
	if delta.Cron == 0 && delta.ScheduledRun == 0 && delta.Webhook == 0 {
		return nil
	}

	return q.InsertAllocatedResourceCountDelta(ctx, db, sqlcv1.InsertAllocatedResourceCountDeltaParams{
		Tenantid:      tenantId,
		Crondelta:     delta.Cron,
		Scheduledelta: delta.ScheduledRun,
		Webhookdelta:  delta.Webhook,
	})
}
