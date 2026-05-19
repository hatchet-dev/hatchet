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

func PrepareTxWithStatementTimeout(ctx context.Context, pool *pgxpool.Pool, l *zerolog.Logger, timeoutMs int) (pgx.Tx, func(context.Context) error, func(), error) {
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

	commit := func(ctx context.Context) error {
		// reset statement timeout
		_, err = tx.Exec(ctx, "SET statement_timeout=30000")

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
		// reset statement timeout with a separate ctx; we don't want to use the original ctx here in case it has been cancelled
		resetCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err = conn.Exec(resetCtx, "SET statement_timeout=30000")

		if err != nil {
			l.Error().Err(err).Msg("failed to reset statement timeout on released connection")
		}

		conn.Release()
	}

	_, err = conn.Exec(ctx, fmt.Sprintf("SET statement_timeout=%d", timeoutMs))

	if err != nil {
		release()
		return nil, nil, err
	}

	return conn.Conn(), release, nil
}

func DeferRollback(ctx context.Context, l *zerolog.Logger, rollback func(context.Context) error) {
	if err := rollback(ctx); err != nil {
		if !errors.Is(err, pgx.ErrTxClosed) {
			l.Error().Err(err).Msg("failed to rollback transaction")

			// TRACE
			trace := debug.Stack()
			l.Error().Str("stack_trace", string(trace)).Msg("Stack trace for rollback failure")
		}
	}
}
