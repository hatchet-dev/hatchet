package repository

import (
	"context"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

type WebhookWorkerRepository interface {
	// GetWebhookWorkerByID returns the webhook worker with the given id
	GetWebhookWorkerByID(ctx context.Context, id string) (*dbsqlc.WebhookWorker, error)
}
