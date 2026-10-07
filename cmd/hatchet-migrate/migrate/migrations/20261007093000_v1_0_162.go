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
	goose.AddMigrationNoTxContext(upV10162, downV10162)
}

const (
	v10162ParentIndex = "ix_v1_runs_olap_tenant_ins_at_status"
	v10162Columns     = "(tenant_id, inserted_at DESC, readable_status)"
)

func v10162IndexName(partition string) string {
	return fmt.Sprintf("ix_%s_tenant_ins_at_status", partition)
}

// upV10162 drops ix_v1_runs_olap_tenant_ins_at_status. ix_v1_runs_olap_tenant_ins_at_status_wf leads with the same
// columns in the same order, so it serves tenant, insertion-time and status queries.
func upV10162(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Dropping a partitioned index locks v1_runs_olap and then each partition. lock_timeout bounds each wait and
	// statement_timeout the total time v1_runs_olap stays locked, so a long query fails the drop instead of queueing
	// writers; a re-run retries. SET LOCAL keeps both off the pooled connection that later migrations reuse.
	stmts := []string{
		`SET LOCAL lock_timeout = '5s'`,
		`SET LOCAL statement_timeout = '15s'`,
		fmt.Sprintf(`DROP INDEX IF EXISTS %s`, quoteIdent(v10162ParentIndex)),
	}

	for _, stmt := range stmts {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to drop %s: %w", v10162ParentIndex, err)
		}
	}

	return tx.Commit()
}

// downV10162 builds the index CONCURRENTLY on each partition, then creates the parent index, which attaches them,
// so a rollback does not block writes to v1_runs_olap for the length of the build.
func downV10162(ctx context.Context, db *sql.DB) error {
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
			log.Printf("v1_0_162: failed to reset session settings: %v", err)
		}
	}()

	// A per-partition build can run for many minutes, so it must not inherit a server or role statement_timeout.
	if _, err := conn.ExecContext(ctx, `SET statement_timeout = 0;`); err != nil {
		return fmt.Errorf("failed to disable statement_timeout: %w", err)
	}

	for i, partition := range partitions {
		indexName := v10162IndexName(partition)

		valid, err := indexIsValid(ctx, db, indexName)
		if err != nil {
			return err
		}

		if valid {
			continue
		}

		log.Printf("v1_0_162: building %s (partition %d/%d)", indexName, i+1, len(partitions))
		start := time.Now()

		if _, err := conn.ExecContext(ctx, fmt.Sprintf(`DROP INDEX CONCURRENTLY IF EXISTS %s;`, quoteIdent(indexName))); err != nil {
			return fmt.Errorf("failed to drop invalid index %s: %w", indexName, err)
		}

		stmt := fmt.Sprintf(
			`CREATE INDEX CONCURRENTLY %s ON %s %s;`,
			quoteIdent(indexName),
			quoteIdent(partition),
			v10162Columns,
		)

		if _, err := conn.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to create index concurrently on %s: %w", partition, err)
		}

		log.Printf("v1_0_162: built %s in %s", indexName, time.Since(start).Round(time.Second))
	}

	// The parent create takes SHARE locks on v1_runs_olap, so waiting behind a long transaction would queue every
	// writer; fail instead, a re-run only retries the attach.
	if _, err := conn.ExecContext(ctx, `SET lock_timeout = '5s';`); err != nil {
		return fmt.Errorf("failed to set lock_timeout: %w", err)
	}

	stmt := fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s ON v1_runs_olap %s;`, quoteIdent(v10162ParentIndex), v10162Columns)

	if _, err := conn.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("failed to create index on %s: %w", "v1_runs_olap", err)
	}

	return nil
}
