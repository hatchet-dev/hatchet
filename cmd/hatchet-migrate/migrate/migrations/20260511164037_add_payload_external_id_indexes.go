package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(up20260511164037, down20260511164037)
}

func up20260511164037(ctx context.Context, db *sql.DB) error {
	partitions, err := listLeafPartitions(ctx, db, "v1_payload", 1)
	if err != nil {
		return err
	}

	for _, partition := range partitions {
		stmt := fmt.Sprintf(
			`CREATE INDEX CONCURRENTLY IF NOT EXISTS %s ON %s (external_id ASC);`,
			quoteIdent(v1PayloadPartitionIdxName(partition)),
			quoteIdent(partition),
		)

		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to create index concurrently on %s: %w", partition, err)
		}
	}

	stmt := "CREATE INDEX IF NOT EXISTS v1_payload_external_id_idx ON v1_payload (external_id ASC);"

	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("failed to create index on %s: %w", "v1_payload", err)
	}

	olapPartitions, err := listLeafPartitions(ctx, db, "v1_payloads_olap", 1)
	if err != nil {
		return err
	}

	for _, partition := range olapPartitions {
		stmt := fmt.Sprintf(
			`CREATE INDEX CONCURRENTLY IF NOT EXISTS %s ON %s (external_id ASC);`,
			quoteIdent(v1PayloadPartitionIdxName(partition)),
			quoteIdent(partition),
		)

		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to create index concurrently on %s: %w", partition, err)
		}
	}

	stmt = "CREATE INDEX IF NOT EXISTS v1_payloads_olap_external_id_idx ON v1_payloads_olap (external_id ASC);"

	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("failed to create index on %s: %w", "v1_payloads_olap", err)
	}

	return nil
}

func down20260511164037(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, "DROP INDEX IF EXISTS v1_payload_external_id_idx"); err != nil {
		return fmt.Errorf("drop index on %s: %w", "v1_payload", err)
	}

	if _, err := db.ExecContext(ctx, "DROP INDEX IF EXISTS v1_payloads_olap_external_id_idx"); err != nil {
		return fmt.Errorf("drop index on %s: %w", "v1_payload", err)
	}

	return nil
}

func v1PayloadPartitionIdxName(partition string) string {
	return fmt.Sprintf("%s_external_id_idx", partition)
}
