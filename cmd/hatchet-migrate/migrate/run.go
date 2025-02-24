package migrate

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
	"github.com/sethvargo/go-retry"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func RunMigrations(ctx context.Context) {
	var db *sql.DB

	retryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err := retry.Do(retryCtx, retry.NewConstant(1*time.Second), func(ctx context.Context) error {
		var err error
		db, err = goose.OpenDBWithDriver("postgres", os.Getenv("DATABASE_URL"))
		if err != nil {
			return retry.RetryableError(fmt.Errorf("failed to open DB: %w", err))
		}

		return nil
	})

	if err != nil {
		log.Fatalf("goose: failed to open DB: %v", err)
	}

	conn, err := db.Conn(ctx)

	if err != nil {
		log.Fatalf("goose: failed to open DB connection: %v", err)
	}

	defer func() {
		if err := conn.Close(); err != nil {
			log.Fatalf("goose: failed to close DB connection: %v", err)
		}

		if err := db.Close(); err != nil {
			log.Fatalf("goose: failed to close DB: %v", err)
		}
	}()

	if err != nil {
		log.Fatalf("goose: failed to open DB connection: %v", err)
	}

	locker, err := lock.NewPostgresSessionLocker()

	if err != nil {
		log.Fatalf("goose: failed to create locker: %v", err)
	}

	err = locker.SessionLock(ctx, conn)

	if err != nil {
		log.Fatalf("goose: failed to lock session: %v", err)
	}

	// Check whether the goose migrations table exists.
	var gooseExists bool
	{
		query := TableExists("goose_db_version")
		err = conn.QueryRowContext(ctx, query).Scan(&gooseExists)
		if err != nil {
			log.Fatalf("goose: failed to check goose migrations table existence: %v", err)
		}
	}

	// If the goose migrations table doesn't exist, create it and set a baseline.
	if !gooseExists {
		// Create goose migrations table.
		createTableSQL := CreateTable("goose_db_version")
		_, err = conn.ExecContext(ctx, createTableSQL)
		if err != nil {
			log.Fatalf("goose: failed to create goose migrations table: %v", err)
		}

		// Insert a 0 version.
		insertQuery := InsertVersion("goose_db_version")
		_, err = conn.ExecContext(ctx, insertQuery, 0, true)

		if err != nil {
			log.Fatalf("goose: failed to insert baseline migration: %v", err)
		}

		// Determine baseline version from atlas or prisma migrations.
		var baseline string

		// 1. Check that the atlas_schema_revisions.atlas_schema_revisions table exists.
		var atlasExists bool
		atlasExistQuery := "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'atlas_schema_revisions' AND table_name = 'atlas_schema_revisions')"
		err = conn.QueryRowContext(ctx, atlasExistQuery).Scan(&atlasExists)
		if err != nil {
			log.Fatalf("goose: error checking atlas_schema_revisions existence: %v", err)
		}

		// 2. If it does, check for the latest migration in the atlas schema.
		if atlasExists {
			var version string
			atlasLatestQuery := "SELECT version FROM atlas_schema_revisions.atlas_schema_revisions ORDER BY version DESC LIMIT 1"
			err = conn.QueryRowContext(ctx, atlasLatestQuery).Scan(&version)
			if err == nil {
				baseline = version
			}
		}

		// 3. If not found, check whether the _prisma_migrations table exists.
		if baseline == "" {
			var prismaExists bool
			prismaExistQuery := "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = '_prisma_migrations')"
			err = conn.QueryRowContext(ctx, prismaExistQuery).Scan(&prismaExists)
			if err != nil {
				log.Fatalf("goose: error checking _prisma_migrations existence: %v", err)
			}

			// 4. If it does, check for the latest migration in the prisma schema.
			if prismaExists {
				var migrationName string
				prismaLatestQuery := "SELECT migration_name FROM _prisma_migrations ORDER BY started_at DESC LIMIT 1"
				err = conn.QueryRowContext(ctx, prismaLatestQuery).Scan(&migrationName)
				if err == nil {
					baseline = migrationName
				}
			}
		}

		// If a baseline version was found, check for a match in the ./migrations directory.
		if baseline != "" {
			fsys, err := fs.Sub(embedMigrations, "migrations")
			if err != nil {
				log.Fatalf("goose: failed to create sub filesystem: %v", err)
			}
			entries, err := fs.ReadDir(fsys, ".")
			if err != nil {
				log.Fatalf("goose: failed to read migrations directory: %v", err)
			}

			type migration struct {
				version  string
				filename string
			}
			var migrations []migration
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				name := entry.Name()
				parts := strings.SplitN(name, "__", 2)
				if len(parts) != 2 {
					continue
				}
				version := parts[0]
				if version <= baseline {
					migrations = append(migrations, migration{version: version, filename: name})
				}
			}

			sort.Slice(migrations, func(i, j int) bool {
				return migrations[i].version < migrations[j].version
			})

			for _, m := range migrations {
				insertQuery := InsertVersion("goose_db_version")
				v, err := strconv.ParseInt(m.version, 10, 64)
				if err != nil {
					log.Fatalf("goose: invalid migration version %s: %v", m.version, err)
				}
				_, err = conn.ExecContext(ctx, insertQuery, v, true)
				if err != nil {
					log.Fatalf("goose: failed to insert baseline migration %s: %v", m.filename, err)
				}
			}
		}
	}

	err = locker.SessionUnlock(ctx, conn)

	if err != nil {
		log.Fatalf("goose: failed to unlock session: %v", err)
	}

	// decouple from existing structure
	fsys, err := fs.Sub(embedMigrations, "migrations")
	if err != nil {
		log.Fatalf("goose: failed to create sub filesystem: %v", err)
	}
	goose.SetBaseFS(fsys)

	err = goose.Up(db, ".")
	if err != nil {
		log.Fatalf("goose: failed to apply migrations: %v", err)
	}
}

