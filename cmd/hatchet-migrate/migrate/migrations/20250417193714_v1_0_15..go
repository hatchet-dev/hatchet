package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(upCreateIndexOnPartitions, downDropIndexFromPartitions)
}

func createIndexOnPartitions(ctx context.Context, db *sql.DB, tableName string, indexName string, indexColumns string) error {
	// Query to list all partitions for the table
	rows, err := db.QueryContext(ctx,
		fmt.Sprintf(`SELECT partition_name FROM get_v1_partitions_before_date('%s', '2099-12-31')`, tableName))
	if err != nil {
		return err
	}
	defer rows.Close()

	fmt.Printf("Creating indexes on partitions for %s:\n", tableName)

	// Store all partition names
	var partitions []string
	for rows.Next() {
		var partitionName string
		if err := rows.Scan(&partitionName); err != nil {
			return err
		}
		partitions = append(partitions, partitionName)
	}

	// Step 1: Create an invalid index on the parent table
	parentIndexQuery := fmt.Sprintf(
		`CREATE INDEX IF NOT EXISTS %s ON ONLY %s %s`,
		indexName,
		tableName,
		indexColumns,
	)

	fmt.Printf("Step 1: Creating parent index on table: %s\n", tableName)
	_, err = db.ExecContext(ctx, parentIndexQuery)
	if err != nil {
		return fmt.Errorf("failed to create parent index on table %s: %w", tableName, err)
	}

	// Step 2: Create indexes on each partition (without CONCURRENTLY) and attach to parent index
	for _, partitionName := range partitions {
		fmt.Printf("Creating index on partition: %s\n", partitionName)

		// Create a separate connection for each index creation
		connDB, err := db.Conn(ctx)
		if err != nil {
			return fmt.Errorf("failed to open new connection for %s: %w", partitionName, err)
		}

		// Create the index on the partition (without CONCURRENTLY)
		partitionIndexName := fmt.Sprintf("idx_%s_%s", partitionName, indexName)
		createIndexQuery := fmt.Sprintf(
			`CREATE INDEX CONCURRENTLY IF NOT EXISTS %s ON ONLY %s %s`,
			partitionIndexName,
			partitionName,
			indexColumns,
		)

		_, err = connDB.ExecContext(ctx, createIndexQuery)
		if err != nil {
			connDB.Close()
			return fmt.Errorf("failed to create index on partition %s: %w", partitionName, err)
		}

		// Attach the partition index to the parent index
		attachQuery := fmt.Sprintf(
			`ALTER INDEX %s ATTACH PARTITION %s`,
			indexName,
			partitionIndexName,
		)

		_, err = connDB.ExecContext(ctx, attachQuery)
		if err != nil {
			connDB.Close()
			return fmt.Errorf("failed to attach partition index %s to parent: %w", partitionIndexName, err)
		}

		connDB.Close()
		fmt.Printf("Successfully created and attached index on partition: %s\n", partitionName)

		// Add a small delay to prevent overwhelming the database
		time.Sleep(100 * time.Millisecond)
	}

	// Step 3: Verify that all indexes are valid
	fmt.Println("Verifying indexes validity:")

	verifyQuery := `SELECT count(*) FROM pg_index WHERE indisvalid = false`
	var invalidCount int
	err = db.QueryRowContext(ctx, verifyQuery).Scan(&invalidCount)
	if err != nil {
		return fmt.Errorf("failed to verify index validity: %w", err)
	}

	if invalidCount > 0 {
		fmt.Printf("Warning: Found %d invalid indexes. Please check your script for mistakes.\n", invalidCount)
	} else {
		fmt.Println("All indexes are valid.")
	}

	fmt.Println("All indexes created successfully")
	return nil
}

func upCreateIndexOnPartitions(ctx context.Context, db *sql.DB) error {
	err := createIndexOnPartitions(ctx, db, "v1_runs_olap", "external_id", "(parent_task_external_id) WHERE parent_task_external_id IS NOT NULL")
	if err != nil {
		return err
	}

	err = createIndexOnPartitions(ctx, db, "v1_statuses_olap", "query_optim", "(tenant_id, inserted_at, workflow_id)")
	if err != nil {
		return err
	}

	return nil
}

func downDropIndexFromPartitions(ctx context.Context, db *sql.DB) error {
	// Query to list all partitions for v1_runs_olap
	rows, err := db.QueryContext(ctx,
		`SELECT partition_name FROM get_v1_partitions_before_date('v1_runs_olap', '2099-12-31')`)
	if err != nil {
		return err
	}
	defer rows.Close()

	fmt.Println("Dropping indexes on partitions for v1_runs_olap:")

	// Store all partition names
	var partitions []string
	for rows.Next() {
		var partitionName string
		if err := rows.Scan(&partitionName); err != nil {
			return err
		}
		partitions = append(partitions, partitionName)
	}

	// Drop indexes one by one, not in a transaction
	for _, partitionName := range partitions {
		fmt.Printf("Dropping index on partition: %s\n", partitionName)

		// Create a separate connection for each index drop to ensure no transaction
		connDB, err := db.Conn(ctx)
		if err != nil {
			return fmt.Errorf("failed to open new connection for %s: %w", partitionName, err)
		}

		// Drop the index on the partition
		indexName := fmt.Sprintf("idx_%s_external_id", partitionName)
		dropQuery := fmt.Sprintf("DROP INDEX IF EXISTS %s", indexName)

		// Execute the drop command
		_, err = connDB.ExecContext(ctx, dropQuery)
		if err != nil {
			connDB.Close()
			return fmt.Errorf("failed to drop index on partition %s: %w", partitionName, err)
		}

		connDB.Close()
		fmt.Printf("Successfully dropped index on partition: %s\n", partitionName)

		// Add a small delay to prevent overwhelming the database
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("All indexes dropped successfully")
	return nil
}
