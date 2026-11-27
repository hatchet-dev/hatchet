package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/hatchet-dev/hatchet/pkg/repository/fairpool"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// writeBatch implements sqlcv1.DBTX so that generated :exec queries can be queued in one batch.
// Nothing is sent until send or sendWithCommit, and it cannot run a query that returns rows.
type writeBatch struct {
	batch *pgx.Batch
}

var _ sqlcv1.DBTX = (*writeBatch)(nil)

var errWriteBatchCannotReturnRows = errors.New("a write batch cannot run a statement that returns rows")

func newWriteBatch() *writeBatch {
	return &writeBatch{batch: &pgx.Batch{}}
}

func (q *writeBatch) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	q.batch.Queue(sql, args...)

	return pgconn.CommandTag{}, nil
}

func (q *writeBatch) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errWriteBatchCannotReturnRows
}

func (q *writeBatch) QueryRow(context.Context, string, ...any) pgx.Row {
	return failedRow{err: errWriteBatchCannotReturnRows}
}

func (q *writeBatch) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, errWriteBatchCannotReturnRows
}

func (q *writeBatch) SendBatch(_ context.Context, batch *pgx.Batch) pgx.BatchResults {
	return pgx.FailedBatchResults(batch, errWriteBatchCannotReturnRows)
}

func (q *writeBatch) send(ctx context.Context, db sqlcv1.DBTX) error {
	if q.batch.Len() == 0 {
		return nil
	}

	batch := q.batch
	q.batch = &pgx.Batch{}

	return db.SendBatch(ctx, batch).Close()
}

func (q *writeBatch) sendWithCommit(ctx context.Context, db sqlcv1.DBTX) error {
	batch := q.batch
	q.batch = &pgx.Batch{}

	batch.Queue("commit")

	results := db.SendBatch(ctx, batch)

	for range batch.Len() - 1 {
		if _, err := results.Exec(); err != nil {
			results.Close() // nolint: errcheck
			return err
		}
	}

	commitTag, err := results.Exec()

	if closeErr := results.Close(); err == nil {
		err = closeErr
	}

	if err != nil {
		return err
	}

	// COMMIT of an aborted transaction succeeds and reports ROLLBACK, which pgx.Tx checks for too
	if commitTag.String() == "ROLLBACK" {
		return pgx.ErrTxCommitRollback
	}

	return nil
}

// openBatchedTx hands a transaction opened by a queued "begin" (acquireConnForBatchedTransaction)
// to code that takes a pgx.Tx. It only runs statements: the caller still ends the transaction with
// writeBatch.sendWithCommit or the rollback acquireConnForBatchedTransaction returns, so ending or
// nesting it here is refused.
type openBatchedTx struct {
	conn *fairpool.Conn
}

var _ pgx.Tx = openBatchedTx{}

var errOpenBatchedTxOwnedByCaller = errors.New("a batched transaction is ended by its caller")

func (t openBatchedTx) Begin(context.Context) (pgx.Tx, error) {
	return nil, errOpenBatchedTxOwnedByCaller
}

func (t openBatchedTx) Commit(context.Context) error {
	return errOpenBatchedTxOwnedByCaller
}

func (t openBatchedTx) Rollback(context.Context) error {
	return errOpenBatchedTxOwnedByCaller
}

func (t openBatchedTx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return t.conn.CopyFrom(ctx, tableName, columnNames, rowSrc)
}

func (t openBatchedTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return t.conn.SendBatch(ctx, b)
}

// LargeObjects is only reachable through a real pgx.Tx, and nothing given this type uses them.
func (t openBatchedTx) LargeObjects() pgx.LargeObjects {
	return pgx.LargeObjects{}
}

func (t openBatchedTx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return t.conn.Conn().Prepare(ctx, name, sql)
}

func (t openBatchedTx) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return t.conn.Exec(ctx, sql, arguments...)
}

func (t openBatchedTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return t.conn.Query(ctx, sql, args...)
}

func (t openBatchedTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return t.conn.QueryRow(ctx, sql, args...)
}

func (t openBatchedTx) Conn() *pgx.Conn {
	return t.conn.Conn()
}

type failedRow struct {
	err error
}

func (r failedRow) Scan(...any) error {
	return r.err
}

// The caller queues "begin" as the first statement of its first batch, commits with
// writeBatch.sendWithCommit, and must defer the returned function.
func (s *sharedRepository) acquireConnForBatchedTransaction(ctx context.Context, pool fairpool.Handle) (*fairpool.Conn, func(), error) {
	conn, err := pool.Acquire(ctx)

	if err != nil {
		return nil, nil, err
	}

	rollbackOpenTransactionAndRelease := func() {
		defer conn.Release()

		const idleOutsideTransaction = 'I'

		if conn.Conn().PgConn().TxStatus() == idleOutsideTransaction {
			return
		}

		if _, err := conn.Exec(ctx, "rollback"); err != nil {
			if ctx.Err() == nil {
				s.l.Error().Err(err).Msg("failed to roll back batched transaction")
			}

			// the pool discards a closed connection instead of handing out one with an open transaction
			conn.Conn().Close(context.Background()) // nolint: errcheck
		}
	}

	return conn, rollbackOpenTransactionAndRelease, nil
}
