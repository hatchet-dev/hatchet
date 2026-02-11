package transformers

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func ToV1Webhook(webhook *sqlcv1.V1IncomingWebhook) gen.V1Webhook {
	// Intentionally empty uuid
	var id uuid.UUID

	result := gen.V1Webhook{
		AuthType: gen.V1WebhookAuthType(webhook.AuthMethod),
		Metadata: gen.APIResourceMeta{
			CreatedAt: webhook.InsertedAt.Time,
			UpdatedAt: webhook.UpdatedAt.Time,
			Id:        id.String(),
		},
		TenantId:           webhook.TenantID.String(),
		EventKeyExpression: webhook.EventKeyExpression,
		Name:               webhook.Name,
		SourceName:         gen.V1WebhookSourceName(webhook.SourceName),
	}

	if webhook.ScopeExpression.Valid {
		result.ScopeExpression = &webhook.ScopeExpression.String
	}

	if len(webhook.StaticPayload) > 0 {
		var staticPayload map[string]interface{}
		if err := json.Unmarshal(webhook.StaticPayload, &staticPayload); err != nil {
			log.Error().Err(err).Str("webhook", webhook.Name).Msg("failed to unmarshal static payload")
		}
		result.StaticPayload = &staticPayload
	}

	return result
}

func ToV1WebhookList(webhooks []*sqlcv1.V1IncomingWebhook) gen.V1WebhookList {
	rows := make([]gen.V1Webhook, len(webhooks))

	for i, webhook := range webhooks {
		rows[i] = ToV1Webhook(webhook)
	}

	return gen.V1WebhookList{
		Rows: &rows,
	}
}
