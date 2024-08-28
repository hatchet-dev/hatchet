package transformers

import (
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

func ToWebhookWorkerRequest(webhookWorker *dbsqlc.WebhookWorkerRequest) *gen.WebhookWorkerRequest {
	return &gen.WebhookWorkerRequest{
		CreatedAt:  webhookWorker.CreatedAt.Time,
		Method:     webhookWorker.Method,
		StatusCode: int(webhookWorker.StatusCode),
	}
}

func ToWebhookWorker(webhookWorker *dbsqlc.WebhookWorker) *gen.WebhookWorker {
	return &gen.WebhookWorker{
		Metadata: *toAPIMetadata(
			sqlchelpers.UUIDToStr(webhookWorker.ID),
			webhookWorker.CreatedAt.Time,
			webhookWorker.UpdatedAt.Time,
		),
		Name: webhookWorker.Name,
		Url:  webhookWorker.Url,
	}
}

func ToWebhookWorkerCreated(webhookWorker *dbsqlc.WebhookWorker) *gen.WebhookWorkerCreated {
	return &gen.WebhookWorkerCreated{
		Metadata: *toAPIMetadata(
			sqlchelpers.UUIDToStr(webhookWorker.ID),
			webhookWorker.CreatedAt.Time,
			webhookWorker.UpdatedAt.Time,
		),
		Name:   webhookWorker.Name,
		Url:    webhookWorker.Url,
		Secret: webhookWorker.Secret,
	}
}
