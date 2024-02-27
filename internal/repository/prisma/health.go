package prisma

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
)

type healthRepository struct {
	client  *db.PrismaClient
	queries *dbsqlc.Queries
	pool    *pgxpool.Pool
}

func NewHealthRepository(client *db.PrismaClient, pool *pgxpool.Pool) repository.HealthRepository {
	queries := dbsqlc.New()

	return &healthRepository{
		client:  client,
		queries: queries,
		pool:    pool,
	}
}

func (a *healthRepository) IsHealthy() bool {
	_, err := a.client.User.FindMany().Take(1).Exec(context.Background())
	if err != nil {
		return false
	}

	_, err = a.queries.Health(context.Background(), a.pool)
	if err != nil { //nolint:gosimple
		return false
	}

	return true
}
