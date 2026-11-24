package sqlcv1

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func (q *Queries) BeginAndLockLogFilesForAppend(
	ctx context.Context,
	db DBTX,
	lockParams GetAndLockLogFilesWithBranchPointsParams,
	childSignalParams BulkUpsertDurableChildSignalCreatedEventsParams,
	entriesAfterLatestNodeParams BulkGetDurableEventLogEntriesAfterLatestNodeParams,
) (
	logFiles []*GetAndLockLogFilesWithBranchPointsRow,
	upsertedChildSignalEvents []*BulkUpsertDurableChildSignalCreatedEventsRow,
	entriesAfterLatestNode []*BulkGetDurableEventLogEntriesAfterLatestNodeRow,
	err error,
) {
	batch := &pgx.Batch{}

	batch.Queue("begin")

	batch.Queue(getAndLockLogFilesWithBranchPoints,
		lockParams.Mindurabletaskinsertedat,
		lockParams.Maxdurabletaskinsertedat,
		lockParams.Durabletaskids,
		lockParams.Durabletaskinsertedats,
		lockParams.Tenantids,
	).Query(func(rows pgx.Rows) error {
		for rows.Next() {
			var i GetAndLockLogFilesWithBranchPointsRow
			if err := rows.Scan(
				&i.V1DurableEventLogFile.TenantID,
				&i.V1DurableEventLogFile.DurableTaskID,
				&i.V1DurableEventLogFile.DurableTaskInsertedAt,
				&i.V1DurableEventLogFile.LatestInvocationCount,
				&i.V1DurableEventLogFile.LatestInsertedAt,
				&i.V1DurableEventLogFile.LatestNodeID,
				&i.V1DurableEventLogFile.LatestBranchID,
				&i.V1DurableEventLogFile.LatestSatisfiedOrder,
				&i.TenantID,
				&i.ID,
				&i.InsertedAt,
				&i.DurableTaskID,
				&i.DurableTaskInsertedAt,
				&i.FirstNodeIDInNewBranch,
				&i.ParentBranchID,
				&i.NextBranchID,
				&i.ReplayChildExternalIds,
			); err != nil {
				return err
			}
			logFiles = append(logFiles, &i)
		}

		return rows.Err()
	})

	if len(childSignalParams.Eventkeys) > 0 {
		batch.Queue(bulkUpsertDurableChildSignalCreatedEvents,
			childSignalParams.Mindurabletaskinsertedat,
			childSignalParams.Maxdurabletaskinsertedat,
			childSignalParams.Tenantids,
			childSignalParams.Durabletaskids,
			childSignalParams.Durabletaskinsertedats,
			childSignalParams.Eventkeys,
			childSignalParams.Childexternalids,
			childSignalParams.Invocationcounts,
		).Query(func(rows pgx.Rows) error {
			var err error
			upsertedChildSignalEvents, err = pgx.CollectRows(rows, pgx.RowToAddrOfStructByPos[BulkUpsertDurableChildSignalCreatedEventsRow])
			return err
		})
	}

	batch.Queue(bulkGetDurableEventLogEntriesAfterLatestNode,
		entriesAfterLatestNodeParams.Mindurabletaskinsertedat,
		entriesAfterLatestNodeParams.Maxdurabletaskinsertedat,
		entriesAfterLatestNodeParams.Durabletaskids,
		entriesAfterLatestNodeParams.Durabletaskinsertedats,
		entriesAfterLatestNodeParams.Entrycounts,
	).Query(func(rows pgx.Rows) error {
		var err error
		entriesAfterLatestNode, err = pgx.CollectRows(rows, pgx.RowToAddrOfStructByPos[BulkGetDurableEventLogEntriesAfterLatestNodeRow])
		return err
	})

	if err := db.SendBatch(ctx, batch).Close(); err != nil {
		return nil, nil, nil, err
	}

	return logFiles, upsertedChildSignalEvents, entriesAfterLatestNode, nil
}

func (q *Queries) BeginAndClaimDurableEventLogEntriesForTrigger(ctx context.Context, db DBTX, arg ClaimDurableEventLogEntriesForTriggerParams) ([]*V1DurableEventLogEntry, error) {
	batch := &pgx.Batch{}

	batch.Queue("begin")

	var claimedEntries []*V1DurableEventLogEntry

	batch.Queue(claimDurableEventLogEntriesForTrigger,
		arg.Mindurabletaskinsertedat,
		arg.Maxdurabletaskinsertedat,
		arg.Nodeids,
		arg.Branchids,
		arg.Durabletaskids,
		arg.Durabletaskinsertedats,
	).Query(func(rows pgx.Rows) error {
		var err error
		claimedEntries, err = pgx.CollectRows(rows, pgx.RowToAddrOfStructByPos[V1DurableEventLogEntry])
		return err
	})

	if err := db.SendBatch(ctx, batch).Close(); err != nil {
		return nil, err
	}

	return claimedEntries, nil
}
