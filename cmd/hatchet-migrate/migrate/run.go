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

	_ "github.com/hatchet-dev/hatchet/cmd/hatchet-migrate/migrate/migrations" // register go migrations
	"github.com/hatchet-dev/hatchet/pkg/migratediag"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

type runMigrationsOpt struct {
	upToPenultimate bool
}

type RunMigrationsOpt func(*runMigrationsOpt)

func WithUpToPenultimate() RunMigrationsOpt {
	return func(o *runMigrationsOpt) {
		o.upToPenultimate = true
	}
}

func RunMigrations(ctx context.Context, opts ...RunMigrationsOpt) {
	if err := runMigrationsImpl(ctx, opts...); err != nil {
		log.Fatal(err)
	}
}

func runMigrationsImpl(ctx context.Context, opts ...RunMigrationsOpt) error {
	// Set default options
	options := &runMigrationsOpt{}

	for _, opt := range opts {
		opt(options)
	}

	const (
		databaseEnvVar = "DATABASE_URL"
		phaseName      = "oss"
	)

	rawURL := os.Getenv(databaseEnvVar)
	if rawURL == "" {
		return migratediag.MissingEnvError(databaseEnvVar, phaseName)
	}

	dsn := migratediag.SummarizePostgresDSN(rawURL)

	var db *sql.DB
	var conn *sql.Conn

	retryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)

	err := retry.Do(retryCtx, retry.NewConstant(1*time.Second), func(ctx context.Context) error {
		var err error
		if db == nil {
			db, err = goose.OpenDBWithDriver("postgres", rawURL)

			if err != nil {
				return retry.RetryableError(fmt.Errorf("failed to open DB: %w", err))
			}
		}

		conn, err = db.Conn(ctx)

		if err != nil {
			return retry.RetryableError(fmt.Errorf("failed to open DB connection: %w", err))
		}

		return nil
	})

	cancel()

	if err != nil {
		stage := "connect"
		if db == nil {
			stage = "open"
		}
		return migratediag.PhaseError(databaseEnvVar, phaseName, dsn, stage, err)
	}

	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("%v", migratediag.PhaseError(databaseEnvVar, phaseName, dsn, "close DB connection", err))
		}

		if err := db.Close(); err != nil {
			log.Printf("%v", migratediag.PhaseError(databaseEnvVar, phaseName, dsn, "close DB", err))
		}
	}()

	locker, err := lock.NewPostgresSessionLocker()

	if err != nil {
		return migratediag.PhaseError(databaseEnvVar, phaseName, dsn, "create session locker", err)
	}

	err = locker.SessionLock(ctx, conn)

	if err != nil {
		return migratediag.PhaseError(databaseEnvVar, phaseName, dsn, "session lock", err)
	}

	// Check whether the goose migrations table exists.
	var gooseExists bool
	{
		query := TableExists("goose_db_version")
		err = conn.QueryRowContext(ctx, query).Scan(&gooseExists)
		if err != nil {
			return migratediag.PhaseError(databaseEnvVar, phaseName, dsn, "check goose_db_version existence", err)
		}
	}

	// If the goose migrations table doesn't exist, create it and set a baseline.
	if !gooseExists {
		// Create goose migrations table.
		createTableSQL := CreateTable("goose_db_version")
		_, err = conn.ExecContext(ctx, createTableSQL)
		if err != nil {
			return migratediag.PhaseError(databaseEnvVar, phaseName, dsn, "create goose_db_version table", err)
		}

		// Insert a 0 version.
		insertQuery := InsertVersion("goose_db_version")
		_, err = conn.ExecContext(ctx, insertQuery, 0, true)

		if err != nil {
			return migratediag.PhaseError(databaseEnvVar, phaseName, dsn, "insert baseline version 0", err)
		}

		// Determine baseline version from atlas or prisma migrations.
		var baseline string

		// 1. Check that the atlas_schema_revisions.atlas_schema_revisions table exists.
		var atlasExists bool
		atlasExistQuery := "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'atlas_schema_revisions' AND table_name = 'atlas_schema_revisions')"
		err = conn.QueryRowContext(ctx, atlasExistQuery).Scan(&atlasExists)
		if err != nil {
			return migratediag.PhaseError(databaseEnvVar, phaseName, dsn, "check atlas_schema_revisions existence", err)
		}

		fmt.Printf("Does existing atlas schema exist? %v\n", atlasExists)

		// 2. If it does, check for the latest migration in the atlas schema.
		if atlasExists {
			var version string
			atlasLatestQuery := "SELECT version FROM atlas_schema_revisions.atlas_schema_revisions ORDER BY version DESC LIMIT 1"
			err = conn.QueryRowContext(ctx, atlasLatestQuery).Scan(&version)
			if err == nil {
				baseline = version
				fmt.Printf("Baseline version from atlas: %s\n", baseline)
			}
		}

		// 3. If not found, check whether the _prisma_migrations table exists.
		if baseline == "" {
			var prismaExists bool
			prismaExistQuery := "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = '_prisma_migrations')"
			err = conn.QueryRowContext(ctx, prismaExistQuery).Scan(&prismaExists)
			if err != nil {
				return migratediag.PhaseError(databaseEnvVar, phaseName, dsn, "check _prisma_migrations existence", err)
			}

			fmt.Printf("Does existing prisma schema exist? %v\n", prismaExists)

			// 4. If it does, check for the latest migration in the prisma schema.
			if prismaExists {
				var migrationName string
				prismaLatestQuery := "SELECT migration_name FROM _prisma_migrations ORDER BY started_at DESC LIMIT 1"
				err = conn.QueryRowContext(ctx, prismaLatestQuery).Scan(&migrationName)
				if err == nil {
					baseline = migrationName
					fmt.Printf("Baseline version from prisma: %s\n", baseline)
				}
			}
		}

		// If a baseline version was found, check for a match in the ./migrations directory.
		if baseline != "" {
			fsys, err := fs.Sub(embedMigrations, "migrations")
			if err != nil {
				return migratediag.PhaseError(databaseEnvVar, phaseName, dsn, "create migrations sub filesystem", err)
			}
			entries, err := fs.ReadDir(fsys, ".")
			if err != nil {
				return migratediag.PhaseError(databaseEnvVar, phaseName, dsn, "read migrations directory", err)
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
				parts := strings.SplitN(name, "_", 2)
				if len(parts) < 2 {
					continue
				}
				version := parts[0]
				if version <= baseline {
					fmt.Printf("Including version %s from %s\n", version, name)
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
					return migratediag.PhaseError(databaseEnvVar, phaseName, dsn, "parse baseline migration version", fmt.Errorf("invalid migration version %s: %w", m.version, err))
				}
				_, err = conn.ExecContext(ctx, insertQuery, v, true)
				if err != nil {
					return migratediag.PhaseError(databaseEnvVar, phaseName, dsn, "insert baseline migration", fmt.Errorf("%s: %w", m.filename, err))
				}
			}
		}
	}

	err = locker.SessionUnlock(ctx, conn)

	if err != nil {
		return migratediag.PhaseError(databaseEnvVar, phaseName, dsn, "session unlock", err)
	}

	// decouple from existing structure
	fsys, err := fs.Sub(embedMigrations, "migrations")
	if err != nil {
		return migratediag.PhaseError(databaseEnvVar, phaseName, dsn, "create migrations sub filesystem", err)
	}
	goose.SetBaseFS(fsys)

	switch {
	case options.upToPenultimate:
		migrations, err := listMigrations()

		if err != nil {
			return migratediag.PhaseError(databaseEnvVar, phaseName, dsn, "list migrations", err)
		}

		if len(migrations) < 2 {
			return migratediag.PhaseError(databaseEnvVar, phaseName, dsn, "select penultimate migration", fmt.Errorf("not enough migrations to roll back to penultimate version"))
		}

		err = goose.UpTo(db, ".", migrations[len(migrations)-2].Version)

		if err != nil {
			return migratediag.PhaseError(databaseEnvVar, phaseName, dsn, "apply migrations up to penultimate version", err)
		}
	default:
		err = goose.Up(db, ".")
		if err != nil {
			return migratediag.PhaseError(databaseEnvVar, phaseName, dsn, "apply migrations", err)
		}
	}

	return nil
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

