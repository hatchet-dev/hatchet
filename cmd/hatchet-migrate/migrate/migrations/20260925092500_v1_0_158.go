package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(upV10158, downV10158)
}

// v10158Tables are the OLAP tables whose list/count queries have an "active runs older
// than the time window" branch. The partial index keeps that branch an index range scan
// over only QUEUED/RUNNING rows, which is a small, self-pruning subset of each partition.
var v10158Tables = []string{"v1_runs_olap", "v1_tasks_olap"}

func v10158IndexName(table string) string {
	return fmt.Sprintf("ix_%s_tenant_ins_at_active", table)
}

// CONCURRENTLY is not supported on a partitioned parent, so the index is built
// concurrently on each leaf partition first and then created on the parent, which
// attaches the matching partition indexes instead of rebuilding them. New partitions
// inherit the parent index automatically.
func upV10158(ctx context.Context, db *sql.DB) error {
	for _, table := range v10158Tables {
		partitions, err := listLeafPartitions(ctx, db, table, 1)

		if err != nil {
			return err
		}

		for _, partition := range partitions {
			// #nosec G201 -- identifiers are quoted and derived from internal migration logic, not user input
			stmt := fmt.Sprintf(
				`CREATE INDEX CONCURRENTLY IF NOT EXISTS %s ON %s (tenant_id, inserted_at DESC) WHERE readable_status IN ('QUEUED', 'RUNNING');`,
				quoteIdent(v10158IndexName(partition)),
				quoteIdent(partition),
			)

			if _, err := db.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("failed to create index concurrently on %s: %w", partition, err)
			}
		}

		// #nosec G201 -- identifiers are quoted and derived from internal migration logic, not user input
		stmt := fmt.Sprintf(
			`CREATE INDEX IF NOT EXISTS %s ON %s (tenant_id, inserted_at DESC) WHERE readable_status IN ('QUEUED', 'RUNNING');`,
			quoteIdent(v10158IndexName(table)),
			quoteIdent(table),
		)

		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to create index on %s: %w", table, err)
		}
	}

	return nil
}

func downV10158(ctx context.Context, db *sql.DB) error {
	for _, table := range v10158Tables {
		stmt := fmt.Sprintf("DROP INDEX IF EXISTS %s", quoteIdent(v10158IndexName(table)))

		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to drop index on %s: %w", table, err)
		}
	}

	return nil
}
