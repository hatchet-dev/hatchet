package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(upV10142, downV10142)
}

const v1DurableEventLogEntryTable = "v1_durable_event_log_entry"

func upV10142(ctx context.Context, db *sql.DB) error {
	const batchSize = 5000

	partitions, err := listLeafPartitions(ctx, db, v1DurableEventLogEntryTable, 1)
	if err != nil {
		return err
	}

	for _, partition := range partitions {
		var lastDurableTaskId, lastBranchId, lastNodeId int64
		var lastDurableTaskInsertedAt pgtype.Timestamptz

		for {
			// #nosec G201 -- identifier is quoted and derived from pg_partition_tree, not user input
			stmt := fmt.Sprintf(
				`
				WITH batch AS (
					SELECT durable_task_id, durable_task_inserted_at, branch_id, node_id
					FROM %s
					WHERE (durable_task_id, durable_task_inserted_at, branch_id, node_id) > ($1, $2, $3, $4)
					ORDER BY durable_task_id, durable_task_inserted_at, branch_id, node_id
					LIMIT $5
				), updates AS (
					UPDATE %s e
					SET triggered_at = e.inserted_at
					FROM batch
					WHERE
						(e.durable_task_id, e.durable_task_inserted_at, e.branch_id, e.node_id) = (batch.durable_task_id, batch.durable_task_inserted_at, batch.branch_id, batch.node_id)
						AND e.triggered_at IS NULL
					RETURNING e.durable_task_id, e.durable_task_inserted_at, e.branch_id, e.node_id
				)

				SELECT durable_task_id, durable_task_inserted_at, branch_id, node_id
				FROM batch
				ORDER BY durable_task_id DESC, durable_task_inserted_at DESC, branch_id DESC, node_id DESC
				`,
				quoteIdent(partition),
				quoteIdent(partition),
			)

			rows, err := db.QueryContext(ctx, stmt, lastDurableTaskId, lastDurableTaskInsertedAt, lastBranchId, lastNodeId, batchSize)
			if err != nil {
				return fmt.Errorf("failed to backfill triggered_at on %s: %w", partition, err)
			}

			count := 0
			for rows.Next() {
				var durableTaskId, branchId, nodeId int64
				var durableTaskInsertedAt pgtype.Timestamptz

				if err := rows.Scan(&durableTaskId, &durableTaskInsertedAt, &branchId, &nodeId); err != nil {
					rows.Close()
					return fmt.Errorf("failed to scan backfilled row on %s: %w", partition, err)
				}

				if count == 0 {
					lastDurableTaskId = durableTaskId
					lastDurableTaskInsertedAt = durableTaskInsertedAt
					lastBranchId = branchId
					lastNodeId = nodeId
				}

				count++
			}

			if err := rows.Err(); err != nil {
				rows.Close()
				return fmt.Errorf("failed to iterate backfilled rows on %s: %w", partition, err)
			}

			rows.Close()

			if count == 0 {
				break
			}
		}
	}

	return nil
}

func downV10142(_ context.Context, _ *sql.DB) error {
	return nil
}
