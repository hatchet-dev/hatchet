package repository

import "github.com/jackc/pgx/v5/pgxpool"

type HealthRepository interface {
	IsHealthy() bool
	PgStat() *pgxpool.Stat
}
