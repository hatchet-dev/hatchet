package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(up20260324135804, down20260324135804)
}

const (
	v1LogLineTable     = "v1_log_line"
	v1LogLineParentIdx = "v1_log_line_tenant_id_level_idx"
)

func v1LogLineIdxName(partition string) string {
	return fmt.Sprintf("v1_log_line_tenant_id_level_idx_%s", partition)
}

// up20260324135804 creates an index concurrently on each leaf partition of v1_log_line.
// Postgres cannot create indexes concurrently on a partitioned parent table, so we
// must build them per-partition, then create the index on the parent non-concurrently
// (which attaches the already-built child indexes rather than rebuilding them).
func up20260324135804(ctx context.Context, db *sql.DB) error {
	partitions, err := listLeafPartitions(ctx, db, v1LogLineTable, 1)
	if err != nil {
		return err
	}

	for _, partition := range partitions {
		stmt := fmt.Sprintf(
			`CREATE INDEX CONCURRENTLY IF NOT EXISTS %s ON %s (tenant_id ASC, created_at DESC, level ASC)`,
			quoteIdent(v1LogLineIdxName(partition)),
			quoteIdent(partition),
		)

		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("create index concurrently on %s: %w", partition, err)
		}
	}

	stmt := fmt.Sprintf(
		`CREATE INDEX IF NOT EXISTS %s ON %s (tenant_id ASC, created_at DESC, level ASC)`,
		quoteIdent(v1LogLineParentIdx),
		quoteIdent(v1LogLineTable),
	)

	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("create index on %s: %w", v1LogLineTable, err)
	}

	return nil
}

// down20260324135804 drops the parent index, which cascades to all child partition indexes.
func down20260324135804(ctx context.Context, db *sql.DB) error {
	stmt := fmt.Sprintf(
		`DROP INDEX IF EXISTS %s`,
		quoteIdent(v1LogLineParentIdx),
	)

	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("drop index on %s: %w", v1LogLineTable, err)
	}

	return nil
}
