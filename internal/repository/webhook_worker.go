package repository

import (
	"context"

	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
)

type CreateWebhookWorkerOpts struct {
	TenantId  string `validate:"required,uuid"`
	URL       string `validate:"required,url"`
	Secret    string
	Workflows []string `validate:"required,dive,uuid"`
}

type WebhookWorkerRepository interface {
	// ListWebhookWorkers returns the list of webhook workers for the given tenant
	ListWebhookWorkers(ctx context.Context, tenantId string) ([]db.WebhookWorkerModel, error)

	// UpsertWebhookWorker creates a new webhook worker with the given options
	UpsertWebhookWorker(ctx context.Context, opts *CreateWebhookWorkerOpts) (*db.WebhookWorkerModel, error)

	// GetActionNames
	GetActionNames(ctx context.Context, webhookWorkerId string) ([]string, error)
}
