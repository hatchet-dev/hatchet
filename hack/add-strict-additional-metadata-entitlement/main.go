package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	dsn := flag.String("database-url", "", "Postgres URL for the core engine database")
	tenantID := flag.String("tenant-id", "", "Optional tenant UUID to enable strict additional_metadata filters for")
	flag.Parse()

	if *dsn == "" {
		*dsn = os.Getenv("DATABASE_URL")
	}
	if *dsn == "" {
		return fmt.Errorf("set -database-url or DATABASE_URL")
	}

	db, err := sql.Open("pgx", *dsn)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	ctx := context.Background()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}

	if _, err := db.ExecContext(ctx, `
ALTER TABLE tenant_entitlement
ADD COLUMN IF NOT EXISTS strict_additional_metadata_filters BOOLEAN NOT NULL DEFAULT FALSE
`); err != nil {
		return fmt.Errorf("add strict_additional_metadata_filters column: %w", err)
	}

	log.Print("ensured tenant_entitlement.strict_additional_metadata_filters exists")

	if *tenantID == "" {
		return nil
	}

	if _, err := db.ExecContext(ctx, `
INSERT INTO tenant_entitlement (tenant_id, strict_additional_metadata_filters)
VALUES ($1::uuid, TRUE)
ON CONFLICT (tenant_id) DO UPDATE
SET
    strict_additional_metadata_filters = TRUE,
    updated_at = NOW()
`, *tenantID); err != nil {
		return fmt.Errorf("enable strict additional_metadata filters for tenant %s: %w", *tenantID, err)
	}

	log.Printf("enabled strict additional_metadata filters for tenant %s", *tenantID)

	return nil
}
