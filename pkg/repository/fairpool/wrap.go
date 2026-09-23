package fairpool

import (
	"context"
	"runtime"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Conn is a pooled connection counted against a tenant. Hijack releases the
// tenant slot immediately, because a hijacked connection no longer occupies
// the pool. Release returns the connection and the slot.
type Conn struct {
	hold *connHold
	stop runtime.Cleanup
}

type connHold struct {
	inner *pgxpool.Conn
	slot  *slotHold
	once  sync.Once
}

func newConn(inner *pgxpool.Conn, slot *slotHold) *Conn {
	hold := &connHold{inner: inner, slot: slot}
	c := &Conn{hold: hold}
	c.stop = runtime.AddCleanup(c, func(h *connHold) { h.releaseConn() }, hold)

	return c
}

func (h *connHold) run(fn func()) bool {
	first := false

	h.once.Do(func() {
		first = true
		fn()
	})

	return first
}

func (h *connHold) releaseConn() bool {
	return h.run(func() {
		h.inner.Release()
		h.slot.release()
	})
}

func (h *connHold) hijack() (*pgx.Conn, bool) {
	var conn *pgx.Conn

	ok := h.run(func() {
		conn = h.inner.Hijack()
		h.slot.release()
	})

	return conn, ok
}

// Release returns the connection to the pool and frees the tenant slot.
func (c *Conn) Release() {
	c.stop.Stop()
	c.hold.releaseConn()
}

// Hijack removes the connection from the pool and frees the tenant slot.
// The caller owns the returned connection.
func (c *Conn) Hijack() *pgx.Conn {
	c.stop.Stop()

	conn, ok := c.hold.hijack()
	if !ok {
		panic("cannot hijack already released or hijacked connection")
	}

	return conn
}

func (c *Conn) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	c.hold.slot.retag(queryName(sql))
	return c.hold.inner.Exec(ctx, sql, arguments...)
}

func (c *Conn) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	c.hold.slot.retag(queryName(sql))
	return c.hold.inner.Query(ctx, sql, args...)
}

func (c *Conn) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	c.hold.slot.retag(queryName(sql))
	return c.hold.inner.QueryRow(ctx, sql, args...)
}

func (c *Conn) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	c.hold.slot.retag(batchLabel(b))
	return c.hold.inner.SendBatch(ctx, b)
}

func (c *Conn) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	c.hold.slot.retag("copy")
	return c.hold.inner.CopyFrom(ctx, tableName, columnNames, rowSrc)
}

func (c *Conn) Begin(ctx context.Context) (pgx.Tx, error) {
	return c.hold.inner.Begin(ctx)
}

func (c *Conn) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	return c.hold.inner.BeginTx(ctx, txOptions)
}

func (c *Conn) Ping(ctx context.Context) error {
	return c.hold.inner.Ping(ctx)
}

func (c *Conn) Conn() *pgx.Conn {
	return c.hold.inner.Conn()
}

type rowsHold struct {
	rows    pgx.Rows
	release func()
	once    sync.Once
}

func (h *rowsHold) finish() {
	h.once.Do(func() {
		defer h.release()
		h.rows.Close()
	})
}

type gatingRows struct {
	pgx.Rows
	hold *rowsHold
	stop runtime.Cleanup
}

func newGatingRows(rows pgx.Rows, release func()) pgx.Rows {
	hold := &rowsHold{rows: rows, release: release}
	g := &gatingRows{Rows: rows, hold: hold}
	g.stop = runtime.AddCleanup(g, func(h *rowsHold) { h.finish() }, hold)

	return g
}

func (g *gatingRows) finish() {
	g.stop.Stop()
	g.hold.finish()
}

func (g *gatingRows) Close() {
	g.finish()
}

func (g *gatingRows) Next() bool {
	if g.Rows.Next() {
		return true
	}

	// pgx releases the connection when Next returns false. Free the slot too,
	// including when the caller never calls Close.
	g.finish()

	return false
}

func (g *gatingRows) Scan(dest ...any) error {
	err := g.Rows.Scan(dest...)
	if err != nil {
		g.finish()
	}

	return err
}

