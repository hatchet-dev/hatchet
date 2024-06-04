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

func (r *webhookWorkerRepository) UpsertWebhookWorker(ctx context.Context, opts *repository.CreateWebhookWorkerOpts) (*db.WebhookWorkerModel, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	ww, err := r.db.WebhookWorker.UpsertOne(
		db.WebhookWorker.URL.Equals(opts.URL),
	).Create(
		db.WebhookWorker.Secret.Set(opts.Secret),
		db.WebhookWorker.URL.Set(opts.URL),
		db.WebhookWorker.Tenant.Link(
			db.Tenant.ID.Equals(opts.TenantId),
		),
	).Update(
		db.WebhookWorker.Secret.Set(opts.Secret),
		db.WebhookWorker.URL.Set(opts.URL),
	).Exec(ctx)
	if err != nil {
		return nil, err
	}

	var txn []db.PrismaTransaction
	for _, workflow := range opts.Workflows {
		tx := r.db.WebhookWorkerWorkflow.CreateOne(
			db.WebhookWorkerWorkflow.WebhookWorker.Link(
				db.WebhookWorker.ID.Equals(ww.ID),
			),
			db.WebhookWorkerWorkflow.Workflow.Link(
				db.Workflow.ID.Equals(workflow),
			),
		).Tx()
		txn = append(txn, tx)
	}

	if err := r.db.Prisma.Transaction(txn...).Exec(ctx); err != nil {
		return nil, err
	}

	return ww, nil
}
