package migrations

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(upV10136, downV10136)
}

func v10136IndexName(partition string) string {
	return fmt.Sprintf("%s_external_id_seen_at", partition)
}

func indexIsValid(ctx context.Context, db *sql.DB, indexName string) (bool, error) {
	var isValid bool

	err := db.QueryRowContext(
		ctx,
		`
			SELECT i.indisvalid
			FROM pg_index i
			JOIN pg_class c ON c.oid = i.indexrelid
			WHERE c.relname = $1
		`,
		indexName,
	).Scan(&isValid)

	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("failed to check index validity for %s: %w", indexName, err)
	}

	return isValid, nil
}

func upV10136(ctx context.Context, db *sql.DB) error {
	table := "v1_event"

	partitions, err := listLeafPartitions(ctx, db, table, 1)

	if err != nil {
		return err
	}

	for _, partition := range partitions {
		indexName := v10136IndexName(partition)
		valid, err := indexIsValid(ctx, db, indexName)

		if err != nil {
			return err
		}

		if valid {
			continue
		}

		if _, err := db.ExecContext(ctx, fmt.Sprintf(`DROP INDEX CONCURRENTLY IF EXISTS %s;`, quoteIdent(indexName))); err != nil {
			return fmt.Errorf("failed to drop stale index %s: %w", indexName, err)
		}

		// #nosec G201 -- identifiers are quoted and derived from internal migration logic, not user input
		dedupeStmt := fmt.Sprintf(
			`
			DELETE FROM %s a
			USING %s b
			WHERE
				a.external_id = b.external_id
				AND a.seen_at = b.seen_at
				AND a.tenant_id = b.tenant_id
				AND a.id > b.id;
			`,
			quoteIdent(partition),
			quoteIdent(partition),
		)

		if _, err := db.ExecContext(ctx, dedupeStmt); err != nil {
			return fmt.Errorf("failed to deduplicate rows on %s: %w", table, err)
		}

		// #nosec G201 -- identifiers are quoted and derived from internal migration logic, not user input
		buildStmt := fmt.Sprintf(
			`CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS %s ON %s (external_id, seen_at);`,
			quoteIdent(indexName),
			quoteIdent(partition),
		)

		if _, err := db.ExecContext(ctx, buildStmt); err != nil {
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
