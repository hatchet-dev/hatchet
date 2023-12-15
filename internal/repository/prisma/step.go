package prisma

import (
	"context"

	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/validator"
)

type stepRepository struct {
	client *db.PrismaClient
	v      validator.Validator
}

func NewStepRepository(client *db.PrismaClient, v validator.Validator) repository.StepRepository {
	return &stepRepository{
		client: client,
		v:      v,
	}
}

func (j *stepRepository) ListStepsByActions(tenantId string, actions []string) ([]db.StepModel, error) {
	return j.client.Step.FindMany(
		db.Step.TenantID.Equals(tenantId),
		db.Step.ActionID.In(actions),
	).Exec(context.Background())
}
