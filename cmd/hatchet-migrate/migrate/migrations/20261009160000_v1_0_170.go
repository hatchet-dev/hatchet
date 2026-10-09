package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(upV10170, downV10170)
}

const (
	v10170ParentIndex = "ix_v1_runs_olap_tenant_active_ins_at"
	v10170Definition  = "(tenant_id, inserted_at DESC) WHERE readable_status IN ('QUEUED', 'RUNNING', 'EVICTED')"
)

func v10170IndexName(partition string) string {
	return fmt.Sprintf("ix_%s_tenant_active_ins_at", partition)
}

// upV10170 adds a partial index over runs that are still active, which serves counts of active runs across the
// whole retention window without scanning every finished run. It builds the index CONCURRENTLY on each partition,
// then creates the parent index, which attaches them, so the build does not block writes to v1_runs_olap.
func upV10170(ctx context.Context, db *sql.DB) error {
	partitions, err := listLeafPartitions(ctx, db, "v1_runs_olap", 1)
	if err != nil {
		return err
	}

	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}
	defer conn.Close()

	// Session settings would otherwise stay on the pooled connection for later migrations.
	defer func() {
		if _, err := conn.ExecContext(context.Background(), `RESET statement_timeout; RESET lock_timeout;`); err != nil {
			log.Printf("v1_0_170: failed to reset session settings: %v", err)
		}
	}()

	// Each per-partition build scans the partition twice, so it must not inherit a server or role statement_timeout.
	if _, err := conn.ExecContext(ctx, `SET statement_timeout = 0;`); err != nil {
		return fmt.Errorf("failed to disable statement_timeout: %w", err)
	}

	for i, partition := range partitions {
		indexName := v10170IndexName(partition)

		valid, err := indexIsValid(ctx, db, indexName)
		if err != nil {
			return err
		}

		if valid {
			continue
		}

		log.Printf("v1_0_170: building %s (partition %d/%d)", indexName, i+1, len(partitions))
		start := time.Now()

		if _, err := conn.ExecContext(ctx, fmt.Sprintf(`DROP INDEX CONCURRENTLY IF EXISTS %s;`, quoteIdent(indexName))); err != nil {
			return fmt.Errorf("failed to drop invalid index %s: %w", indexName, err)
		}

		// #nosec G201 -- identifiers are quoted and derived from internal migration logic, not user input
		stmt := fmt.Sprintf(
			`CREATE INDEX CONCURRENTLY %s ON %s %s;`,
			quoteIdent(indexName),
			quoteIdent(partition),
			v10170Definition,
		)

		if _, err := conn.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to create index concurrently on %s: %w", partition, err)
		}

		log.Printf("v1_0_170: built %s in %s", indexName, time.Since(start).Round(time.Second))
	}

	// The parent create takes SHARE locks on v1_runs_olap, so waiting behind a long transaction would queue every
	// writer; fail instead, a re-run only retries the attach. A partition created since the listing above gets its
	// index built here, which is quick because partitions are created a day ahead, while still empty.
	if _, err := conn.ExecContext(ctx, `SET lock_timeout = '5s';`); err != nil {
		return fmt.Errorf("failed to set lock_timeout: %w", err)
	}

	// #nosec G201 -- identifiers are quoted and derived from internal migration logic, not user input
	stmt := fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s ON v1_runs_olap %s;`, quoteIdent(v10170ParentIndex), v10170Definition)

	if _, err := conn.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("failed to create index on %s: %w", "v1_runs_olap", err)
	}

	return nil
}

func downV10170(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Dropping a partitioned index locks v1_runs_olap and then each partition; bound the wait so a long query fails
	// the drop instead of queueing writers.
	stmts := []string{
		`SET LOCAL lock_timeout = '5s'`,
		`SET LOCAL statement_timeout = '15s'`,
		fmt.Sprintf(`DROP INDEX IF EXISTS %s`, quoteIdent(v10170ParentIndex)),
	}

	for _, stmt := range stmts {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to drop %s: %w", v10170ParentIndex, err)
		}
	}

	return tx.Commit()
}
