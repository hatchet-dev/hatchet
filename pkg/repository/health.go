package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthRepository interface {
	IsHealthy(ctx context.Context) bool
	PgStat() *pgxpool.Stat
}
