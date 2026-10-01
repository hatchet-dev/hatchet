package fairpool

import (
	"context"
	"fmt"
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

// Pool is a pgx pool plus an optional connection cap. ForTenant counts a
// connection against a tenant after one is checked out. ForShared counts it
// against shared engine work. Both buckets use the same limit. Ungated and
// Unwrap do not count.
type Pool struct {
	inner  *pgxpool.Pool
	gate   *gate
	shared *handle
}

// Handle is one tenant's checkout, or the shared checkout. sqlc accepts it
// anywhere it accepts a DBTX. sqlchelpers.Pool is the smaller begin-a-transaction
// view of a Handle.
type Handle interface {
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
	if err := checkPercent(opts.MaxPercent); err != nil {
		return nil, err
	}

	inner, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return newPool(inner, opts), nil
}

// Ungated returns a pool that does not cap connections. Tests and call sites
// that must not take a slot use this instead of ForTenant or ForShared.
func Ungated(inner *pgxpool.Pool) *Pool {
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

// checkPercent accepts 0 (no cap, used by Ungated) and 1 through 100 (100 installs no cap).
func checkPercent(percent int) error {
	if percent == 0 || (percent >= 1 && percent <= 100) {
		return nil
	}

	return fmt.Errorf("fairpool max percent must be from 1 to 100, got %d", percent)
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

// ConnectionLimit is how many connections one tenant, or shared work, may hold.
// Zero means the pool is ungated.
func (p *Pool) ConnectionLimit() int64 {
	if p == nil || p.gate == nil {
		return 0
	}

	return p.gate.limit
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
func (p *Pool) ForShared() Handle {
	if p == nil {
		return nil
	}

	return p.shared
}

// ForTenant returns a handle that counts acquires against tenantID.
// A nil id counts against the shared bucket. A pool with no gate returns a
// handle that does not count.
func (p *Pool) ForTenant(tenantID uuid.UUID) Handle {
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

func releaseCheckedOut(hold *slotHold, releaseConn func()) {
	if releaseConn != nil {
		releaseConn()
	}

	hold.release()
}

func (h *handle) pinConn(ctx context.Context) (*pgxpool.Conn, *slotHold, error) {
	var conn *pgxpool.Conn

	hold, err := h.pool.gate.pin(ctx, h.key, h.tenantID, func() error {
		var acquireErr error
		conn, acquireErr = h.pool.inner.Acquire(ctx)
		return acquireErr
	}, func() { conn.Release() })
	if err != nil {
		return nil, nil, err
	}

	return conn, hold, nil
}

func (h *handle) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	if !h.gated {
		return h.pool.inner.Exec(ctx, sql, arguments...)
	}

	conn, hold, err := h.pinConn(ctx)
	if err != nil {
		return pgconn.CommandTag{}, err
	}
	defer releaseCheckedOut(hold, conn.Release)

	return conn.Exec(ctx, sql, arguments...)
}

func (h *handle) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if !h.gated {
		return h.pool.inner.Query(ctx, sql, args...)
	}

	conn, hold, err := h.pinConn(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := conn.Query(ctx, sql, args...)
	if err != nil {
		releaseCheckedOut(hold, conn.Release)
		return nil, err
	}

	return newGatingRows(rows, func() { releaseCheckedOut(hold, conn.Release) }), nil
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
	if !h.gated {
		return h.pool.inner.CopyFrom(ctx, tableName, columnNames, rowSrc)
	}

	conn, hold, err := h.pinConn(ctx)
	if err != nil {
		return 0, err
	}
	defer releaseCheckedOut(hold, conn.Release)

	return conn.CopyFrom(ctx, tableName, columnNames, rowSrc)
}

func (h *handle) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	if !h.gated {
		return h.pool.inner.SendBatch(ctx, b)
	}

	conn, hold, err := h.pinConn(ctx)
	if err != nil {
		return errBatchResults{err: err}
	}

	return newGatingBatch(conn.SendBatch(ctx, b), func() { releaseCheckedOut(hold, conn.Release) })
}

func (h *handle) Begin(ctx context.Context) (pgx.Tx, error) {
	return h.BeginTx(ctx, pgx.TxOptions{})
}

func (h *handle) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	if !h.gated {
		return h.pool.inner.BeginTx(ctx, txOptions)
	}

	var tx pgx.Tx
	hold, err := h.pool.gate.pin(ctx, h.key, h.tenantID, func() error {
		var beginErr error
		tx, beginErr = h.pool.inner.BeginTx(ctx, txOptions)
		return beginErr
	}, func() { _ = tx.Rollback(ctx) })
	if err != nil {
		return nil, err
	}

	return newGatingTx(tx, hold), nil
}

func (h *handle) Acquire(ctx context.Context) (*Conn, error) {
	if !h.gated {
		conn, err := h.pool.inner.Acquire(ctx)
		if err != nil {
			return nil, err
		}

		return newConn(conn, nil), nil
	}

	conn, hold, err := h.pinConn(ctx)
	if err != nil {
		return nil, err
	}

	return newConn(conn, hold), nil
}

func (h *handle) Stat() *pgxpool.Stat {
	return h.pool.inner.Stat()
}
