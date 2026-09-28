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
// v1_runs_olap.
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
