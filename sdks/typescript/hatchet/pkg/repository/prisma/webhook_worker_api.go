package prisma

import (
	"context"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type webhookWorkerRepository struct {
	client *db.PrismaClient
	v      validator.Validator
}

func NewWebhookWorkerRepository(client *db.PrismaClient, v validator.Validator) repository.WebhookWorkerRepository {
	return &webhookWorkerRepository{
		client: client,
		v:      v,
	}
}

func (r *webhookWorkerRepository) GetWebhookWorkerByID(id string) (*db.WebhookWorkerModel, error) {
	return r.client.WebhookWorker.FindUnique(
		db.WebhookWorker.ID.Equals(id),
	).Exec(context.Background())
}
