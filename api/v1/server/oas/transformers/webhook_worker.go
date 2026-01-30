package transformers

import (
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func ToWebhookWorkerRequest(webhookWorker *sqlcv1.WebhookWorkerRequest) *gen.WebhookWorkerRequest {
	return &gen.WebhookWorkerRequest{
		CreatedAt:  webhookWorker.CreatedAt.Time,
		Method:     webhookWorker.Method,
		StatusCode: int(webhookWorker.StatusCode),
	}
}

func ToWebhookWorker(webhookWorker *sqlcv1.WebhookWorker) *gen.WebhookWorker {
	return &gen.WebhookWorker{
		Metadata: *toAPIMetadata(
			webhookWorker.ID.String(),
			webhookWorker.CreatedAt.Time,
			webhookWorker.UpdatedAt.Time,
		),
		Name: webhookWorker.Name,
		Url:  webhookWorker.Url,
	}
}

func ToWebhookWorkerCreated(webhookWorker *sqlcv1.WebhookWorker) *gen.WebhookWorkerCreated {
	return &gen.WebhookWorkerCreated{
		Metadata: *toAPIMetadata(
			webhookWorker.ID.String(),
			webhookWorker.CreatedAt.Time,
			webhookWorker.UpdatedAt.Time,
		),
		Name:   webhookWorker.Name,
		Url:    webhookWorker.Url,
		Secret: webhookWorker.Secret,
	}
}
