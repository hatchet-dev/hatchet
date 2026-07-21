package keyset

import (
	"context"
	"fmt"
	"regexp"

	"github.com/jackc/pgx/v5"
)

// schemaNameRE matches a safe Postgres identifier: a leading letter or underscore
// followed by letters, digits, or underscores, up to the 63-byte identifier limit.
// The schema name is interpolated into DDL, so we reject anything outside this set
// to avoid identifier injection.
var schemaNameRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,62}$`)

func validateSchemaName(schema string) error {
	if !schemaNameRE.MatchString(schema) {
		return fmt.Errorf("invalid schema name %q: must match %s", schema, schemaNameRE)
	}
	return nil
}

// Migrate ensures the keyset schema and table exist. It is the explicit alternative to the
// auto-migration Resolve performs, for callers that want to run DDL in a separate phase.
func Migrate(ctx context.Context, databaseURL string, opts ...Opt) error {
	o := defaultOpts()
	for _, f := range opts {
		f(o)
	}
	if err := validateSchemaName(o.schema); err != nil {
		return err
	}

	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("could not connect to database: %w", err)
	}
	defer func() { _ = conn.Close(ctx) }()

	return migrate(ctx, conn, o.schema)
}

func migrate(ctx context.Context, conn *pgx.Conn, schema string) error {
	if _, err := conn.Exec(ctx, "CREATE SCHEMA IF NOT EXISTS "+pgx.Identifier{schema}.Sanitize()); err != nil {
		return fmt.Errorf("could not create schema %q: %w", schema, err)
	}

	table := pgx.Identifier{schema, "keyset"}.Sanitize()
	if _, err := conn.Exec(ctx, "CREATE TABLE IF NOT EXISTS "+table+` (
		id bool PRIMARY KEY DEFAULT true CHECK (id),
		master text NOT NULL,
		private_jwt text NOT NULL,
		public_jwt text NOT NULL
	)`); err != nil {
		return fmt.Errorf("could not create keyset table: %w", err)
	}

	return nil
}
