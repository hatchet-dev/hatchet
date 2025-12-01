package postgres

import (
	"context"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

type webhookWorkerRepository struct {
	*sharedRepository
}

func NewWebhookWorkerRepository(shared *sharedRepository) repository.WebhookWorkerRepository {
	return &webhookWorkerRepository{
		sharedRepository: shared,
	}
}

func (r *webhookWorkerRepository) GetWebhookWorkerByID(ctx context.Context, id string) (*dbsqlc.WebhookWorker, error) {
	return r.queries.GetWebhookWorkerByID(ctx, r.pool, uuid.MustParse(id))
}
