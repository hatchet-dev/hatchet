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

	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
)

// Options configures the connection caps.
//
// MaxPercent is the share of the pool one tenant may have in flight. SharedMaxPercent is the
// same share for the bucket of engine work that is not tied to a tenant (ForShared and a nil
// tenant id). For both, a value from 1 to 99 installs a cap, and 100 (or 0, the zero value)
// installs none, so that bucket may hold the whole pool. NewWithConfig rejects values outside
// 0 to 100. SetTenantMaxPercents overrides MaxPercent for single tenants at runtime.
//
// BudgetPercent is each tenant's database time budget: the share of the pool's connection-seconds
// one tenant may hold on average. A value from 1 to 99 installs a budget, and 100 (or 0) installs
// none. SetTenantBudgetPercents overrides it for single tenants. The shared bucket has no budget.
type Options struct {
	MaxPercent       int
	SharedMaxPercent int
	BudgetPercent    int
	MaxWait          time.Duration
	PoolName         string
	L                *zerolog.Logger
}

// Pool is a pgx pool plus connection caps and accounting. ForTenant counts a checkout
// against a tenant and ForShared counts it against shared engine work. Each
// tenant has its own limit, and the shared bucket has a separate one. Ungated
// and Unwrap do not count.
type Pool struct {
	inner    *pgxpool.Pool
	maxConns int32
	// gate and sharedGate are nil only on an Ungated pool.
	gate       *gate
	sharedGate *gate
	shared     *handle
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
	if err := checkPercent("max percent", opts.MaxPercent); err != nil {
		return nil, err
	}

	if err := checkPercent("shared max percent", opts.SharedMaxPercent); err != nil {
		return nil, err
	}

	if err := checkPercent("budget percent", opts.BudgetPercent); err != nil {
		return nil, err
	}

	inner, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return newPool(inner, opts), nil
}

// Ungated returns a pool that neither caps nor counts connections. Tests and call sites
// that must not take a slot use this instead of ForTenant or ForShared.
func Ungated(inner *pgxpool.Pool) *Pool {
	if inner == nil {
		return nil
	}

	p := &Pool{inner: inner, maxConns: inner.Config().MaxConns}
	p.shared = &handle{pool: p, key: sharedKey}

	return p
}

func newPool(inner *pgxpool.Pool, opts Options) *Pool {
	p := &Pool{inner: inner, maxConns: inner.Config().MaxConns}

	if opts.MaxWait <= 0 {
		opts.MaxWait = 5 * time.Second
	}

	if opts.PoolName == "" {
		opts.PoolName = "main"
	}

	poolSize := int64(p.maxConns)

	limit := connectionLimit(p.maxConns, opts.MaxPercent)
	p.gate = newGate(limit, poolSize, p.budgetRate(opts.BudgetPercent), opts.MaxWait, opts.PoolName, opts.L)

	if limit > 0 {
		prometheus.FairpoolConnectionLimit.WithLabelValues(opts.PoolName, "tenant").Set(float64(limit))
	}

	sharedLimit := connectionLimit(p.maxConns, opts.SharedMaxPercent)
	p.sharedGate = newGate(sharedLimit, poolSize, 0, opts.MaxWait, opts.PoolName, opts.L)

	if sharedLimit > 0 {
		prometheus.FairpoolConnectionLimit.WithLabelValues(opts.PoolName, sharedKey).Set(float64(sharedLimit))
	}

	p.shared = &handle{pool: p, key: sharedKey, gate: p.sharedGate}

	return p
}

// budgetRate converts a budget percent into held-seconds per second. Zero means no budget.
func (p *Pool) budgetRate(percent int) float64 {
	if percent <= 0 || percent >= 100 {
		return 0
	}

	return float64(p.maxConns) * float64(percent) / 100
}

// checkPercent accepts 0 (the zero value) and 1 through 100; both 0 and 100 install nothing.
func checkPercent(name string, percent int) error {
	if percent == 0 || (percent >= 1 && percent <= 100) {
		return nil
	}

	return fmt.Errorf("fairpool %s must be from 1 to 100, got %d", name, percent)
}

// connectionLimit is the number of checkouts one bucket may have in flight.
// Zero means no gate is installed.
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

// ConnectionLimit is how many checkouts one tenant without an override may have in flight.
// Zero means such tenants are not capped.
func (p *Pool) ConnectionLimit() int64 {
	if p == nil || p.gate == nil {
		return 0
	}

	return p.gate.limit
}

// SetTenantMaxPercents replaces the per-tenant overrides of MaxPercent. A tenant in percents may
// have that share of the pool in flight, at least one connection; any other tenant gets the
// default. Checkouts already in flight finish against the limit they started under.
func (p *Pool) SetTenantMaxPercents(percents map[uuid.UUID]int) {
	limits := make(map[uuid.UUID]int64, len(percents))

	for tenantID, percent := range percents {
		limits[tenantID] = max(1, int64(p.maxConns)*int64(percent)/100)
	}

	p.gate.overrides.Store(&limits)
}

