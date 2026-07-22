package embed

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func activeFleetSize(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `SELECT count(*) FROM "Dispatcher" WHERE "isActive" = true AND "lastHeartbeatAt" > now() - interval '15 seconds'`).Scan(&n)
	return n, err
}
