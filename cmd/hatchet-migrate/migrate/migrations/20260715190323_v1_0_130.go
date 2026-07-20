package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(upV10126, downV10126)
}

func v10126IndexName(partition string) string {
	return fmt.Sprintf("ix_%s_idempotency_key", partition)
}

func upV10126(ctx context.Context, db *sql.DB) error {
	for _, table := range []string{"v1_tasks_olap", "v1_runs_olap"} {
		partitions, err := listLeafPartitions(ctx, db, table, 1)

		if err != nil {
			return err
		}

		for _, partition := range partitions {
			// #nosec G201 -- identifiers are quoted and derived from internal migration logic, not user input
			stmt := fmt.Sprintf(
				`CREATE INDEX CONCURRENTLY IF NOT EXISTS %s ON %s (idempotency_key, inserted_at) WHERE idempotency_key IS NOT NULL;`,
				quoteIdent(v10126IndexName(partition)),
				quoteIdent(partition),
			)

			if _, err := db.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("failed to create index concurrently on %s: %w", partition, err)
			}
		}

		// #nosec G201 -- identifiers are quoted and derived from internal migration logic, not user input
		stmt := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (idempotency_key, inserted_at) WHERE idempotency_key IS NOT NULL;", quoteIdent(v10126IndexName(table)), quoteIdent(table))

		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to create index on %s: %w", table, err)
		}
	}

	return nil
}

func downV10126(ctx context.Context, db *sql.DB) error {
	for _, table := range []string{"v1_tasks_olap", "v1_runs_olap"} {
		stmt := fmt.Sprintf("DROP INDEX IF EXISTS %s", quoteIdent(v10126IndexName(table)))

		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to drop index on %s: %w", table, err)
		}
	}

	return nil
}
