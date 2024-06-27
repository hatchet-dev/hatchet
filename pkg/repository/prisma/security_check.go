package prisma

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

type securityCheckRepository struct {
	client  *db.PrismaClient
	queries *dbsqlc.Queries
	pool    *pgxpool.Pool
}

func NewSecurityCheckRepository(client *db.PrismaClient, pool *pgxpool.Pool) repository.SecurityCheckRepository {
	queries := dbsqlc.New()

	return &securityCheckRepository{
		client:  client,
		queries: queries,
		pool:    pool,
	}
}

func (a *securityCheckRepository) GetIdent() (string, error) {
	id, err := a.queries.GetSecurityCheckIdent(context.Background(), a.pool)

	if err != nil {
		return "", err
	}

	return sqlchelpers.UUIDToStr(id), nil
}