// SetTenantBudgetPercents replaces the per-tenant overrides of BudgetPercent. A tenant in percents
// gets that budget, and 100 means none; any other tenant gets the default.
func (p *Pool) SetTenantBudgetPercents(percents map[uuid.UUID]int) {
	rates := make(map[uuid.UUID]float64, len(percents))

	for tenantID, percent := range percents {
		rates[tenantID] = p.budgetRate(percent)
	}

	p.gate.budgets.Store(&rates)
}

// AwaitBudget is the yield point of a tenant's deferrable work: a per-tenant loop calls it at the
// top of an iteration, before it takes any lock or connection. It returns at once when the tenant
// is within its database time budget. Otherwise it waits until the budget is back to zero, for at
// most budgetWindow (10 seconds) in total, or until ctx ends, and then returns ctx's error.
// Checkouts never wait for the budget themselves; every release debits it.
func (p *Pool) AwaitBudget(ctx context.Context, tenantID uuid.UUID) error {
	return p.gate.awaitBudget(ctx, tenantID.String(), tenantID)
}

// SharedConnectionLimit is how many checkouts the shared bucket may have in flight.
// Zero means the shared bucket is not capped.
func (p *Pool) SharedConnectionLimit() int64 {
	if p == nil || p.sharedGate == nil {
		return 0
	}

	return p.sharedGate.limit
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
// It counts against the shared limit, not a tenant's.
func (p *Pool) ForShared() Handle {
	if p == nil {
		return nil
	}

	return p.shared
}

// ForTenant returns a handle that counts acquires against tenantID.
// A nil id counts against the shared bucket.
func (p *Pool) ForTenant(tenantID uuid.UUID) Handle {
	if p == nil {
		return nil
	}

	if tenantID == uuid.Nil {
		return p.shared
	}

	return &handle{pool: p, key: tenantID.String(), tenantID: tenantID, gate: p.gate}
}

// Stat returns the underlying pool's statistics.
func (p *Pool) Stat() *pgxpool.Stat {
	return p.inner.Stat()
}

// Config returns a copy of the underlying pool's configuration.
func (p *Pool) Config() *pgxpool.Config {
	return p.inner.Config()
}

// Close closes the underlying pool. It blocks until all checked-out connections are released.
func (p *Pool) Close() {
	p.inner.Close()
}

type handle struct {
	pool     *Pool
	key      string
	tenantID uuid.UUID
	// gate is nil on an Ungated pool.
	gate *gate
}

func releaseCheckedOut(hold *slotHold, releaseConn func()) {
	if releaseConn != nil {
		releaseConn()
	}

	hold.release()
}

func (h *handle) pinConn(ctx context.Context) (*pgxpool.Conn, *slotHold, error) {
	var conn *pgxpool.Conn

	hold, err := h.gate.pin(ctx, h.key, h.tenantID, func() error {
		var acquireErr error
		conn, acquireErr = h.pool.inner.Acquire(ctx)
		return acquireErr
	})
	if err != nil {
		return nil, nil, err
	}

	return conn, hold, nil
}

func (h *handle) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	conn, hold, err := h.pinConn(ctx)
	if err != nil {
		return pgconn.CommandTag{}, err
	}
	defer releaseCheckedOut(hold, conn.Release)

	return conn.Exec(ctx, sql, arguments...)
}

func (h *handle) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	conn, hold, err := h.pinConn(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := conn.Query(ctx, sql, args...)
	if err != nil {
		releaseCheckedOut(hold, conn.Release)
		return nil, err
	}

	return newGatingRows(rows, conn, hold), nil
}

func (h *handle) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	rows, err := h.Query(ctx, sql, args...)
	if err != nil {
		return errRow{err: err}
	}

	return &gatingRow{rows: rows}
}

func (h *handle) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	conn, hold, err := h.pinConn(ctx)
	if err != nil {
		return 0, err
	}
	defer releaseCheckedOut(hold, conn.Release)

	return conn.CopyFrom(ctx, tableName, columnNames, rowSrc)
}

func (h *handle) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	conn, hold, err := h.pinConn(ctx)
	if err != nil {
		return errBatchResults{err: err}
	}

	return newGatingBatch(conn.SendBatch(ctx, b), conn, hold)
}

func (h *handle) Begin(ctx context.Context) (pgx.Tx, error) {
	return h.BeginTx(ctx, pgx.TxOptions{})
}

func (h *handle) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	var tx pgx.Tx
	hold, err := h.gate.pin(ctx, h.key, h.tenantID, func() error {
		var beginErr error
		tx, beginErr = h.pool.inner.BeginTx(ctx, txOptions)
		return beginErr
	})
	if err != nil {
		return nil, err
	}

	return newGatingTx(tx, hold), nil
}

func (h *handle) Acquire(ctx context.Context) (*Conn, error) {
	conn, hold, err := h.pinConn(ctx)
	if err != nil {
		return nil, err
	}

	return newConn(conn, hold), nil
}

func (h *handle) Stat() *pgxpool.Stat {
	return h.pool.inner.Stat()
}
