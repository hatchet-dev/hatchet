package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(upV10153, downV10153)
}

func v10153IndexName(table string) string {
	return fmt.Sprintf("%s_external_id_ins_at_uq", table)
}

func upV10153(ctx context.Context, db *sql.DB) error {
	table := "v1_payload"

	partitions, err := listLeafPartitions(ctx, db, table, 1)

	if err != nil {
		return err
	}

	for _, partition := range partitions {
		indexName := v10153IndexName(partition)
		valid, err := indexIsValid(ctx, db, indexName)

		if err != nil {
			return err
		}

		if valid {
			continue
		}

		if _, err := db.ExecContext(ctx, fmt.Sprintf(`DROP INDEX CONCURRENTLY IF EXISTS %s;`, quoteIdent(indexName))); err != nil {
			return fmt.Errorf("failed to drop stale index %s: %w", indexName, err)
		}

		// #nosec G201 -- identifiers are quoted and derived from internal migration logic, not user input
		buildStmt := fmt.Sprintf(
			`CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS %s ON %s (external_id ASC, inserted_at ASC);`,
			quoteIdent(indexName),
			quoteIdent(partition),
		)

		if _, err := db.ExecContext(ctx, buildStmt); err != nil {
			return fmt.Errorf("failed to create index concurrently on %s: %w", partition, err)
		}
	}

	// #nosec G201 -- identifiers are quoted and derived from internal migration logic, not user input
	stmt := fmt.Sprintf("CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s (external_id ASC, inserted_at ASC);", quoteIdent(v10153IndexName(table)), quoteIdent(table))

	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("failed to create index on %s: %w", table, err)
	}

	if _, err := db.ExecContext(ctx, "DROP INDEX IF EXISTS v1_payload_external_id_idx;"); err != nil {
		return fmt.Errorf("failed to drop old index on %s: %w", table, err)
	}

	for _, partition := range partitions {
		if _, err := db.ExecContext(ctx, fmt.Sprintf(`DROP INDEX CONCURRENTLY IF EXISTS %s;`, quoteIdent(v1PayloadPartitionIdxName(partition)))); err != nil {
			return fmt.Errorf("failed to drop old index on %s: %w", partition, err)
		}
	}

	return nil
}

func downV10153(ctx context.Context, db *sql.DB) error {
	table := "v1_payload"

	if _, err := db.ExecContext(ctx, fmt.Sprintf("DROP INDEX IF EXISTS %s;", quoteIdent(v10153IndexName(table)))); err != nil {
		return fmt.Errorf("failed to drop index on %s: %w", table, err)
	}

	if _, err := db.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS v1_payload_external_id_idx ON v1_payload (external_id ASC);"); err != nil {
		return fmt.Errorf("failed to recreate old index on %s: %w", table, err)
	}

	return nil
}
