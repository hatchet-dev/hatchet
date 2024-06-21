package repository

import (
	"context"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
)

type UpsertWebhookWorkerOpts struct {
	Name       string
	URL        string `validate:"required,url"`
	Secret     string
	TenantId   string `validate:"uuid"`
	Workflows  []string
	TokenValue *string
	TokenID    *string
}

type WebhookWorkerEngineRepository interface {
	// ListWebhookWorkers returns the list of webhook workers for the given tenant
	ListWebhookWorkers(ctx context.Context, tenantId string) ([]*dbsqlc.WebhookWorker, error)

	// UpsertWebhookWorker creates a new webhook worker with the given options
	UpsertWebhookWorker(ctx context.Context, opts *UpsertWebhookWorkerOpts) (*dbsqlc.WebhookWorker, error)
}
