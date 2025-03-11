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

func PrepareTx(ctx context.Context, pool *pgxpool.Pool, l *zerolog.Logger, timeoutMs int) (pgx.Tx, func(context.Context) error, func(), error) {
	start := time.Now()

	tx, err := pool.Begin(ctx)

	if err != nil {
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
		_, err = tx.Exec(ctx, "SET statement_timeout=0")

		if err != nil {
			return err
		}

		return tx.Commit(ctx)
	}

	rollback := func() {
		DeferRollback(ctx, l, tx.Rollback)
	}

	// set tx timeout to 5 seconds to avoid deadlocks
	_, err = tx.Exec(ctx, fmt.Sprintf("SET statement_timeout=%d", timeoutMs))

	if err != nil {
		return nil, nil, nil, err
	}

	return tx, commit, rollback, nil
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
