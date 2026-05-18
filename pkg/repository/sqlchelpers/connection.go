package sqlchelpers

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

const defaultStatementTimeoutMs = 30000

func AcquirePoolConnectionWithStatementTimeout(ctx context.Context, pool *pgxpool.Pool, l *zerolog.Logger, timeoutMs int) (*pgxpool.Conn, error) {
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

		return nil, err
	}

	if sinceStart := time.Since(start); sinceStart > 100*time.Millisecond {
		l.Warn().Dur(
			"duration", sinceStart,
		).Int(
			"acquired_connections", int(pool.Stat().AcquiredConns()),
		).Caller(1).Msg("long connection acquire")
	}

	if _, err = conn.Exec(ctx, fmt.Sprintf("SET statement_timeout=%d", timeoutMs)); err != nil {
		ReleasePoolConnectionWithStatementTimeout(conn, l)
		return nil, err
	}

	return conn, nil
}

func ReleasePoolConnectionWithStatementTimeout(conn *pgxpool.Conn, l *zerolog.Logger) {
	resetCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := conn.Exec(resetCtx, fmt.Sprintf("SET statement_timeout=%d", defaultStatementTimeoutMs)); err != nil {
		l.Error().Err(err).Msg("failed to reset statement timeout on released connection")
	}

	conn.Release()
}

func TryAcquireSessionAdvisoryLock(ctx context.Context, conn *pgx.Conn, key int64) (bool, error) {
	var acquired bool
	err := conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1::bigint)", key).Scan(&acquired)
	return acquired, err
}

func ReleasePoolConnectionWithSessionAdvisoryLock(conn *pgxpool.Conn, l *zerolog.Logger, lockKey int64, lockAcquired bool, lockName string) {
	if lockAcquired {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var unlocked bool
		err := conn.QueryRow(unlockCtx, "SELECT pg_advisory_unlock($1::bigint)", lockKey).Scan(&unlocked)
		if err != nil {
			l.Error().Err(err).Int64("lock_key", lockKey).Str("lock_name", lockName).Msg("failed to release session advisory lock; closing connection")

			closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer closeCancel()

			if err := conn.Hijack().Close(closeCtx); err != nil {
				l.Error().Err(err).Int64("lock_key", lockKey).Str("lock_name", lockName).Msg("failed to close connection after advisory lock release failure")
			}

			return
		}

		if !unlocked {
			l.Warn().Int64("lock_key", lockKey).Str("lock_name", lockName).Msg("session advisory lock was not held when releasing connection")
		}
	}

	ReleasePoolConnectionWithStatementTimeout(conn, l)
}
