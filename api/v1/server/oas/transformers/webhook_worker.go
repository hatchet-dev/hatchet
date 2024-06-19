package transformers

import (
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func ToWebhookWorker(webhookWorker *db.WebhookWorkerModel) *gen.WebhookWorker {
	return &gen.WebhookWorker{
		Metadata: *toAPIMetadata(webhookWorker.ID, webhookWorker.CreatedAt, webhookWorker.UpdatedAt),
		Url:      webhookWorker.URL,
		Secret:   webhookWorker.Secret,
	}
}
