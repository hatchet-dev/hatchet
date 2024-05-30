package repository

import (
	"context"

	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
)

type WebhookWorkerEngineRepository interface {
	// GetAllWebhookWorkers gets all webhook workers for a tenant
	GetAllWebhookWorkers(ctx context.Context, tenantId string) ([]*dbsqlc.WebhookWorker, error)
}
