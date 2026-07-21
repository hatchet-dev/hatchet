// Package keyset manages the encryption keysets shared across embedded Hatchet
// instances. The keyset is persisted in a dedicated Postgres schema so that every
// process in an embedded fleet — single or multi-engine — resolves the same keys
// without the user having to generate or distribute them.
package keyset

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/hatchet-dev/hatchet/pkg/encryption"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
)

// DefaultSchema is the Postgres schema that stores the embedded keyset.
const DefaultSchema = "hatchet_embed"

// Keyset holds the encryption keysets shared across embedded instances.
type Keyset struct {
	Master     []byte
	PrivateJWT []byte
	PublicJWT  []byte
}

type options struct {
	schema string
}

// Opt configures keyset resolution.
type Opt func(*options)

func defaultOpts() *options {
	return &options{schema: DefaultSchema}
}

// WithSchema overrides the Postgres schema that stores the keyset (default "hatchet_embed").
func WithSchema(schema string) Opt {
	return func(o *options) { o.schema = schema }
}

// Resolve returns the shared keyset, generating and persisting it on first use. It is safe to
// call concurrently across processes: a Postgres advisory lock serializes generation so every
// caller converges on a single keyset. The schema is created if absent, which requires only
// CREATE on the database (not CREATEDB).
func Resolve(ctx context.Context, databaseURL string, opts ...Opt) (*Keyset, error) {
	o := defaultOpts()
	for _, f := range opts {
		f(o)
	}
	if err := validateSchemaName(o.schema); err != nil {
		return nil, err
	}

	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("could not connect to database: %w", err)
	}
	defer func() { _ = conn.Close(ctx) }()

	lockKey := sqlchelpers.AdvisoryLockKey("hatchet-embed-keyset:" + o.schema)
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", lockKey); err != nil {
		return nil, fmt.Errorf("could not acquire keyset lock: %w", err)
	}
	defer func() { _, _ = conn.Exec(ctx, "SELECT pg_advisory_unlock($1)", lockKey) }()

	if err := migrate(ctx, conn, o.schema); err != nil {
		return nil, err
	}

	table := pgx.Identifier{o.schema, "keyset"}.Sanitize()

	var master, privateJWT, publicJWT string
	err = conn.QueryRow(ctx, "SELECT master, private_jwt, public_jwt FROM "+table+" LIMIT 1").
		Scan(&master, &privateJWT, &publicJWT)
	if err == nil {
		return &Keyset{Master: []byte(master), PrivateJWT: []byte(privateJWT), PublicJWT: []byte(publicJWT)}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("could not read keyset: %w", err)
	}

	genMaster, genPrivate, genPublic, _, err := encryption.GenerateLocalKeys()
	if err != nil {
		return nil, fmt.Errorf("could not generate keysets: %w", err)
	}

	if _, err := conn.Exec(ctx,
		"INSERT INTO "+table+" (master, private_jwt, public_jwt) VALUES ($1, $2, $3)",
		string(genMaster), string(genPrivate), string(genPublic)); err != nil {
		return nil, fmt.Errorf("could not store keyset: %w", err)
	}

	return &Keyset{Master: genMaster, PrivateJWT: genPrivate, PublicJWT: genPublic}, nil
}
