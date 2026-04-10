package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(up20260408190713, down20260408190713)
}

func v1RunsOlapTenantStatusInsAtIdxName(table string) string {
	return fmt.Sprintf("ix_%s_tenant_status_ins_at", table)
}

func up20260408190713(ctx context.Context, db *sql.DB) error {
	// drop the old outdated index first
	stmt := "DROP INDEX IF EXISTS ix_v1_runs_olap_tenant_id"

	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("drop old index on %s: %w", v1RunsOlapTable, err)
	}

	grandchildPartitions, err := listLeafPartitions(ctx, db, v1RunsOlapTable, 2)
	if err != nil {
		return err
	}

	for _, partition := range grandchildPartitions {
		stmt := fmt.Sprintf(
			`CREATE INDEX CONCURRENTLY IF NOT EXISTS %s ON %s (tenant_id, readable_status, inserted_at DESC)`,
			quoteIdent(v1RunsOlapTenantStatusInsAtIdxName(partition)),
			quoteIdent(partition),
		)
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("create index concurrently on %s: %w", partition, err)
		}
	}

	childPartitions, err := listLeafPartitions(ctx, db, v1RunsOlapTable, 1)
	if err != nil {
		return err
	}

	for _, partition := range childPartitions {
		stmt := fmt.Sprintf(
			`CREATE INDEX IF NOT EXISTS %s ON %s (tenant_id, readable_status, inserted_at DESC)`,
			quoteIdent(v1RunsOlapTenantStatusInsAtIdxName(partition)),
			quoteIdent(partition),
		)
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("create index on partition %s: %w", partition, err)
		}
	}

	stmt = fmt.Sprintf(
		`CREATE INDEX IF NOT EXISTS %s ON %s (tenant_id, readable_status, inserted_at DESC)`,
		quoteIdent(v1RunsOlapTenantStatusInsAtIdxName(v1RunsOlapTable)),
		quoteIdent(v1RunsOlapTable),
	)
	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("create index on %s: %w", v1RunsOlapTable, err)
	}

	stmt = "ANALYZE v1_runs_olap, v1_dags_olap, v1_tasks_olap"

	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("analyze tables: %w", err)
	}

	return nil
}

func down20260408190713(ctx context.Context, db *sql.DB) error {
	// drop the new index first so we can rebuild the old one bottom-up
	stmt := "DROP INDEX IF EXISTS ix_v1_runs_olap_tenant_status_ins_at"
	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("drop new index on %s: %w", v1RunsOlapTable, err)
	}

	grandchildPartitions, err := listLeafPartitions(ctx, db, v1RunsOlapTable, 2)
	if err != nil {
		return err
	}

	for _, partition := range grandchildPartitions {
		stmt := fmt.Sprintf(
			`CREATE INDEX CONCURRENTLY IF NOT EXISTS %s ON %s (tenant_id, inserted_at, id, readable_status, kind)`,
			quoteIdent(idxNameForPartition(partition)),
			quoteIdent(partition),
		)
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("create index concurrently on %s: %w", partition, err)
		}
	}

	childPartitions, err := listLeafPartitions(ctx, db, v1RunsOlapTable, 1)
	if err != nil {
		return err
	}

	for _, partition := range childPartitions {
		stmt := fmt.Sprintf(
			`CREATE INDEX IF NOT EXISTS %s ON %s (tenant_id, inserted_at, id, readable_status, kind)`,
			quoteIdent(idxNameForPartition(partition)),
			quoteIdent(partition),
		)
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("create index on partition %s: %w", partition, err)
		}
	}

	stmt = fmt.Sprintf(
		`CREATE INDEX IF NOT EXISTS %s ON %s (tenant_id, inserted_at, id, readable_status, kind)`,
		quoteIdent(idxNameForPartition(v1RunsOlapTable)),
		quoteIdent(v1RunsOlapTable),
	)
	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("create index on %s: %w", v1RunsOlapTable, err)
	}

	return nil
}
