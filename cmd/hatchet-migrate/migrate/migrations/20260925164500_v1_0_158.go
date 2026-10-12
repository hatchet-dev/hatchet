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
	goose.AddMigrationNoTxContext(upV10158, downV10158)
}

const v10158Columns = "(tenant_id, inserted_at DESC, readable_status, workflow_id)"

func v10158IndexName(table string) string {
	return fmt.Sprintf("ix_%s_tenant_ins_at_status_wf", table)
}

func upV10158(ctx context.Context, db *sql.DB) error {
	partitions, err := listLeafPartitions(ctx, db, "v1_runs_olap", 1)
	if err != nil {
		return err
	}

	// A per-partition build can run for many minutes, so it must not inherit a server or
	// role statement_timeout. SET is per session, hence one pinned connection for the builds.
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, `SET statement_timeout = 0;`); err != nil {
		return fmt.Errorf("failed to disable statement_timeout: %w", err)
	}

	for i, partition := range partitions {
		indexName := v10158IndexName(partition)

		valid, err := indexIsValid(ctx, db, indexName)
		if err != nil {
			return err
		}

		if valid {
			continue
		}

		log.Printf("v1_0_158: building %s (partition %d/%d)", indexName, i+1, len(partitions))
		start := time.Now()

		if _, err := conn.ExecContext(ctx, fmt.Sprintf(`DROP INDEX CONCURRENTLY IF EXISTS %s;`, quoteIdent(indexName))); err != nil {
			return fmt.Errorf("failed to drop invalid index %s: %w", indexName, err)
		}

		stmt := fmt.Sprintf(
			`CREATE INDEX CONCURRENTLY %s ON %s %s;`,
			quoteIdent(indexName),
			quoteIdent(partition),
			v10158Columns,
		)

		if _, err := conn.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to create index concurrently on %s: %w", partition, err)
		}

		log.Printf("v1_0_158: built %s in %s", indexName, time.Since(start).Round(time.Second))
	}

	// The parent create takes SHARE locks on v1_runs_olap, so waiting behind a long
	// transaction would queue every writer; fail instead, a re-run only retries the attach.
	if _, err := conn.ExecContext(ctx, `SET lock_timeout = '5s';`); err != nil {
		return fmt.Errorf("failed to set lock_timeout: %w", err)
	}

	stmt := fmt.Sprintf(
		`CREATE INDEX IF NOT EXISTS %s ON v1_runs_olap %s;`,
		quoteIdent(v10158IndexName("v1_runs_olap")),
		v10158Columns,
	)

	if _, err := conn.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("failed to create index on %s: %w", "v1_runs_olap", err)
	}

	return nil
}

// downV10158 drops the parent index, which cascades to all child partition indexes.
func downV10158(ctx context.Context, db *sql.DB) error {
	stmt := fmt.Sprintf(`DROP INDEX IF EXISTS %s;`, quoteIdent(v10158IndexName("v1_runs_olap")))

	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("failed to drop index on %s: %w", "v1_runs_olap", err)
	}

	return nil
}
