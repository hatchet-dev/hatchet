package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(upV10126, downV10126)
}

func upV10126(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `
		ALTER TABLE tenant_entitlement
		ADD COLUMN IF NOT EXISTS strict_additional_metadata_filters BOOLEAN NOT NULL DEFAULT FALSE
	`); err != nil {
		return fmt.Errorf("add strict_additional_metadata_filters to tenant_entitlement: %w", err)
	}

	return nil
}

func downV10126(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `
		ALTER TABLE tenant_entitlement
		DROP COLUMN IF EXISTS strict_additional_metadata_filters
	`); err != nil {
		return fmt.Errorf("drop strict_additional_metadata_filters from tenant_entitlement: %w", err)
	}

	return nil
}
