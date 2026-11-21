package loader

import (
	"cmp"
	"context"
	"fmt"
	"net"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	// defaultPgxStatementTimeout bounds every statement and idle-in-transaction session on
	// connections built here. It is what the engine has always used.
	defaultPgxStatementTimeout = 30 * time.Second

	// defaultPgxDialTimeout bounds the TCP handshake. In-region it takes milliseconds, and Linux
	// resends a lost SYN after 1s, so 3s survives one dropped packet.
	defaultPgxDialTimeout = 3 * time.Second

	// defaultPgxConnectTimeout bounds dial, TLS and auth. A live but overloaded server can stall
	// backend startup for several seconds, so it stays well above the dial timeout.
	defaultPgxConnectTimeout = 15 * time.Second
)

// PgxPoolOpts configures a pool built with NewPgxPool or NewPgxPoolConfig. Zero values leave
// the pgxpool defaults in place, except StatementTimeout, DialTimeout and ConnectTimeout, which
// default to the constants above.
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

	// DialTimeout bounds the TCP connect only. During a failover the old primary's address can
	// drop packets, and a dial to it holds a pool slot until this fires.
	DialTimeout time.Duration

	// ConnectTimeout bounds the connect (dial, TLS, auth) and, separately, the AfterConnect setup
	// queries. It overrides connect_timeout in the URL.
	ConnectTimeout time.Duration

	// Tracer replaces the default OpenTelemetry query tracer.
	Tracer pgx.QueryTracer
}

// NewPgxPoolConfig parses url and applies the connection setup every Hatchet pool shares: the
// application name, UTC session time zone, the enum and range types pgx cannot encode without
// registration, the statement timeouts and the connect bounds. Callers that need further hooks
// (the engine's query debugger) adjust the returned config before opening the pool.
func NewPgxPoolConfig(url string, opts PgxPoolOpts) (*pgxpool.Config, error) {
	config, err := pgxpool.ParseConfig(url)

	if err != nil {
		return nil, fmt.Errorf("could not parse database url: %w", err)
	}

	if opts.ApplicationName != "" {
		setPgxApplicationName(config, opts.ApplicationName)
	}

	boundPgxConnect(config.ConnConfig, opts.DialTimeout, opts.ConnectTimeout)

	config.AfterConnect = pgxAfterConnect(
		cmp.Or(opts.StatementTimeout, defaultPgxStatementTimeout),
		config.ConnConfig.ConnectTimeout,
	)

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

// boundPgxConnect sets separate dial and connect timeouts, defaulting zero values. pgxpool builds
// a new connection on a context the caller's cancel doesn't reach, so these timeouts are what
// free a pool slot stuck on an unreachable or stalled server.
func boundPgxConnect(cfg *pgx.ConnConfig, dialTimeout, connectTimeout time.Duration) {
	cfg.ConnectTimeout = cmp.Or(connectTimeout, defaultPgxConnectTimeout)
	cfg.DialFunc = (&net.Dialer{Timeout: cmp.Or(dialTimeout, defaultPgxDialTimeout)}).DialContext
}

// pgxAfterConnect prepares a fresh connection: UTC, the custom types the sqlc queries bind as
// parameters, and the timeouts. The setup is bounded by connectTimeout, since pgxpool runs it on
// a context without a deadline.
func pgxAfterConnect(statementTimeout, connectTimeout time.Duration) func(ctx context.Context, conn *pgx.Conn) error {
	timeoutMs := statementTimeout.Milliseconds()

	return func(ctx context.Context, conn *pgx.Conn) error {
		ctx, cancel := context.WithTimeout(ctx, connectTimeout)
		defer cancel()

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
