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

const v10158Columns = "(tenant_id, inserted_at DESC, readable_status, workflow_id)"

func v10158IndexName(table string) string {
	return fmt.Sprintf("ix_%s_tenant_ins_at_status_wf", table)
}

// upV10158 adds (tenant_id, inserted_at DESC, readable_status, workflow_id) on
// v1_runs_olap. workflow_id is a trailing key so workflow-filtered run lists check it
// on the index entry instead of fetching every tenant row in the window from the heap,
// while the scan still returns rows in inserted_at order for ORDER BY ... LIMIT.
//
// Postgres cannot create indexes concurrently on a partitioned parent table, so the
// index is built concurrently on each partition first; creating it on the parent then
// attaches the existing child indexes instead of rebuilding them. A child index left
// invalid by an interrupted build is dropped and rebuilt, since IF NOT EXISTS would
// otherwise skip it.
func upV10158(ctx context.Context, db *sql.DB) error {
	partitions, err := listLeafPartitions(ctx, db, "v1_runs_olap", 1)
	if err != nil {
		return err
	}

	for _, partition := range partitions {
		indexName := v10158IndexName(partition)

		valid, err := indexIsValid(ctx, db, indexName)
		if err != nil {
			return err
		}

		if valid {
			continue
		}

		if _, err := db.ExecContext(ctx, fmt.Sprintf(`DROP INDEX CONCURRENTLY IF EXISTS %s;`, quoteIdent(indexName))); err != nil {
			return fmt.Errorf("failed to drop invalid index %s: %w", indexName, err)
		}

		stmt := fmt.Sprintf(
			`CREATE INDEX CONCURRENTLY %s ON %s %s;`,
			quoteIdent(indexName),
			quoteIdent(partition),
			v10158Columns,
		)

		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to create index concurrently on %s: %w", partition, err)
		}
	}

	stmt := fmt.Sprintf(
		`CREATE INDEX IF NOT EXISTS %s ON v1_runs_olap %s;`,
		quoteIdent(v10158IndexName("v1_runs_olap")),
		v10158Columns,
	)

	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("failed to create index on %s: %w", "v1_runs_olap", err)
	}

	return nil
}

// downV10158 drops the parent index, which cascades to all child partition indexes.
func downV10158(ctx context.Context, db *sql.DB) error {
	stmt := fmt.Sprintf(`DROP INDEX IF EXISTS %s;`, quoteIdent(v10158IndexName("v1_runs_olap")))

	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("failed to drop index on %s: %w", "v1_runs_olap", err)
	}

	return nil
}
