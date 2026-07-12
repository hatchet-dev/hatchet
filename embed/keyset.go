package embed

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/jackc/pgx/v5"

	"github.com/hatchet-dev/hatchet/pkg/encryption"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
)

const keysetDatabase = "hatchet_embed"

func resolveKeysets(ctx context.Context, databaseURL string) (master, privateJWT, publicJWT []byte, err error) {
	lockKey := sqlchelpers.AdvisoryLockKey("hatchet-embed-keyset")

	mainConn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("could not connect to database: %w", err)
	}
	defer func() { _ = mainConn.Close(ctx) }()

	if _, advLkErr := mainConn.Exec(ctx, "SELECT pg_advisory_lock($1)", lockKey); advLkErr != nil {
		return nil, nil, nil, fmt.Errorf("could not acquire keyset lock: %w", advLkErr)
	}
	defer func() { _, _ = mainConn.Exec(ctx, "SELECT pg_advisory_unlock($1)", lockKey) }()

	if dbCheckErr := ensureKeysetDatabase(ctx, mainConn); dbCheckErr != nil {
		return nil, nil, nil, dbCheckErr
	}

	embedURL, err := swapDatabase(databaseURL, keysetDatabase)
	if err != nil {
		return nil, nil, nil, err
	}

	embedConn, err := pgx.Connect(ctx, embedURL)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("could not connect to %s: %w", keysetDatabase, err)
	}
	defer func() { _ = embedConn.Close(ctx) }()

	if _, createTableErr := embedConn.Exec(ctx, `CREATE TABLE IF NOT EXISTS keyset (
		id bool PRIMARY KEY DEFAULT true CHECK (id),
		master text NOT NULL,
		private_jwt text NOT NULL,
		public_jwt text NOT NULL
	)`); createTableErr != nil {
		return nil, nil, nil, fmt.Errorf("could not create keyset table: %w", createTableErr)
	}

	var m, priv, pub string
	err = embedConn.QueryRow(ctx, "SELECT master, private_jwt, public_jwt FROM keyset LIMIT 1").Scan(&m, &priv, &pub)
	if err == nil {
		return []byte(m), []byte(priv), []byte(pub), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil, fmt.Errorf("could not read keyset: %w", err)
	}

	genMaster, genPriv, genPub, _, err := encryption.GenerateLocalKeys()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("could not generate keysets: %w", err)
	}

	if _, err := embedConn.Exec(ctx,
		"INSERT INTO keyset (master, private_jwt, public_jwt) VALUES ($1, $2, $3)",
		string(genMaster), string(genPriv), string(genPub)); err != nil {
		return nil, nil, nil, fmt.Errorf("could not store keyset: %w", err)
	}

	return genMaster, genPriv, genPub, nil
}

func ensureKeysetDatabase(ctx context.Context, conn *pgx.Conn) error {
	var exists bool
	if err := conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", keysetDatabase).Scan(&exists); err != nil {
		return fmt.Errorf("could not check for %s database: %w", keysetDatabase, err)
	}
	if exists {
		return nil
	}
	if _, err := conn.Exec(ctx, "CREATE DATABASE "+keysetDatabase); err != nil {
		return fmt.Errorf("could not create %s database (does the role have CREATEDB?): %w", keysetDatabase, err)
	}
	return nil
}

func swapDatabase(databaseURL, dbName string) (string, error) {
	u, err := url.Parse(databaseURL)
	if err != nil {
		return "", fmt.Errorf("could not parse database url: %w", err)
	}
	u.Path = "/" + dbName
	return u.String(), nil
}
