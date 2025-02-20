package repository

import (
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

type WebhookWorkerRepository interface {
	// GetWebhookWorkerByID returns the webhook worker with the given id
	GetWebhookWorkerByID(id string) (*db.WebhookWorkerModel, error)
}
