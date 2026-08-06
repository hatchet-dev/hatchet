package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(upV10136, downV10136)
}

func v10136IndexName(partition string) string {
	return fmt.Sprintf("ix_%s_external_id_seen_at", partition)
}

func upV10136(ctx context.Context, db *sql.DB) error {
	table := "v1_event"

	partitions, err := listLeafPartitions(ctx, db, table, 1)

	if err != nil {
		return err
	}

	for _, partition := range partitions {
		// #nosec G201 -- identifiers are quoted and derived from internal migration logic, not user input
		stmt := fmt.Sprintf(
			`CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS %s ON %s (external_id, seen_at);`,
			quoteIdent(v10136IndexName(partition)),
			quoteIdent(partition),
		)

		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to create index concurrently on %s: %w", partition, err)
		}
	}

	// #nosec G201 -- identifiers are quoted and derived from internal migration logic, not user input
	stmt := fmt.Sprintf("CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s (external_id, seen_at);", quoteIdent(v10136IndexName(table)), quoteIdent(table))

	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("failed to create index on %s: %w", table, err)
	}

	return nil
}

func downV10136(ctx context.Context, db *sql.DB) error {
	table := "v1_event"
	stmt := fmt.Sprintf("DROP INDEX IF EXISTS %s", quoteIdent(v10136IndexName(table)))

	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("failed to drop index on %s: %w", table, err)
	}

	return nil
}
