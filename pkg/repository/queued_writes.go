package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/hatchet-dev/hatchet/pkg/repository/fairpool"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// queuedWrites implements sqlcv1.DBTX so that generated :exec queries can be queued in one batch.
// Nothing is sent until send or sendThenCommit, and it cannot run a query that returns rows.
type queuedWrites struct {
	batch *pgx.Batch
}

var _ sqlcv1.DBTX = (*queuedWrites)(nil)

var errQueuedWritesCannotReturnRows = errors.New("queued writes cannot run a statement that returns rows")

func newQueuedWrites() *queuedWrites {
	return &queuedWrites{batch: &pgx.Batch{}}
}

func (q *queuedWrites) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	q.batch.Queue(sql, args...)

	return pgconn.CommandTag{}, nil
}

func (q *queuedWrites) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errQueuedWritesCannotReturnRows
}

func (q *queuedWrites) QueryRow(context.Context, string, ...any) pgx.Row {
	return failedRow{err: errQueuedWritesCannotReturnRows}
}

func (q *queuedWrites) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, errQueuedWritesCannotReturnRows
}

func (q *queuedWrites) SendBatch(_ context.Context, batch *pgx.Batch) pgx.BatchResults {
	return pgx.FailedBatchResults(batch, errQueuedWritesCannotReturnRows)
}

func (q *queuedWrites) send(ctx context.Context, db sqlcv1.DBTX) error {
	if q.batch.Len() == 0 {
		return nil
	}

	batch := q.batch
	q.batch = &pgx.Batch{}

	return db.SendBatch(ctx, batch).Close()
}

func (q *queuedWrites) sendThenCommit(ctx context.Context, db sqlcv1.DBTX) error {
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

type failedRow struct {
	err error
}

func (r failedRow) Scan(...any) error {
	return r.err
}

// The caller queues "begin" as the first statement of its first batch, commits with
// queuedWrites.sendThenCommit, and must defer the returned function.
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
