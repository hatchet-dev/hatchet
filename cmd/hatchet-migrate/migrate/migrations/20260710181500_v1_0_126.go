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

var v10126GinTables = []string{"v1_runs_olap", "v1_tasks_olap"}

func v10126GinIndexName(table string) string {
	return fmt.Sprintf("ix_%s_additional_metadata_gin", table)
}

func upV10126(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `
		ALTER TABLE tenant_entitlement
		ADD COLUMN IF NOT EXISTS strict_additional_metadata_filters BOOLEAN NOT NULL DEFAULT FALSE
	`); err != nil {
		return fmt.Errorf("add strict_additional_metadata_filters to tenant_entitlement: %w", err)
	}

	for _, table := range v10126GinTables {
		partitions, err := listLeafPartitions(ctx, db, table, 1)
		if err != nil {
			return err
		}

		for _, partition := range partitions {
			stmt := fmt.Sprintf(
				`CREATE INDEX CONCURRENTLY IF NOT EXISTS %s ON %s USING gin (additional_metadata jsonb_path_ops);`,
				quoteIdent(v10126GinIndexName(partition)),
				quoteIdent(partition),
			)

			if _, err := db.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("failed to create gin index concurrently on %s: %w", partition, err)
			}
		}

		// Creating the index on the partitioned parent attaches the equivalent
		// partition indexes created above instead of rebuilding them, and ensures
		// future partitions are created with the index.
		stmt := fmt.Sprintf(
			"CREATE INDEX IF NOT EXISTS %s ON %s USING gin (additional_metadata jsonb_path_ops);",
			quoteIdent(v10126GinIndexName(table)),
			quoteIdent(table),
		)

		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to create gin index on %s: %w", table, err)
		}
	}

	return nil
}

func downV10126(ctx context.Context, db *sql.DB) error {
	for _, table := range v10126GinTables {
		stmt := fmt.Sprintf("DROP INDEX IF EXISTS %s;", quoteIdent(v10126GinIndexName(table)))

		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to drop gin index on %s: %w", table, err)
		}
	}

	if _, err := db.ExecContext(ctx, `
		ALTER TABLE tenant_entitlement
		DROP COLUMN IF EXISTS strict_additional_metadata_filters
	`); err != nil {
		return fmt.Errorf("drop strict_additional_metadata_filters from tenant_entitlement: %w", err)
	}

	return nil
}
