package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var tables = []string{"v1_runs_olap", "v1_tasks_olap"}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	dsn := flag.String("database-url", "", "Postgres URL for the Timescale OLAP database")
	flag.Parse()

	if *dsn == "" {
		*dsn = os.Getenv("CLOUD_TIMESCALE_V1_DATABASE_URL")
	}
	if *dsn == "" {
		return fmt.Errorf("set -database-url or CLOUD_TIMESCALE_V1_DATABASE_URL")
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

	for _, table := range tables {
		if err := createIndexesForTable(ctx, db, table); err != nil {
			return fmt.Errorf("create indexes for %s: %w", table, err)
		}
	}

	return nil
}

func createIndexesForTable(ctx context.Context, db *sql.DB, table string) error {
	partitions, err := listLeafPartitions(ctx, db, table)
	if err != nil {
		return err
	}

	for _, partition := range partitions {
		indexName := ginIndexName(partition)
		stmt := fmt.Sprintf(
			`CREATE INDEX CONCURRENTLY IF NOT EXISTS %s ON %s USING gin (additional_metadata jsonb_path_ops);`,
			quoteIdent(indexName),
			quoteIdent(partition),
		)

		log.Printf("creating %s", indexName)
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("create partition index %s: %w", indexName, err)
		}
	}

	// Attach the equivalent child indexes to the parent and make future
	// partitions inherit this index via CREATE TABLE ... LIKE INCLUDING INDEXES.
	parentIndexName := ginIndexName(table)
	stmt := fmt.Sprintf(
		`CREATE INDEX IF NOT EXISTS %s ON %s USING gin (additional_metadata jsonb_path_ops);`,
		quoteIdent(parentIndexName),
		quoteIdent(table),
	)

	log.Printf("creating parent index %s", parentIndexName)
	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("create parent index %s: %w", parentIndexName, err)
	}

	return nil
}

func listLeafPartitions(ctx context.Context, db *sql.DB, parentTable string) ([]string, error) {
	rows, err := db.QueryContext(ctx, `
SELECT relid::regclass::text AS partition
FROM pg_partition_tree($1::regclass)
WHERE isleaf
  AND relid <> $1::regclass
ORDER BY 1
`, parentTable)
	if err != nil {
		return nil, fmt.Errorf("list partitions for %s: %w", parentTable, err)
	}
	defer rows.Close()

	var partitions []string
	for rows.Next() {
		var partition string
		if err := rows.Scan(&partition); err != nil {
			return nil, fmt.Errorf("scan partition: %w", err)
		}
		partitions = append(partitions, partition)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate partitions: %w", err)
	}

	return partitions, nil
}

func ginIndexName(table string) string {
	return fmt.Sprintf("ix_%s_additional_metadata_gin", strings.ReplaceAll(table, ".", "_"))
}

func quoteIdent(ident string) string {
	return `"` + strings.ReplaceAll(ident, `"`, `""`) + `"`
}
