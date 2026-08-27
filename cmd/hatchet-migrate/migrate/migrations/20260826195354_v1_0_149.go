package migrations

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(upV10149, downV10149)
}

const statusesOlapTenantInsertedAtIndexName = "v1_statuses_olap_tenant_inserted_at_idx"

func upV10149(ctx context.Context, db *sql.DB) error {
	// A previously failed concurrent build can leave the index behind in an INVALID
	// state, which IF NOT EXISTS would silently keep. Drop it so the create below
	// builds a usable index.
	exists, valid, err := statusesOlapIndexState(ctx, db)
	if err != nil {
		return err
	}

	if exists {
		if valid {
			return nil
		}

		log.Printf("dropping invalid index %s left behind by a failed build", statusesOlapTenantInsertedAtIndexName)

		if _, err := db.ExecContext(ctx, "DROP INDEX "+statusesOlapTenantInsertedAtIndexName); err != nil {
			return fmt.Errorf("failed to drop invalid index %s: %w", statusesOlapTenantInsertedAtIndexName, err)
		}
	}

	isHypertable, err := statusesOlapIsHypertable(ctx, db)
	if err != nil {
		return err
	}

	if isHypertable {
		// TimescaleDB does not support CREATE INDEX CONCURRENTLY on hypertables;
		// transaction_per_chunk builds the index one chunk at a time without
		// holding a lock on the whole hypertable for the full build.
		_, err = db.ExecContext(ctx, fmt.Sprintf(
			"CREATE INDEX IF NOT EXISTS %s ON v1_statuses_olap (tenant_id, inserted_at) WITH (timescaledb.transaction_per_chunk)",
			statusesOlapTenantInsertedAtIndexName,
		))

		if err != nil {
			return fmt.Errorf("failed to create index %s on hypertable: %w", statusesOlapTenantInsertedAtIndexName, err)
		}

		return nil
	}

	_, err = db.ExecContext(ctx, fmt.Sprintf(
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS %s ON v1_statuses_olap (tenant_id, inserted_at)",
		statusesOlapTenantInsertedAtIndexName,
	))

	if err != nil {
		return fmt.Errorf("failed to create index %s: %w", statusesOlapTenantInsertedAtIndexName, err)
	}

	return nil
}

func downV10149(ctx context.Context, db *sql.DB) error {
	isHypertable, err := statusesOlapIsHypertable(ctx, db)
	if err != nil {
		return err
	}

	if isHypertable {
		// DROP INDEX CONCURRENTLY is likewise unsupported on hypertables
		_, err = db.ExecContext(ctx, "DROP INDEX IF EXISTS "+statusesOlapTenantInsertedAtIndexName)
		return err
	}

	_, err = db.ExecContext(ctx, "DROP INDEX CONCURRENTLY IF EXISTS "+statusesOlapTenantInsertedAtIndexName)

	return err
}

// statusesOlapIndexState reports whether the index exists in the current schema and,
// if so, whether it is valid.
func statusesOlapIndexState(ctx context.Context, db *sql.DB) (exists bool, valid bool, err error) {
	err = db.QueryRowContext(ctx, `
		SELECT i.indisvalid
		FROM pg_class c
		JOIN pg_index i ON i.indexrelid = c.oid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relname = $1 AND n.nspname = current_schema()
	`, statusesOlapTenantInsertedAtIndexName).Scan(&valid)

	if errors.Is(err, sql.ErrNoRows) {
		return false, false, nil
	}

	if err != nil {
		return false, false, fmt.Errorf("failed to check state of index %s: %w", statusesOlapTenantInsertedAtIndexName, err)
	}

	return true, valid, nil
}

func statusesOlapIsHypertable(ctx context.Context, db *sql.DB) (bool, error) {
	var extensionInstalled bool

	err := db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb')`).Scan(&extensionInstalled)
	if err != nil {
		return false, fmt.Errorf("failed to check for timescaledb extension: %w", err)
	}

	if !extensionInstalled {
		return false, nil
	}

	var isHypertable bool

	err = db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM timescaledb_information.hypertables
			WHERE hypertable_name = 'v1_statuses_olap'
		)
	`).Scan(&isHypertable)

	if err != nil {
		return false, fmt.Errorf("failed to check whether v1_statuses_olap is a hypertable: %w", err)
	}

	return isHypertable, nil
}