func (g *gatingRows) Values() ([]any, error) {
	values, err := g.Rows.Values()
	if err != nil {
		g.finish()
	}

	return values, err
}

type gatingRow struct {
	rows pgx.Rows
	stop runtime.Cleanup
}

func newGatingRow(rows pgx.Rows) *gatingRow {
	r := &gatingRow{rows: rows}
	r.stop = runtime.AddCleanup(r, func(rows pgx.Rows) { rows.Close() }, rows)

	return r
}

func (r *gatingRow) Scan(dest ...any) error {
	r.stop.Stop()
	defer r.rows.Close()

	if !r.rows.Next() {
		if err := r.rows.Err(); err != nil {
			return err
		}

		return pgx.ErrNoRows
	}

	err := r.rows.Scan(dest...)

	for r.rows.Next() {
	}

	if err == nil {
		err = r.rows.Err()
	}

	return err
}

type txHold struct {
	tx   pgx.Tx
	slot *slotHold
	once sync.Once
}

func (h *txHold) finish(ctx context.Context, commit bool) error {
	var err error

	h.once.Do(func() {
		defer h.slot.release()

		if commit {
			err = h.tx.Commit(ctx)
			return
		}

		err = h.tx.Rollback(ctx)
	})

	return err
}

type gatingTx struct {
	pgx.Tx
	hold *txHold
	stop runtime.Cleanup
}

func newGatingTx(tx pgx.Tx, slot *slotHold) pgx.Tx {
	hold := &txHold{tx: tx, slot: slot}
	g := &gatingTx{Tx: tx, hold: hold}
	// An abandoned transaction still occupies a pool connection. Roll it back
	// once nothing references it so the slot does not stick for the process lifetime.
	g.stop = runtime.AddCleanup(g, func(h *txHold) { _ = h.finish(context.Background(), false) }, hold)

	return g
}

func (g *gatingTx) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	g.hold.slot.retag(queryName(sql))
	return g.Tx.Exec(ctx, sql, arguments...)
}

func (g *gatingTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	g.hold.slot.retag(queryName(sql))
	return g.Tx.Query(ctx, sql, args...)
}

func (g *gatingTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	g.hold.slot.retag(queryName(sql))
	return g.Tx.QueryRow(ctx, sql, args...)
}

func (g *gatingTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	g.hold.slot.retag(batchLabel(b))
	return g.Tx.SendBatch(ctx, b)
}

func (g *gatingTx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	g.hold.slot.retag("copy")
	return g.Tx.CopyFrom(ctx, tableName, columnNames, rowSrc)
}

func (g *gatingTx) Commit(ctx context.Context) error {
	g.stop.Stop()
	return g.hold.finish(ctx, true)
}

func (g *gatingTx) Rollback(ctx context.Context) error {
	g.stop.Stop()
	return g.hold.finish(ctx, false)
}

type batchHold struct {
	results pgx.BatchResults
	release func()
	once    sync.Once
}

func (h *batchHold) close() error {
	var err error

	h.once.Do(func() {
		defer h.release()
		err = h.results.Close()
	})

	return err
}

type gatingBatch struct {
	pgx.BatchResults
	hold *batchHold
	stop runtime.Cleanup
}

func newGatingBatch(results pgx.BatchResults, release func()) pgx.BatchResults {
	hold := &batchHold{results: results, release: release}
	g := &gatingBatch{BatchResults: results, hold: hold}
	g.stop = runtime.AddCleanup(g, func(h *batchHold) { _ = h.close() }, hold)

	return g
}

func (g *gatingBatch) Close() error {
	g.stop.Stop()
	return g.hold.close()
}

type errRow struct {
	err error
}

func (e errRow) Scan(dest ...any) error { return e.err }

type errBatchResults struct {
	err error
}

func (b errBatchResults) Exec() (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, b.err
}

func (b errBatchResults) Query() (pgx.Rows, error) { return nil, b.err }

func (b errBatchResults) QueryRow() pgx.Row { return errRow(b) }

func (b errBatchResults) Close() error { return b.err }