// listAllDBVersions returns a list of all migrations, ordered ascending.
func listMigrations() (goose.Migrations, error) {
	var (
		minVersion = int64(0)
		maxVersion = int64((1 << 63) - 1)
	)

	return goose.CollectMigrations(".", minVersion, maxVersion)
}

// RunDownMigration runs down migrations to a specific version.
func RunDownMigration(ctx context.Context, targetVersion string) {
	if err := runDownMigrationImpl(ctx, targetVersion); err != nil {
		log.Fatal(err)
	}
}

func runDownMigrationImpl(ctx context.Context, targetVersion string) error {
	const (
		databaseEnvVar = "DATABASE_URL"
		phaseName      = "oss-down"
	)

	rawURL := os.Getenv(databaseEnvVar)
	if rawURL == "" {
		return migratediag.MissingEnvError(databaseEnvVar, phaseName)
	}

	dsn := migratediag.SummarizePostgresDSN(rawURL)

	var db *sql.DB
	var conn *sql.Conn

	retryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)

	err := retry.Do(retryCtx, retry.NewConstant(1*time.Second), func(ctx context.Context) error {
		var err error
		if db == nil {
			db, err = goose.OpenDBWithDriver("postgres", rawURL)

			if err != nil {
				return retry.RetryableError(fmt.Errorf("failed to open DB: %w", err))
			}
		}

		conn, err = db.Conn(ctx)

		if err != nil {
			return retry.RetryableError(fmt.Errorf("failed to open DB connection: %w", err))
		}

		return nil
	})

	cancel()

	if err != nil {
		stage := "connect"
		if db == nil {
			stage = "open"
		}
		return migratediag.PhaseError(databaseEnvVar, phaseName, dsn, stage, err)
	}

	defer func() {
		if conn != nil {
			if err := conn.Close(); err != nil {
				log.Printf("%v", migratediag.PhaseError(databaseEnvVar, phaseName, dsn, "close DB connection", err))
			}
		}

		if db != nil {
			if err := db.Close(); err != nil {
				log.Printf("%v", migratediag.PhaseError(databaseEnvVar, phaseName, dsn, "close DB", err))
			}
		}
	}()

	locker, err := lock.NewPostgresSessionLocker()

	if err != nil {
		return migratediag.PhaseError(databaseEnvVar, phaseName, dsn, "create session locker", err)
	}

	err = locker.SessionLock(ctx, conn)

	if err != nil {
		return migratediag.PhaseError(databaseEnvVar, phaseName, dsn, "session lock", err)
	}

	defer func() {
		if err := locker.SessionUnlock(ctx, conn); err != nil {
			log.Printf("%v", migratediag.PhaseError(databaseEnvVar, phaseName, dsn, "session unlock", err))
		}
	}()

	fsys, err := fs.Sub(embedMigrations, "migrations")
	if err != nil {
		return migratediag.PhaseError(databaseEnvVar, phaseName, dsn, "create migrations sub filesystem", err)
	}
	goose.SetBaseFS(fsys)

	targetVersionInt, err := strconv.ParseInt(targetVersion, 10, 64)
	if err != nil {
		return migratediag.PhaseError(databaseEnvVar, phaseName, dsn, "parse target version", fmt.Errorf("invalid target version %s: %w", targetVersion, err))
	}

	currentVersion, err := goose.GetDBVersion(db)
	if err != nil {
		return migratediag.PhaseError(databaseEnvVar, phaseName, dsn, "get current database version", err)
	}

	if currentVersion < targetVersionInt {
		return migratediag.PhaseError(databaseEnvVar, phaseName, dsn, "validate target version", fmt.Errorf("target version %d is higher than current version %d. Use standard migration (without --down flag) to upgrade", targetVersionInt, currentVersion))
	}

	if currentVersion == targetVersionInt {
		fmt.Printf("Database is already at version %d. No migration needed.\n", targetVersionInt)
		return nil
	}

	fmt.Printf("Migrating down from version %d to version %d\n", currentVersion, targetVersionInt)

	err = goose.DownTo(db, ".", targetVersionInt)
	if err != nil {
		return migratediag.PhaseError(databaseEnvVar, phaseName, dsn, "apply down migration", fmt.Errorf("target version %d: %w", targetVersionInt, err))
	}

	fmt.Printf("Successfully migrated down to version %d\n", targetVersionInt)
	return nil
}
