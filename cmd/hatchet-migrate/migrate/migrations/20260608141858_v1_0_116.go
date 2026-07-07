package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(upV10116, downV10116)
}

func v10116NewIndexName(partition string) string {
	return fmt.Sprintf("ix_%s_tenant_ins_at_status", partition)
}

func v10116OldIndexName(partition string) string {
	return fmt.Sprintf("ix_%s_tenant_status_ins_at", partition)
}

func upV10116(ctx context.Context, db *sql.DB) error {
	partitions, err := listLeafPartitions(ctx, db, "v1_runs_olap", 1)
	if err != nil {
		return err
	}

	for _, partition := range partitions {
		stmt := fmt.Sprintf(
			`CREATE INDEX CONCURRENTLY IF NOT EXISTS %s ON %s (tenant_id, inserted_at DESC, readable_status);`,
			quoteIdent(v10116NewIndexName(partition)),
			quoteIdent(partition),
		)

		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to create index concurrently on %s: %w", partition, err)
		}
	}

	stmt := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON v1_runs_olap (tenant_id, inserted_at DESC, readable_status);", quoteIdent(v10116NewIndexName("v1_runs_olap")))

	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("failed to create index on %s: %w", "v1_runs_olap", err)
	}

	stmt = fmt.Sprintf("DROP INDEX IF EXISTS %s;", quoteIdent(v10116OldIndexName("v1_runs_olap")))

	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("failed to drop old index on %s: %w", "v1_runs_olap", err)
	}

	return nil
}

func downV10116(ctx context.Context, db *sql.DB) error {
	partitions, err := listLeafPartitions(ctx, db, "v1_runs_olap", 1)
	if err != nil {
		return err
	}

	for _, partition := range partitions {
		stmt := fmt.Sprintf(
			`CREATE INDEX CONCURRENTLY IF NOT EXISTS %s ON %s (tenant_id, readable_status, inserted_at DESC);`,
			quoteIdent(v10116OldIndexName(partition)),
			quoteIdent(partition),
		)

		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to create index concurrently on %s: %w", partition, err)
		}
	}

	stmt := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON v1_runs_olap (tenant_id, readable_status, inserted_at DESC);", quoteIdent(v10116OldIndexName("v1_runs_olap")))

	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("failed to create index on %s: %w", "v1_runs_olap", err)
	}

	stmt = fmt.Sprintf("DROP INDEX IF EXISTS %s;", quoteIdent(v10116NewIndexName("v1_runs_olap")))

	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("failed to drop new index on %s: %w", "v1_runs_olap", err)
	}

	return nil
}
