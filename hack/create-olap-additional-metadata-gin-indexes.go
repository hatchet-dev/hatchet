package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var tables = []string{"v1_runs_olap", "v1_tasks_olap"}

type indexJob struct {
	table     string
	partition string
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	dsn := flag.String("database-url", "", "Postgres URL for the Timescale OLAP database")
	parallel := flag.Int("parallel", 1, "Number of partition indexes to build concurrently")
	flag.Parse()

	if *parallel < 1 {
		return fmt.Errorf("-parallel must be at least 1")
	}

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
	db.SetMaxOpenConns(*parallel + 2)
	db.SetMaxIdleConns(*parallel + 2)

	ctx := context.Background()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}

	jobs, err := listIndexJobs(ctx, db)
	if err != nil {
		return err
	}

	log.Printf("creating %d partition indexes with parallel=%d", len(jobs), *parallel)

	if err := createPartitionIndexes(ctx, db, jobs, *parallel); err != nil {
		return err
	}

	for _, table := range tables {
		if err := createParentIndex(ctx, db, table); err != nil {
			return fmt.Errorf("create parent index for %s: %w", table, err)
		}
	}

	return nil
}

func listIndexJobs(ctx context.Context, db *sql.DB) ([]indexJob, error) {
	var jobs []indexJob

	for _, table := range tables {
		partitions, err := listLeafPartitions(ctx, db, table)
		if err != nil {
			return nil, err
		}

		for _, partition := range partitions {
			jobs = append(jobs, indexJob{
				table:     table,
				partition: partition,
			})
		}
	}

	return jobs, nil
}

func createPartitionIndexes(ctx context.Context, db *sql.DB, jobs []indexJob, parallel int) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	jobCh := make(chan indexJob)
	errCh := make(chan error, 1)
	var wg sync.WaitGroup

	for workerID := 1; workerID <= parallel; workerID++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for job := range jobCh {
				if err := createPartitionIndex(ctx, db, workerID, job); err != nil {
					select {
					case errCh <- err:
						cancel()
					default:
					}
					return
				}
			}
		}(workerID)
	}

	for _, job := range jobs {
		select {
		case jobCh <- job:
		case err := <-errCh:
			close(jobCh)
			wg.Wait()
			return err
		case <-ctx.Done():
			close(jobCh)
			wg.Wait()
			return ctx.Err()
		}
	}

	close(jobCh)
	wg.Wait()

	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}

func createPartitionIndex(ctx context.Context, db *sql.DB, workerID int, job indexJob) error {
	indexName := ginIndexName(job.partition)
	stmt := fmt.Sprintf(
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS %s ON %s USING gin (additional_metadata jsonb_path_ops);`,
		quoteIdent(indexName),
		quoteIdent(job.partition),
	)

	log.Printf("worker %d creating %s (%s)", workerID, indexName, job.table)
	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("create partition index %s: %w", indexName, err)
	}

	return nil
}

func createParentIndex(ctx context.Context, db *sql.DB, table string) error {
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
