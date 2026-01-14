package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(up20251226153622, down20251226153622)
}

const (
	v1RunsOlapTable = "v1_runs_olap"
)

// up20251226153622 creates an index concurrently on each leaf partition of v1_runs_olap.
// Postgres 15 cannot create indexes concurrently on a partitioned parent table, so we
// must build them per-partition.
func up20251226153622(ctx context.Context, db *sql.DB) error {
	partitions, err := listLeafPartitions(ctx, db, v1RunsOlapTable, 2)
	if err != nil {
		return err
	}

	for _, partition := range partitions {
		stmt := fmt.Sprintf(
			`CREATE INDEX CONCURRENTLY IF NOT EXISTS %s ON %s (tenant_id, inserted_at, id, readable_status, kind)`,
			quoteIdent(idxNameForPartition(partition)),
			quoteIdent(partition),
		)

		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("create index concurrently on %s: %w", partition, err)
		}
	}

	partitions, err = listLeafPartitions(ctx, db, v1RunsOlapTable, 1)
	if err != nil {
		return err
	}

	for _, partition := range partitions {
		stmt := fmt.Sprintf(
			`CREATE INDEX IF NOT EXISTS %s ON %s (tenant_id, inserted_at, id, readable_status, kind)`,
			quoteIdent(idxNameForPartition(partition)),
			quoteIdent(partition),
		)

		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("create index on partition %s: %w", partition, err)
		}
	}

	stmt := fmt.Sprintf(
		`CREATE INDEX IF NOT EXISTS %s ON %s (tenant_id, inserted_at, id, readable_status, kind)`,
		quoteIdent(idxNameForPartition(v1RunsOlapTable)),
		quoteIdent(v1RunsOlapTable),
	)

	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("create index on %s: %w", v1RunsOlapTable, err)
	}

	return nil
}

func down20251226153622(ctx context.Context, db *sql.DB) error {
	partitions, err := listLeafPartitions(ctx, db, v1RunsOlapTable, 2)
	if err != nil {
		return err
	}

	for _, partition := range partitions {
		stmt := fmt.Sprintf(
			`DROP INDEX CONCURRENTLY IF EXISTS %s`,
			quoteIdent(idxNameForPartition(partition)),
		)

		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("drop index concurrently %s (partition %s): %w", idxNameForPartition(partition), partition, err)
		}
	}

	stmt := fmt.Sprintf(
		`DROP INDEX IF EXISTS %s`,
		quoteIdent(idxNameForPartition(v1RunsOlapTable)),
	)

	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("drop index on %s: %w", v1RunsOlapTable, err)
	}

	return nil
}

func listLeafPartitions(ctx context.Context, db *sql.DB, parentTable string, level int) ([]string, error) {
	// Leaf partitions are level=2 for this schema (range partition by date -> list partition by status).
	rows, err := db.QueryContext(ctx, `
SELECT relid::regclass::text AS partition
FROM pg_partition_tree($1::regclass)
WHERE level = $2
ORDER BY 1
`, parentTable, level)
	if err != nil {
		return nil, fmt.Errorf("list leaf partitions for %s: %w", parentTable, err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, fmt.Errorf("scan leaf partition: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate leaf partitions: %w", err)
	}

	return out, nil
}

func quoteIdent(ident string) string {
	// Minimal identifier quoting suitable for Postgres.
	return `"` + strings.ReplaceAll(ident, `"`, `""`) + `"`
}

func idxNameForPartition(partition string) string {
	return fmt.Sprintf("ix_%s_tenant_id", partition)
}
