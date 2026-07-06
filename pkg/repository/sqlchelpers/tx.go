package sqlchelpers

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

func PrepareTx(ctx context.Context, pool *pgxpool.Pool, l *zerolog.Logger) (pgx.Tx, func(context.Context) error, func(), error) {
	start := time.Now()

	tx, err := pool.Begin(ctx)

	if err != nil {
		if sinceStart := time.Since(start); sinceStart > 100*time.Millisecond {
			l.Error().Dur(
				"duration", sinceStart,
			).Int(
				"acquired_connections", int(pool.Stat().AcquiredConns()),
			).Caller(1).Msgf("long transaction start with error: %v", err)
		}

		return nil, nil, nil, err
	}

	if sinceStart := time.Since(start); sinceStart > 100*time.Millisecond {
		l.Warn().Dur(
			"duration", sinceStart,
		).Int(
			"acquired_connections", int(pool.Stat().AcquiredConns()),
		).Caller(1).Msg("long transaction start")
	}

	rollback := func() {
		DeferRollback(ctx, l, tx.Rollback)
	}

	return tx, tx.Commit, rollback, nil
}

func PrepareTxWithStatementTimeout(ctx context.Context, pool *pgxpool.Pool, l *zerolog.Logger, timeoutMs int, opts ...pgx.TxOptions) (pgx.Tx, func(context.Context) error, func(), error) {
	start := time.Now()

	txOpts := pgx.TxOptions{}
	if len(opts) > 0 {
		txOpts = opts[0]
	}

	tx, err := pool.BeginTx(ctx, txOpts)

	if err != nil {
		if sinceStart := time.Since(start); sinceStart > 100*time.Millisecond {
			l.Error().Dur(
				"duration", sinceStart,
			).Int(
				"acquired_connections", int(pool.Stat().AcquiredConns()),
			).Caller(1).Msgf("long transaction start with error: %v", err)
		}

		return nil, nil, nil, err
	}

	if sinceStart := time.Since(start); sinceStart > 100*time.Millisecond {
		l.Warn().Dur(
			"duration", sinceStart,
		).Int(
			"acquired_connections", int(pool.Stat().AcquiredConns()),
		).Caller(1).Msg("long transaction start")
	}

	commit := func(ctx context.Context) error {
		// reset statement and idle-in-transaction timeouts
		_, err = tx.Exec(ctx, "SET statement_timeout=30000")

		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, "SET idle_in_transaction_session_timeout=30000")

		if err != nil {
			return err
		}

		return tx.Commit(ctx)
	}

	rollback := func() {
		DeferRollback(ctx, l, tx.Rollback)
	}

	_, err = tx.Exec(ctx, fmt.Sprintf("SET statement_timeout=%d", timeoutMs))

	if err != nil {
		rollback()
		return nil, nil, nil, err
	}

	_, err = tx.Exec(ctx, fmt.Sprintf("SET idle_in_transaction_session_timeout=%d", timeoutMs))

	if err != nil {
		rollback()
		return nil, nil, nil, err
	}

	return tx, commit, rollback, nil
}

// AcquireConnectionWithStatementTimeout acquires a connection from the pool and overwrites the default statement timeout on it.
// It does not support timeout values lower than the default timeout (if called with such a value, it will just use the default timeout).
func AcquireConnectionWithStatementTimeout(ctx context.Context, pool *pgxpool.Pool, l *zerolog.Logger, timeoutMs int) (*pgx.Conn, func(), error) {
	start := time.Now()

	conn, err := pool.Acquire(ctx)

	if err != nil {
		if sinceStart := time.Since(start); sinceStart > 100*time.Millisecond {
			l.Error().Dur(
				"duration", sinceStart,
			).Int(
				"acquired_connections", int(pool.Stat().AcquiredConns()),
			).Caller(1).Msgf("long connection acquire with error: %v", err)
		}

		return nil, nil, err
	}

	if sinceStart := time.Since(start); sinceStart > 100*time.Millisecond {
		l.Warn().Dur(
			"duration", sinceStart,
		).Int(
			"acquired_connections", int(pool.Stat().AcquiredConns()),
		).Caller(1).Msg("long connection acquire")
	}

	release := func() {
		// reset timeouts with a separate ctx; we don't want to use the original ctx here in case it has been cancelled
		resetCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err = conn.Exec(resetCtx, "SET statement_timeout=30000")

		if err != nil {
			l.Error().Err(err).Msg("failed to reset statement timeout on released connection")
		}

		_, err = conn.Exec(resetCtx, "SET idle_in_transaction_session_timeout=30000")

		if err != nil {
			l.Error().Err(err).Msg("failed to reset idle in transaction timeout on released connection")
		}

		conn.Release()
	}

	_, err = conn.Exec(ctx, fmt.Sprintf("SET statement_timeout=%d", timeoutMs))

	if err != nil {
		release()
		return nil, nil, err
	}

	_, err = conn.Exec(ctx, fmt.Sprintf("SET idle_in_transaction_session_timeout=%d", timeoutMs))

	if err != nil {
		release()
		return nil, nil, err
	}

	return conn.Conn(), release, nil
}

func DeferRollback(ctx context.Context, l *zerolog.Logger, rollback func(context.Context) error) {
	// NOTE: rollback must always run, even if the caller's context has been cancelled, because
	// pgxpool only releases the connection back to the pool via Rollback/Commit. Skipping the
	// rollback on a cancelled context permanently leaks the connection from the pool.
	if err := rollback(ctx); err != nil {
		// a cancelled context makes the rollback fail, but pgxpool still destroys and releases
		// the connection, so there's nothing actionable to log
		if ctx.Err() != nil {
			return
		}

		if !errors.Is(err, pgx.ErrTxClosed) {
			l.Error().Err(err).Msg("failed to rollback transaction")

			// TRACE
			trace := debug.Stack()
			l.Error().Str("stack_trace", string(trace)).Msg("Stack trace for rollback failure")
		}
	}
}
