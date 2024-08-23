package repository

import (
	"context"
	"fmt"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
)

type CreateWebhookWorkerOpts struct {
	Name       string
	URL        string `validate:"required,url"`
	Secret     string
	TenantId   string `validate:"uuid"`
	Deleted    *bool
	TokenValue *string
	TokenID    *string
}

type UpdateWebhookWorkerTokenOpts struct {
	TokenValue *string
	TokenID    *string
}

var ErrDuplicateKey = fmt.Errorf("duplicate key error")

type WebhookWorkerEngineRepository interface {
	// ListWebhookWorkersByPartitionId returns the list of webhook workers for a worker partition
	ListWebhookWorkersByPartitionId(ctx context.Context, partitionId string) ([]*dbsqlc.WebhookWorker, error)

	// ListActiveWebhookWorkers returns the list of active webhook workers for the given tenant
	ListActiveWebhookWorkers(ctx context.Context, tenantId string) ([]*dbsqlc.WebhookWorker, error)

	// ListWebhookWorkerRequests returns the list of webhook worker requests for the given webhook worker id
	ListWebhookWorkerRequests(ctx context.Context, webhookWorkerId string) ([]*dbsqlc.WebhookWorkerRequest, error)

	// InsertWebhookWorkerRequest inserts a new webhook worker request with the given options
	InsertWebhookWorkerRequest(ctx context.Context, webhookWorkerId string, method string, statusCode int32) error

	// CreateWebhookWorker creates a new webhook worker with the given options
	CreateWebhookWorker(ctx context.Context, opts *CreateWebhookWorkerOpts) (*dbsqlc.WebhookWorker, error)

	// UpdateWebhookWorkerToken updates a webhook worker with the given id and tenant id
	UpdateWebhookWorkerToken(ctx context.Context, id string, tenantId string, opts *UpdateWebhookWorkerTokenOpts) (*dbsqlc.WebhookWorker, error)

	// SoftDeleteWebhookWorker flags a webhook worker for delete with the given id and tenant id
	SoftDeleteWebhookWorker(ctx context.Context, id string, tenantId string) error

	// HardDeleteWebhookWorker deletes a webhook worker with the given id and tenant id
	HardDeleteWebhookWorker(ctx context.Context, id string, tenantId string) error
}
