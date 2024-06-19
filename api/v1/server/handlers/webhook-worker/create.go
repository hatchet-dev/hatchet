package webhookworker

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/randstr"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (i *WebhookWorkersService) WebhookCreate(ctx echo.Context, request gen.WebhookCreateRequestObject) (gen.WebhookCreateResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)

	var secret string
	if request.Body.Secret == nil {
		s, err := randstr.GenerateWebhookSecret()
		if err != nil {
			return nil, err
		}
		secret = s
	} else {
		secret = *request.Body.Secret
	}

	var wfs []string
	if request.Body.Workflows != nil {
		wfs = append(wfs, *request.Body.Workflows...)
	}

	ww, err := i.config.APIRepository.WebhookWorker().UpsertWebhookWorker(ctx.Request().Context(), &repository.UpsertWebhookWorkerOpts{
		TenantId:  &tenant.ID,
		Name:      request.Body.Name,
		URL:       request.Body.Url,
		Secret:    secret,
		Workflows: wfs,
	})
	if err != nil {
		return nil, err
	}

	return gen.WebhookCreate200JSONResponse{
		Url:    ww.URL,
		Secret: ww.Secret,
		Name:   ww.Name,
		Metadata: gen.APIResourceMeta{
			Id:        uuid.MustParse(ww.ID),
			CreatedAt: ww.CreatedAt,
			UpdatedAt: ww.UpdatedAt,
		},
	}, nil
}
