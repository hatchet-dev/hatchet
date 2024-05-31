package prisma

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/validator"
)

type webhookWorkerRepository struct {
	db *db.PrismaClient
	v  validator.Validator
	l  *zerolog.Logger
}

func NewWebhookWorkerRepository(db *db.PrismaClient, v validator.Validator, l *zerolog.Logger) repository.WebhookWorkerRepository {
	return &webhookWorkerRepository{
		db: db,
		v:  v,
		l:  l,
	}
}

func (r *webhookWorkerRepository) ListWebhookWorkers(ctx context.Context, tenantId string) ([]db.WebhookWorkerModel, error) {
	return r.db.WebhookWorker.FindMany(
		db.WebhookWorker.TenantID.Equals(tenantId),
	).Exec(ctx)
}

func (r *webhookWorkerRepository) CreateWebhookWorker(ctx context.Context, opts *repository.CreateWebhookWorkerOpts) (*db.WebhookWorkerModel, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	return r.db.WebhookWorker.CreateOne(
		db.WebhookWorker.Secret.Set(opts.Secret),
		db.WebhookWorker.URL.Set(opts.URL),
		db.WebhookWorker.Tenant.Link(
			db.Tenant.ID.Equals(opts.TenantId),
		),
	).Exec(ctx)
}
