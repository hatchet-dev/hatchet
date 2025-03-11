package postgres

import (
	"context"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
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
	return r.queries.GetWebhookWorkerByID(ctx, r.pool, sqlchelpers.UUIDFromStr(id))
}