// Copied from https://github.com/pressly/goose/blob/6a70e744c8eb2dc4bb90ba641cb03b42d8eef6cd/internal/dialect/dialectquery/postgres.go
func CreateTable(tableName string) string {
	q := `CREATE TABLE %s (
		id integer PRIMARY KEY GENERATED BY DEFAULT AS IDENTITY,
		version_id bigint NOT NULL,
		is_applied boolean NOT NULL,
		tstamp timestamp NOT NULL DEFAULT now()
	)`
	return fmt.Sprintf(q, tableName)
}

func InsertVersion(tableName string) string {
	q := `INSERT INTO %s (version_id, is_applied) VALUES ($1, $2)`
	return fmt.Sprintf(q, tableName)
}

func DeleteVersion(tableName string) string {
	q := `DELETE FROM %s WHERE version_id=$1`
	return fmt.Sprintf(q, tableName)
}

func GetMigrationByVersion(tableName string) string {
	q := `SELECT tstamp, is_applied FROM %s WHERE version_id=$1 ORDER BY tstamp DESC LIMIT 1`
	return fmt.Sprintf(q, tableName)
}

func ListMigrations(tableName string) string {
	q := `SELECT version_id, is_applied from %s ORDER BY id DESC`
	return fmt.Sprintf(q, tableName)
}

func GetLatestVersion(tableName string) string {
	q := `SELECT max(version_id) FROM %s`
	return fmt.Sprintf(q, tableName)
}

func TableExists(tableName string) string {
	schemaName, table := parseTableIdentifier(tableName)
	if schemaName != "" {
		q := `SELECT EXISTS ( SELECT 1 FROM pg_tables WHERE schemaname = '%s' AND tablename = '%s' )`
		return fmt.Sprintf(q, schemaName, table)
	}
	q := `SELECT EXISTS ( SELECT 1 FROM pg_tables WHERE (current_schema() IS NULL OR schemaname = current_schema()) AND tablename = '%s' )`
	return fmt.Sprintf(q, table)
}

func parseTableIdentifier(name string) (schema, table string) {
	schema, table, found := strings.Cut(name, ".")
	if !found {
		return "", name
	}
	return schema, table
}
