package loader

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// defaultPgxStatementTimeout bounds every statement and idle-in-transaction session on
// connections built here. It is what the engine has always used.
const defaultPgxStatementTimeout = 30 * time.Second

// PgxPoolOpts configures a pool built with NewPgxPool or NewPgxPoolConfig. Zero values leave
// the pgxpool defaults in place, except StatementTimeout, which defaults to
// defaultPgxStatementTimeout.
type PgxPoolOpts struct {
	// ApplicationName shows up in pg_stat_activity so connections can be attributed to a
	// service on shared postgres instances. Not applied when the URL already carries one.
	ApplicationName string

	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration

	// StatementTimeout is set as both statement_timeout and
	// idle_in_transaction_session_timeout on every connection.
	StatementTimeout time.Duration

	// Tracer replaces the default OpenTelemetry query tracer.
	Tracer pgx.QueryTracer
}

// NewPgxPoolConfig parses url and applies the connection setup every Hatchet pool shares: the
// application name, UTC session time zone, the enum and range types pgx cannot encode without
// registration, and the statement timeouts. Callers that need further hooks (the engine's
// query debugger) adjust the returned config before opening the pool.
func NewPgxPoolConfig(url string, opts PgxPoolOpts) (*pgxpool.Config, error) {
	config, err := pgxpool.ParseConfig(url)

	if err != nil {
		return nil, fmt.Errorf("could not parse database url: %w", err)
	}

	if opts.ApplicationName != "" {
		setPgxApplicationName(config, opts.ApplicationName)
	}

	statementTimeout := opts.StatementTimeout

	if statementTimeout == 0 {
		statementTimeout = defaultPgxStatementTimeout
	}

	config.AfterConnect = pgxAfterConnect(statementTimeout)

	if opts.Tracer != nil {
		config.ConnConfig.Tracer = opts.Tracer
	} else {
		config.ConnConfig.Tracer = newOTelPgxTracer()
	}

	if opts.MaxConns != 0 {
		config.MaxConns = opts.MaxConns
	}

	if opts.MinConns != 0 {
		config.MinConns = opts.MinConns
	}

	config.MaxConnLifetime = opts.MaxConnLifetime
	config.MaxConnIdleTime = opts.MaxConnIdleTime

	return config, nil
}

// NewPgxPool opens a pool on url with the shared connection setup of NewPgxPoolConfig. It is
// the constructor for processes that talk to the engine database without the full config
// loader, such as the out-of-process serverless operator.
func NewPgxPool(ctx context.Context, url string, opts PgxPoolOpts) (*pgxpool.Pool, error) {
	config, err := NewPgxPoolConfig(url, opts)

	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)

	if err != nil {
		return nil, fmt.Errorf("could not connect to database: %w", err)
	}

	return pool, nil
}

// pgxAfterConnect prepares a fresh connection: UTC, the custom types the sqlc queries bind as
// parameters, and the timeouts.
func pgxAfterConnect(statementTimeout time.Duration) func(ctx context.Context, conn *pgx.Conn) error {
	timeoutMs := statementTimeout.Milliseconds()

	return func(ctx context.Context, conn *pgx.Conn) error {
		// Set timezone to UTC for all connections
		if _, err := conn.Exec(ctx, "SET TIME ZONE 'UTC'"); err != nil {
			return err
		}

		// ref: https://github.com/jackc/pgx/issues/1549
		for _, typeName := range []string{
			"v1_readable_status_olap",
			"_v1_readable_status_olap",
			"v1_log_line_level",
			"_v1_log_line_level",
		} {
			t, err := conn.LoadType(ctx, typeName)

			if err != nil {
				return err
			}

			conn.TypeMap().RegisterType(t)
		}

		if uuidType, ok := conn.TypeMap().TypeForName("uuid"); ok {
			var uuidrangeOID uint32
			err := conn.QueryRow(ctx, "SELECT oid FROM pg_type WHERE typname = 'uuidrange'").Scan(&uuidrangeOID)
			if err != nil && err != pgx.ErrNoRows {
				return fmt.Errorf("loading uuidrange oid: %w", err)
			}
			if err == nil {
				conn.TypeMap().RegisterType(&pgtype.Type{
					Name:  "uuidrange",
					OID:   uuidrangeOID,
					Codec: &pgtype.RangeCodec{ElementType: uuidType},
				})
			}
		}

		if _, err := conn.Exec(ctx, fmt.Sprintf("SET statement_timeout=%d", timeoutMs)); err != nil {
			return err
		}

		_, err := conn.Exec(ctx, fmt.Sprintf("SET idle_in_transaction_session_timeout=%d", timeoutMs))

		return err
	}
}
