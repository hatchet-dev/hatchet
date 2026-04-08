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

const (
	// v1RunsOlapStatusInsAtParentIdx is the name of the new parent index on v1_runs_olap.
	v1RunsOlapStatusInsAtParentIdx = "ix_v1_runs_olap_tenant_status_ins_at"
)

func idxStatusInsAtForPartition(partition string) string {
	return fmt.Sprintf("ix_%s_tenant_status_ins_at", partition)
}

// up20260408190713 replaces ix_v1_runs_olap_tenant_id (tenant_id, inserted_at, id, readable_status, kind)
// with a more selective index (tenant_id, readable_status, inserted_at DESC) on every partition.
// We build on the leaf partitions first (CONCURRENTLY), then create on the parent non-concurrently
// so Postgres attaches the already-built child indexes rather than rebuilding them.
// The old parent index is dropped last, which cascades to all partition children.
func up20260408190713(ctx context.Context, db *sql.DB) error {
	// Step 1: build new index on every level-2 leaf partition CONCURRENTLY.
	partitions, err := listLeafPartitions(ctx, db, v1RunsOlapTable, 2)
	if err != nil {
		return err
	}

	for _, partition := range partitions {
		stmt := fmt.Sprintf(
			`CREATE INDEX CONCURRENTLY IF NOT EXISTS %s ON %s (tenant_id, readable_status, inserted_at DESC)`,
			quoteIdent(idxStatusInsAtForPartition(partition)),
			quoteIdent(partition),
		)
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("create index concurrently on %s: %w", partition, err)
		}
	}

	// Step 2: create on the parent (NOT CONCURRENTLY). Postgres will create partitioned index
	// objects for each level-1 intermediate partition and attach the pre-built leaf indexes.
	stmt := fmt.Sprintf(
		`CREATE INDEX IF NOT EXISTS %s ON %s (tenant_id, readable_status, inserted_at DESC)`,
		quoteIdent(v1RunsOlapStatusInsAtParentIdx),
		quoteIdent(v1RunsOlapTable),
	)
	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("create index on %s: %w", v1RunsOlapTable, err)
	}

	// Step 3: drop the old parent index; this cascades to all partition children.
	stmt = fmt.Sprintf(
		`DROP INDEX IF EXISTS %s`,
		quoteIdent(idxNameForPartition(v1RunsOlapTable)),
	)
	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("drop old index on %s: %w", v1RunsOlapTable, err)
	}

	return nil
}

// down20260408190713 reverses the index swap: drops the new parent index (cascading to all
// partitions), then rebuilds the old index leaf-first followed by the parent.
func down20260408190713(ctx context.Context, db *sql.DB) error {
	// Step 1: drop the new parent index; cascades to all partition children.
	stmt := fmt.Sprintf(
		`DROP INDEX IF EXISTS %s`,
		quoteIdent(v1RunsOlapStatusInsAtParentIdx),
	)
	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("drop new index on %s: %w", v1RunsOlapTable, err)
	}

	// Step 2: rebuild old index on every level-2 leaf partition CONCURRENTLY.
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
			return fmt.Errorf("create old index concurrently on %s: %w", partition, err)
		}
	}

	// Step 3: create old index on the parent (NOT CONCURRENTLY; attaches pre-built leaf indexes).
	stmt = fmt.Sprintf(
		`CREATE INDEX IF NOT EXISTS %s ON %s (tenant_id, inserted_at, id, readable_status, kind)`,
		quoteIdent(idxNameForPartition(v1RunsOlapTable)),
		quoteIdent(v1RunsOlapTable),
	)
	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("create old index on %s: %w", v1RunsOlapTable, err)
	}

	return nil
}
