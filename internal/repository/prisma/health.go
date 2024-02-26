package prisma

import (
	"context"

	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
)

type healthRepository struct {
	client *db.PrismaClient
}

func NewHealthRepository(client *db.PrismaClient) repository.HealthRepository {
	return &healthRepository{
		client: client,
	}
}

func (a *healthRepository) IsHealthy() bool {
	_, err := a.client.User.FindMany().Exec(context.Background())
	return err == nil
}
