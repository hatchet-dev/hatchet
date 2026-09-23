package fairpool

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// Options configures the per-tenant connection cap.
//
// MaxPercent is the share of the pool one tenant, and shared engine work, may hold.
// The gate is installed only for values from 1 to 99. 100 (the default) and anything
// outside that range leaves ForTenant and ForShared ungated, so either may hold the whole pool.
type Options struct {
	MaxPercent int
	MaxWait    time.Duration
	PoolName   string
	L          *zerolog.Logger
}

// Pool is a pgx pool plus an optional connection cap. Call ForTenant to count an
// acquire against a tenant, or ForShared to count it against shared engine work.
// Both buckets use the same limit. Unwrap does not count.
type Pool struct {
	inner  *pgxpool.Pool
	gate   *gate
	shared *handle
}

// DB is the handle sqlc and transaction helpers call. A handle from ForTenant
// counts connections against that tenant. A handle from ForShared counts them
// against shared engine work.
type DB interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error)
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
	Begin(ctx context.Context) (pgx.Tx, error)
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
	Acquire(ctx context.Context) (*Conn, error)
	Stat() *pgxpool.Stat
}

// NewWithConfig builds a pool from cfg. The tracer already set on cfg is left in place.
func NewWithConfig(ctx context.Context, cfg *pgxpool.Config, opts Options) (*Pool, error) {
	inner, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return newPool(inner, opts), nil
}

// Wrap returns a pool around an existing pgx pool. With no MaxPercent in opts
// the result is ungated; use NewWithConfig to install a cap.
func Wrap(inner *pgxpool.Pool) *Pool {
	if inner == nil {
		return nil
	}

	return newPool(inner, Options{})
}

func newPool(inner *pgxpool.Pool, opts Options) *Pool {
	p := &Pool{inner: inner}

	if opts.MaxWait <= 0 {
		opts.MaxWait = 5 * time.Second
	}

	if opts.PoolName == "" {
		opts.PoolName = "main"
	}

	if limit := connectionLimit(inner.Config().MaxConns, opts.MaxPercent); limit > 0 {
		p.gate = newGate(limit, opts.MaxWait, opts.PoolName, opts.L)
	}

	p.shared = &handle{pool: p, key: sharedKey, gated: p.gate != nil}

	return p
}

// connectionLimit is the number of connections one tenant, or shared engine work, may hold.
// Zero means the gate is not installed.
func connectionLimit(maxConns int32, percent int) int64 {
	if percent <= 0 || percent >= 100 || maxConns <= 0 {
		return 0
	}

	limit := int64(maxConns) * int64(percent) / 100
	if limit < 1 {
		return 1
	}

	return limit
}

// Unwrap returns the underlying pgx pool. Callers that hijack a connection,
// such as LISTEN, use this so they never take a gate slot.
func (p *Pool) Unwrap() *pgxpool.Pool {
	if p == nil {
		return nil
	}

	return p.inner
}

// ForShared returns the handle for engine work that is not tied to one tenant.
// It counts against the same limit as any single tenant. A pool with no gate
// returns a handle that does not count.
func (p *Pool) ForShared() DB {
	if p == nil {
		return nil
	}

	return p.shared
}

// ForTenant returns a handle that counts acquires against tenantID.
// A nil id counts against the shared bucket. A pool with no gate returns a
// handle that does not count.
func (p *Pool) ForTenant(tenantID uuid.UUID) DB {
	if p == nil {
		return nil
	}

	if tenantID == uuid.Nil {
		return p.shared
	}

	if p.gate == nil {
		return &handle{pool: p}
	}

	return &handle{pool: p, key: tenantID.String(), tenantID: tenantID, gated: true}
}

func (p *Pool) Stat() *pgxpool.Stat {
	return p.inner.Stat()
}

func (p *Pool) Config() *pgxpool.Config {
	return p.inner.Config()
}

func (p *Pool) Close() {
	p.inner.Close()
}

type handle struct {
	pool     *Pool
	key      string
	tenantID uuid.UUID
	gated    bool
}

func (h *handle) enter(ctx context.Context) (func(), error) {
	if !h.gated {
		return func() {}, nil
	}

	return h.pool.gate.enter(ctx, h.key, h.tenantID)
}

func (h *handle) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	release, err := h.enter(ctx)
	if err != nil {
		return pgconn.CommandTag{}, err
	}
	defer release()

	return h.pool.inner.Exec(ctx, sql, arguments...)
}

func (h *handle) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	release, err := h.enter(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := h.pool.inner.Query(ctx, sql, args...)
	if err != nil {
		release()
		return nil, err
	}

	if !h.gated {
		return rows, nil
	}

	return newGatingRows(rows, release), nil
}

func (h *handle) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if !h.gated {
		return h.pool.inner.QueryRow(ctx, sql, args...)
	}

	rows, err := h.Query(ctx, sql, args...)
	if err != nil {
		return errRow{err: err}
	}

	return newGatingRow(rows)
}

func (h *handle) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	release, err := h.enter(ctx)
	if err != nil {
		return 0, err
	}
	defer release()

	return h.pool.inner.CopyFrom(ctx, tableName, columnNames, rowSrc)
}

func (h *handle) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	release, err := h.enter(ctx)
	if err != nil {
		return errBatchResults{err: err}
	}

	results := h.pool.inner.SendBatch(ctx, b)
	if !h.gated {
		return results
	}

	return newGatingBatch(results, release)
}

func (h *handle) Begin(ctx context.Context) (pgx.Tx, error) {
	return h.BeginTx(ctx, pgx.TxOptions{})
}

func (h *handle) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	release, err := h.enter(ctx)
	if err != nil {
		return nil, err
	}

	tx, err := h.pool.inner.BeginTx(ctx, txOptions)
	if err != nil {
		release()
		return nil, err
	}

	if !h.gated {
		return tx, nil
	}

	return newGatingTx(tx, release), nil
}

func (h *handle) Acquire(ctx context.Context) (*Conn, error) {
	release, err := h.enter(ctx)
	if err != nil {
		return nil, err
	}

	conn, err := h.pool.inner.Acquire(ctx)
	if err != nil {
		release()
		return nil, err
	}

	if !h.gated {
		return newConn(conn, nil), nil
	}

	return newConn(conn, release), nil
}

func (h *handle) Stat() *pgxpool.Stat {
	return h.pool.inner.Stat()
}
