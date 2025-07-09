package webhooksv1

import (
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/labstack/echo/v4"
)

func (t *V1WebhooksService) V1WebhookList(ctx echo.Context, request gen.V1WebhookListRequestObject) (gen.V1WebhookListResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)

	var sourceNames []string
	var webhookNames []string

	if request.Params.SourceNames != nil {
		sourceNames = *request.Params.SourceNames
	}

	if request.Params.WebhookNames != nil {
		webhookNames = *request.Params.WebhookNames
	}

	webhooks, err := t.config.V1.Webhooks().ListWebhooks(
		ctx.Request().Context(),
		tenant.ID.String(),
		v1.ListWebhooksOpts{
			WebhookNames:       webhookNames,
			WebhookSourceNames: sourceNames,
			Limit:              request.Params.Limit,
			Offset:             request.Params.Offset,
		},
	)

	if err != nil {
		return gen.V1WebhookList400JSONResponse(apierrors.NewAPIErrors("failed to list webhooks")), nil
	}

	transformed := transformers.ToV1WebhookList(webhooks)

	return gen.V1WebhookList200JSONResponse(
		transformed,
	), nil
}
