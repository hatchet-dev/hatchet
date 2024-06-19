package repository

import (
	"context"

	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
)

type UpsertWebhookWorkerOpts struct {
	URL        string `validate:"required,url"`
	Secret     string
	TenantId   *string `validate:"uuid"`
	Workflows  []string
	TokenValue *string
	TokenID    *string
}

type WebhookWorkerRepository interface {
	// ListWebhookWorkers returns the list of webhook workers for the given tenant
	ListWebhookWorkers(ctx context.Context, tenantId string) ([]db.WebhookWorkerModel, error)

	// UpsertWebhookWorker creates a new webhook worker with the given options
	UpsertWebhookWorker(ctx context.Context, opts *UpsertWebhookWorkerOpts) (*db.WebhookWorkerModel, error)
}
